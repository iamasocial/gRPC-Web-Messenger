package service

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
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
	repo      repository.UserRepository
	tokenRepo repository.TokenRepository
}

func NewUserService(repo repository.UserRepository, tokenRepo repository.TokenRepository) *UserService {
	return &UserService{repo: repo, tokenRepo: tokenRepo}
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

	// passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
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

	err = us.tokenRepo.SaveToken(ctx, token)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save token: %v", err)
	}

	return &pb.RegisterResponse{
		Token: token.Token,
	}, nil
}

// func (us *UserService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
// 	user, err := us.repo.GetByUsername(ctx, req.Username)
// 	if err != nil || user == nil {
// 		return nil, errors.New("invalid username or password")
// 	}

// 	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
// 	if err != nil {
// 		return nil, errors.New("invalid username or password")
// 	}

// 	return &pb.LoginResponse{
// 		UserId: user.ID,
// 	}, nil
// }

func (us *UserService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	user, err := us.repo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errors.New("user not found")
	}

	existingToken, err := us.tokenRepo.GetActiveToken(ctx, user.ID)
	if err == nil && existingToken != nil {
		return nil, errors.New("user is already logged in")
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
	log.Println("token generated succussfully")

	err = us.tokenRepo.SaveToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to save token: %v", err)
	}

	return &pb.LoginResponse{Token: token.Token}, nil
}

func (us *UserService) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	tokenString, ok := ctx.Value(middleware.TokenKey("token")).(string)
	if !ok || tokenString == "" {
		return nil, status.Errorf(codes.Unauthenticated, "Token is missing in context")
	}

	token, err := us.tokenRepo.GetByToken(ctx, tokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to find token: %v", err)
	}

	if token == nil {
		return nil, errors.New("token not found")
	}

	err = us.tokenRepo.DeleteToken(ctx, token.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to delete token: %v", err)
	}

	return &pb.LogoutResponse{
		Success: true,
	}, nil
}
