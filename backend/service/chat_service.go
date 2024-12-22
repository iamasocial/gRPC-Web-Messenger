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

func (cs *chatService) GetChats(ctx context.Context, req *pb.GetChatsRequst) (*pb.GetChatsResponse, error) {
	userId, ok := ctx.Value(middleware.TokenKey("user_id")).(uint64)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "User ID is missing in context")
	}

	chatIds, err := cs.chatRepo.GetChatIdsByUserId(ctx, userId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get user chats")
	}

	return &pb.GetChatsResponse{
		ChatIds: chatIds,
	}, nil
}

func (cs *chatService) DeleteChat(ctx context.Context, req *pb.DeleteChatRequest) (*pb.DeleteChatResponse, error) {
	userId, ok := ctx.Value(middleware.TokenKey("user_id")).(uint64)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "User ID is missing in context")
	}

	chatId := req.ChatId

	isInChat, err := cs.chatRepo.IsUserInChat(ctx, userId, chatId)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check chat membership: %v", err)
	}

	if !isInChat {
		return nil, status.Errorf(codes.PermissionDenied, "user is not a paticipant of this chat")
	}

	err = cs.chatRepo.DeleteChat(ctx, chatId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete chat: %v", err)
	}

	return &pb.DeleteChatResponse{
		Success: true,
	}, nil
}
