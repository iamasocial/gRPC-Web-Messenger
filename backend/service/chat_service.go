package service

import (
	"context"
	"database/sql"
	"messenger/backend/middleware"
	"messenger/backend/repository"
	pb "messenger/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ChatService interface {
	CreateChat(ctx context.Context, req *pb.CreateChatRequest)
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
	userId1, ok := ctx.Value(middleware.TokenKey("user_id")).(uint64)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "User ID is missing in context")
	}

	user2, err := cs.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "User %s does not exist", req.Username)
	}
	userId2 := user2.ID

	if userId1 == userId2 {
		return nil, status.Errorf(codes.InvalidArgument, "cannot create chat with yourself")
	}

	chatId, err := cs.chatRepo.GetByUserIds(ctx, userId1, userId2)
	if err != sql.ErrNoRows && err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check existing chat: %v", err)
	}

	if chatId != 0 {
		return nil, status.Errorf(codes.AlreadyExists, "chat with user %s already exists", req.Username)
	}

	chatId, err = cs.chatRepo.CreateChat(ctx, userId1, userId2)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create chat: %v", err)
	}

	return &pb.CreateChatResponse{
		ChatId: chatId,
	}, nil
}
