package service

import (
	"context"
	"database/sql"
	"errors"
	"messenger/backend/middleware"
	"messenger/backend/repository"
	pb "messenger/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ChatService interface {
	CreateChat(ctx context.Context, req *pb.CreateChatRequest) (*pb.CreateChatResponse, error)
	GetChats(ctx context.Context, req *pb.GetChatsRequst) (*pb.GetChatsResponse, error)
}

type chatService struct {
	pb.UnimplementedChatServiceServer
	chatRepo repository.ChatRepository
	userRepo repository.UserRepository
}

func NewChatService(chatRepo repository.ChatRepository, userRepo repository.UserRepository) *chatService {
	return &chatService{chatRepo: chatRepo, userRepo: userRepo}
}

func (cs *chatService) CreateChat(ctx context.Context, req *pb.CreateChatRequest) (*pb.CreateChatResponse, error) {
	userId, ok := ctx.Value(middleware.TokenKey("user_id")).(uint64)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "User ID is missing in context")
	}

	targetUsername := req.Username
	targerUser, err := cs.userRepo.GetByUsername(ctx, targetUsername)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user '%s' not found", targetUsername)
	}

	if targerUser.ID == userId {
		return nil, status.Errorf(codes.InvalidArgument, "cannot create a chat with yourself")
	}

	chatId, err := cs.chatRepo.GetChatByUserIds(ctx, userId, targerUser.ID)
	if err != sql.ErrNoRows && err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check existing chat: %v", err)
	}

	if chatId != 0 {
		return nil, status.Errorf(codes.AlreadyExists, "chat with user '%s' already exists", req.Username)
	}

	err = cs.chatRepo.CreateChat(ctx, userId, targerUser.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create chat: %v", err)
	}

	return &pb.CreateChatResponse{
		Username: targetUsername,
	}, nil
}

func (cs *chatService) GetChats(ctx context.Context, req *pb.GetChatsRequst) (*pb.GetChatsResponse, error) {
	userId, ok := ctx.Value(middleware.TokenKey("user_id")).(uint64)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "User ID is missing in context")
	}

	usernames, err := cs.chatRepo.GetChatsByUserId(ctx, userId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch chats: %v", err)
	}

	return &pb.GetChatsResponse{
		Usernames: usernames,
	}, nil
}

func (cs *chatService) DeleteChat(ctx context.Context, req *pb.DeleteChatRequest) (*pb.DeleteChatResponse, error) {
	userId, ok := ctx.Value(middleware.TokenKey("user_id")).(uint64)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "User ID is missing in context")
	}

	targerUser, err := cs.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user '%s' not found", req.Username)
	}

	chatId, err := cs.chatRepo.GetChatByUserIds(ctx, userId, targerUser.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "chat with '%s' not found", req.Username)
		}
		return nil, status.Errorf(codes.Internal, "failed to retrieve chat: %v", err)
	}

	err = cs.chatRepo.DeleteChat(ctx, chatId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete chat: %v", err)
	}

	return &pb.DeleteChatResponse{
		Success: true,
	}, nil
}
