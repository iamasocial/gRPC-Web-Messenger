package service

import (
	"context"
	"messenger/backend/middleware"
	"messenger/backend/repository"
	"messenger/backend/utils"
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
	token, ok := ctx.Value(middleware.TokenKey("token")).(string)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "authentication token not found")
	}

	claims, err := utils.ValidateToken(token)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid authentication token: %v", err)
	}

	tokenUserId := claims.UserId
	if tokenUserId == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "user_id not found in token")
	}

	targetUsername := req.Username
	targetUser, err := cs.userRepo.GetByUsername(ctx, targetUsername)
	if err != nil || targetUser == nil {
		return nil, status.Errorf(codes.NotFound, "user with username '%s' not found", targetUsername)
	}

	if targetUser.ID == tokenUserId {
		return nil, status.Errorf(codes.InvalidArgument, "cannot create a chat with yourself")
	}

	chatId, err := cs.chatRepo.CreateChat(ctx, tokenUserId, targetUser.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create chat: %v", err)
	}

	return &pb.CreateChatResponse{
		ChatId: chatId,
	}, nil
}
