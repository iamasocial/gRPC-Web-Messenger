package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gRPCWebServer/backend/broker"
	"gRPCWebServer/backend/entities"
	pb "gRPCWebServer/backend/generated"
	"gRPCWebServer/backend/manager"
	"gRPCWebServer/backend/middleware"
	"gRPCWebServer/backend/repository"
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

type MessageStreamProvider interface {
	GetReceiveChannel(username string) (chan *pb.ReceiveMessagesResponse, error)
}

type chatService struct {
	pb.UnimplementedChatServiceServer
	chatRepo      repository.ChatRepository
	userRepo      repository.UserRepository
	messageRepo   repository.MessageRepository
	broker        broker.MessageBroker
	streamManager manager.StreamManager3
}

func NewChatService(
	chatRepo repository.ChatRepository,
	userRepo repository.UserRepository,
	messageRepo repository.MessageRepository,
	broker broker.MessageBroker,
) *chatService {
	return &chatService{
		chatRepo:      chatRepo,
		userRepo:      userRepo,
		messageRepo:   messageRepo,
		broker:        broker,
		streamManager: manager.NewStreamManager3(),
	}
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

	s.streamManager.ClearConnectionAndStream(senderId)
	log.Printf("Connection and stream are cleared")

	receiverUsername := req.GetReceiverusername()
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

	err = s.streamManager.AddConnection(senderId, receiver.ID)
	if err != nil {
		return nil, status.Errorf(codes.AlreadyExists, "Chat with senderID=%d and receiverID=%d already exists", senderId, receiver.ID)
	}

	log.Printf("User %d connected to chat with receiver '%s'", senderId, receiverUsername)

	return &pb.ConnectResponse{
		Success: true,
	}, nil
}

///////////////////////////////////// grpc bidi stream doesn't work in browser :(
// do not delete

func (s *chatService) Chat(stream pb.ChatService_ChatServer) error {
	ctx := stream.Context()
	senderId, ok := ctx.Value(middleware.TokenKey("user_id")).(uint64)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "user ID is missing in context")
	}

	receiverId, err := s.streamManager.GetConnectionData(senderId)
	if err != nil {
		return status.Errorf(codes.NotFound, "connection error: %v", err)
	}

	senderUsername, err := s.userRepo.GetUserNameById(ctx, senderId)
	if err != nil {
		return status.Errorf(codes.NotFound, "failed to get sender username: %v", err)
	}

	receiverUsername, err := s.userRepo.GetUserNameById(ctx, receiverId)
	if err != nil {
		return status.Errorf(codes.NotFound, "failed to get receiver username: %v", err)
	}

	s.streamManager.AddStream(senderId, stream)
	defer s.streamManager.ClearConnectionAndStream(senderId)

	chatID, err := s.chatRepo.GetChatByUserIds(ctx, senderId, receiverId)
	if err != nil {
		return status.Errorf(codes.Internal, "error checking chat existence: %v", err)
	}

	log.Printf("User %d started chatting with user %d", senderId, receiverId)

	messages, err := s.getHistory(stream.Context(), chatID)
	if err != nil {
		return status.Errorf(codes.Internal, "%v", err)
	}

	for _, message := range messages {
		senderUsername, err := s.userRepo.GetUserNameById(stream.Context(), message.SenderId)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to get sender username from message")
		}

		resp := &pb.ChatResponse{
			Senderusername: senderUsername,
			Content:        message.Content,
			Timestamp:      message.Timestamp.Unix(),
		}

		err = stream.Send(resp)
		if err != nil {
			log.Printf("Failed to send message from history to user")
			return status.Error(codes.Internal, "failed to send history message")
		}
	}

	log.Printf("finished loading history")

	handleMessage := func(senderUsername, content string, timestamp time.Time) error {

		resp := &pb.ChatResponse{
			Senderusername: senderUsername,
			Content:        content,
			Timestamp:      timestamp.Unix(),
		}

		if err := stream.Send(resp); err != nil {
			log.Printf("Failed to send message: %v", err)

			return fmt.Errorf("stream.Send failed: %v", err)
		}

		log.Printf("Message sent to client: %s", content)
		return nil
	}

	func() {
		defer log.Printf("Offline message processor for user %d stopped", senderId)

		receiverQueue := fmt.Sprintf("chat_queue_%s", senderUsername)

		hasMessages, err := s.broker.CheckMessages(receiverQueue)
		if err != nil {
			log.Printf("Error checking messages for queue %s", receiverQueue)
			return
		}

		if !hasMessages {
			log.Printf("User %s has no messages", senderUsername)
			return
		}

		if err := s.broker.ProcessMessages(receiverQueue, handleMessage); err != nil {
			log.Printf("Error checking queue %s messages", receiverQueue)
		}
	}()

	messageWG := &sync.WaitGroup{}
	defer messageWG.Wait()

	for {
		select {
		case <-ctx.Done():
			s.streamManager.ClearConnectionAndStream(senderId)
			log.Printf("Chat session for userID=%d ended", senderId)
			return nil

		default:
			req, err := stream.Recv()
			if err != nil {
				if errors.Is(ctx.Err(), context.Canceled) {
					s.streamManager.ClearConnectionAndStream(senderId)
					log.Printf("Stream closed by user %d", senderId)
					return nil
				}
				return status.Errorf(codes.Internal, "failed to receive message: %v", err)
			}

			content := req.GetContent()
			if content == "" {
				return status.Errorf(codes.InvalidArgument, "message content cannot be empty")
			}

			message := &entities.Message{
				ChatID:     chatID,
				SenderId:   senderId,
				ReceiverId: receiverId,
				Content:    content,
				Timestamp:  time.Now(),
			}

			messageWG.Add(1)
			// need to refactor
			go func() {
				defer messageWG.Done()
				if err := s.messageRepo.SaveMessage(message); err != nil {
					log.Printf("Failed to save message: %v", err)
				}
			}()

			messageWG.Add(1)
			go func() {
				defer messageWG.Done()
				recvStream, err := s.streamManager.GetStream(receiverId)
				if err == nil {
					if err := recvStream.Send(&pb.ChatResponse{
						Senderusername: senderUsername,
						Content:        content,
						Timestamp:      message.Timestamp.Unix(),
					}); err != nil {
						log.Printf("Failed to send message to receiver %d: %v", receiverId, err)
						s.broker.PublishMessage(senderUsername, receiverUsername, content, message.Timestamp)
					}
				} else {
					if err := s.broker.PublishMessage(senderUsername, receiverUsername, content, message.Timestamp); err != nil {
						log.Printf("Failed to publish message to queue: %v", err)
					}
				}
			}()
		}
	}
}

func (s *chatService) getHistory(ctx context.Context, chatId uint64) ([]entities.Message, error) {
	limit := 100
	messages, err := s.messageRepo.GetHistory(ctx, chatId, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get last 100 messages: %v", err)
	}

	return messages, nil
}
