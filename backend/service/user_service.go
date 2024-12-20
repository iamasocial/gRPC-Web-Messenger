package service

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"messenger/backend/entities"
	"messenger/backend/middleware"
	"messenger/backend/repository"
	"messenger/backend/utils"
	pb "messenger/proto"
	"strings"

	"golang.org/x/crypto/argon2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserService struct {
	pb.UnimplementedUserServiceServer
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (us *UserService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	existingUser, err := us.repo.GetByUsername(ctx, req.Username)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			existingUser = nil
		} else {
			return nil, fmt.Errorf("failed to check existing user: %v", err)
		}
	}

	if existingUser != nil {
		return nil, errors.New("username already taken")
	}

	if req.Password != req.ConfirmPassword {
		return nil, errors.New("password must match")
	}

	passwordHash, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %v", err)
	}

	user := entities.User{
		Username:     req.Username,
		PasswordHash: passwordHash,
	}

	userID, err := us.repo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	token, err := utils.GenerateToken(userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate token: %v", err)
	}

	return &pb.RegisterResponse{
		Token: token,
	}, nil
}

func (us *UserService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	user, err := us.repo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errors.New("user not found")
	}

	parts := strings.Split(user.PasswordHash, ":")
	if len(parts) != 2 {
		return nil, errors.New("invalid hash format")
	}

	salt := parts[0]
	storedHash := parts[1]

	hash := argon2.IDKey([]byte(req.Password), []byte(salt), 1, 64*1024, 1, 32) // вынести константы в .env
	hashStr := base64.StdEncoding.EncodeToString(hash)

	if subtle.ConstantTimeCompare([]byte(storedHash), []byte(hashStr)) != 1 {
		return nil, errors.New("incorrect password") // wrong username or password
	}

	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %v", err)
	}

	return &pb.LoginResponse{Token: token}, nil
}

func (us *UserService) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	_, ok := ctx.Value(middleware.TokenKey("user_id")).(uint64)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "User ID is missing in context")
	}

	return &pb.LogoutResponse{
		Success: true,
	}, nil
}
