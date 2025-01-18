package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gRPCWebServer/backend/broker"
	"gRPCWebServer/backend/entities"
	pb "gRPCWebServer/backend/generated"
	"gRPCWebServer/backend/middleware"
	"gRPCWebServer/backend/repository"
	"io"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ChatService interface {
	CreateChat(ctx context.Context, req *pb.CreateChatRequest) (*pb.CreateChatResponse, error)
	GetChats(ctx context.Context, req *pb.GetChatsRequst) (*pb.GetChatsResponse, error)
	Chat(stream pb.ChatService_ChatServer) error
}

type chatService struct {
	pb.UnimplementedChatServiceServer
	chatRepo    repository.ChatRepository
	userRepo    repository.UserRepository
	messageRepo repository.MessageRepository
	broker      broker.MessageBroker
	streams     sync.Map
}

// type UserStream struct {
// 	Stream           pb.ChatService_ChatServer
// 	ReceiverUsername string
// }

func NewChatService(chatRepo repository.ChatRepository, userRepo repository.UserRepository, messageRepo repository.MessageRepository, broker broker.MessageBroker) *chatService {
	return &chatService{chatRepo: chatRepo, userRepo: userRepo, messageRepo: messageRepo, broker: broker}
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

func (s *chatService) ConnectToChat(ctx context.Context, req *pb.ConnectRequest) (*pb.ConnectResponse, error) {
	senderId, ok := ctx.Value(middleware.TokenKey("user_id")).(uint64)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "User ID is missing in contenxt")
	}

	receiverUsername := req.GetReceiverUsername()
	if receiverUsername == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Receiver username is required")
	}

	receiver, err := s.userRepo.GetByUsername(ctx, receiverUsername)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "User '%s' does not exists", receiverUsername)
	}

	_, err = s.chatRepo.GetChatByUserIds(ctx, senderId, receiver.ID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Chat with '%s' not found", receiverUsername)
	}

	_, exists := s.streams.LoadOrStore(senderId, receiverUsername)
	if exists {
		return nil, status.Errorf(codes.AlreadyExists, "Chat with '%s' already exists", receiverUsername)
	}

	log.Printf("User %d connected to chat with receiver '%s'", senderId, receiverUsername)

	return &pb.ConnectResponse{
		Success: true,
	}, nil
}

func (s *chatService) Chat(stream pb.ChatService_ChatServer) error {
	senderId, ok := stream.Context().Value(middleware.TokenKey("user_id")).(uint64)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "User ID is missing in context")
	}

	receiverUsername, ok := s.streams.Load(senderId)
	if !ok {
		return status.Errorf(codes.NotFound, "Sender not connected to any chats")
	}

	receiver, _ := s.userRepo.GetByUsername(stream.Context(), receiverUsername.(string))
	senderUsername, _ := s.userRepo.GetUserNameById(stream.Context(), senderId)

	s.streams.Delete(senderId)
	s.streams.Store(senderId, stream)
	defer s.streams.Delete(senderId)
	// s.streams.Swap(senderId, stream)

	chatId, err := s.chatRepo.GetChatByUserIds(stream.Context(), senderId, receiver.ID)
	if err != nil {
		return status.Errorf(codes.Internal, "Error checking chat existance: %v", err)
	}

	log.Printf("User %d started chatting with user %d", senderId, receiver.ID)

	go func() {
		queueName := fmt.Sprintf("chat_queue_%d", senderId)

		handleMessage := func(content string, timestamp time.Time) error {
			err := stream.Send(&pb.ChatResponse{
				SenderUsername: receiverUsername.(string),
				Content:        content,
				Timestamp:      timestamp.Unix(),
			})

			if err != nil {
				log.Printf("Failed to send offline message: %v", err)
			}

			log.Printf("Offline message sent to user %d: %s", senderId, content)
			return nil
		}

		err := s.broker.SubscribeMessages(queueName, handleMessage)
		if err != nil {
			log.Printf("Failed to subscribe to offline messages for user %d: %v", senderId, err)
		}
	}()

	for {
		req, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				log.Printf("Stream closed by user %d", senderId)
				return nil
			}
			return status.Errorf(codes.Internal, "Failed to receiver message: %v", err)
		}

		content := req.GetContent()
		if content == "" {
			return status.Errorf(codes.InvalidArgument, "Message content can't be empty")
		}

		message := entities.Message{
			ChatID:     chatId,
			SenderId:   senderId,
			ReceiverId: receiver.ID,
			Content:    content,
			Timestamp:  time.Now(),
		}

		go func() {
			err = s.messageRepo.SaveMessage(&message)
			if err != nil {
				log.Printf("Failed to save message: %v", err)
			}
		}()

		go func() {
			if recvStream, ok := s.streams.Load(receiver.ID); ok {
				err := recvStream.(pb.ChatService_ChatServer).Send(&pb.ChatResponse{
					SenderUsername: senderUsername,
					Content:        content,
					Timestamp:      message.Timestamp.Unix(),
				})

				if err != nil {
					log.Printf("Couldn't send message to receiver %d: %v", receiver.ID, err)
				} else {
					log.Printf("Message sent from user %d to user %d", senderId, receiver.ID)
				}

			} else {
				log.Printf("Receiver %d not connected, sending message to queue,", receiver.ID)

				err := s.broker.PublishMessage(&message)
				if err != nil {
					log.Printf("Failed to send message to RabbitMQ: %v", err)
				} else {
					log.Printf("Message for user %d queued in RabbitMQ", receiver.ID)
				}
			}
		}()
	}
}
