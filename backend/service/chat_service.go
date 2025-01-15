package service

import (
	"context"
	"database/sql"
	"errors"
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

// middleware issue
// func (s *chatService) Chat(stream pb.ChatService_ChatServer) error {
// 	senderId, ok := stream.Context().Value(middleware.TokenKey("user_id")).(uint64)
// 	if !ok {
// 		return status.Errorf(codes.Unauthenticated, "user ID is missing in contenxt")
// 	}

// 	s.streams.Store(senderId, stream)
// 	defer s.streams.Delete(senderId)

// 	log.Printf("User %d connected to chat", senderId)

// 	errCh := make(chan error, 1)

// 	queueName := fmt.Sprintf("chat_queue_%d", senderId)
// 	go func() {
// 		errCh <- s.broker.SubscribeMessages(queueName, func(msg string) error {
// 			err := stream.Send(&pb.ChatMessage{
// 				ReceiverUsername: "",
// 				Content:          msg,
// 				Timestamp:        time.Now().Unix(),
// 			})
// 			if err != nil {
// 				log.Printf("failed to send message to user '%d': %v", senderId, err)
// 			}
// 			return nil
// 		})

// 		// if err != nil {
// 		// 	log.Printf("error subscribing to messages for user '%d': %v", senderId, err)
// 		// }
// 	}()

// 	for {
// 		select {
// 		case err := <-errCh:
// 			if err != nil {
// 				log.Printf("Subscription error for user %d: %v", senderId, err)
// 				return status.Errorf(codes.Internal, "Failed to subscribe to messages: %v", err)
// 			}

// 		default:
// 			req, err := stream.Recv()
// 			if err != nil {
// 				if err == io.EOF {
// 					log.Printf("Stream closed by user %d", senderId)
// 					return nil
// 				}
// 				return status.Errorf(codes.Internal, "Failed to receive message: %v", err)
// 			}

// 			content := req.GetContent()
// 			if content == "" {
// 				return status.Errorf(codes.InvalidArgument, "content can't be emtpy")
// 			}

// 			receiver, err := s.userRepo.GetByUsername(stream.Context(), req.GetReceiverUsername())
// 			if err != nil || receiver == nil {
// 				return status.Errorf(codes.NotFound, "Receiver not found")
// 			}

// 			chatId, err := s.chatRepo.GetChatByUserIds(stream.Context(), senderId, receiver.ID)
// 			if err != nil {
// 				return status.Errorf(codes.Internal, "Error checking chat existence: %v", err)
// 			}

// 			message := entities.Message{
// 				ChatID:     chatId,
// 				SenderId:   senderId,
// 				ReceiverId: receiver.ID,
// 				Content:    content,
// 				Timestamp:  time.Now(),
// 			}

// 			err = s.messageRepo.SaveMessage(&message)
// 			if err != nil {
// 				return status.Errorf(codes.Internal, "Failed to save message: %v", err)
// 			}

// 			err = s.broker.PublishMessage(&message)
// 			if err != nil {
// 				return status.Errorf(codes.Internal, "Failed to publish message to broker: %v", err)
// 			}

// 			if recvStream, ok := s.streams.Load(receiver.ID); ok {
// 				recvStream := recvStream.(pb.ChatService_ChatServer)

// 				err = recvStream.Send(&pb.ChatMessage{
// 					ReceiverUsername: req.GetReceiverUsername(),
// 					Content:          req.GetContent(),
// 					Timestamp:        time.Now().Unix(),
// 				})
// 				if err != nil {
// 					log.Printf("Couldn't send message to receiver %d: %v", receiver.ID, err)
// 				}
// 			} else {
// 				log.Printf("Receiver %d is not connected, message sent to queue", receiver.ID)
// 			}
// 		}
// 	}
// 	// for {
// 	// 	req, err := stream.Recv()
// 	// 	if err != nil {
// 	// 		if err == io.EOF {
// 	// 			log.Printf("Stream closed by user %d", senderId)
// 	// 			return nil
// 	// 		}
// 	// 	}

// 	// 	content := req.GetContent()
// 	// 	if content == "" {
// 	// 		return status.Errorf(codes.InvalidArgument, "content can't be empty")
// 	// 	}

// 	// 	receiver, err := s.userRepo.GetByUsername(stream.Context(), req.GetReceiverUsername())
// 	// 	if err != nil || receiver == nil {
// 	// 		return status.Errorf(codes.NotFound, "Receiver not found")
// 	// 	}

// 	// 	chatId, err := s.chatRepo.GetChatByUserIds(stream.Context(), senderId, receiver.ID)
// 	// 	if err != nil {
// 	// 		return status.Errorf(codes.Internal, "Error checking existence: %v", err)
// 	// 	}

// 	// 	message := entities.Message{
// 	// 		ChatID:     chatId,
// 	// 		SenderId:   senderId,
// 	// 		ReceiverId: receiver.ID,
// 	// 		Content:    content,
// 	// 		Timestamp:  time.Now(),
// 	// 	}

// 	// 	err = s.messageRepo.SaveMessage(&message)
// 	// 	if err != nil {
// 	// 		return status.Errorf(codes.Internal, "Failed to save message: %v", err)
// 	// 	}

// 	// 	err = s.broker.PublishMessage(&message)
// 	// 	if err != nil {
// 	// 		return status.Errorf(codes.Internal, "couldn't send message to broker")
// 	// 	}

// 	// 	if recvStream, ok := s.streams.Load(receiver.ID); ok {
// 	// 		recvStream := recvStream.(pb.ChatService_ChatServer)

// 	// 		err = recvStream.Send(&pb.ChatMessage{
// 	// 			ReceiverUsername: req.GetReceiverUsername(),
// 	// 			Content:          req.GetContent(),
// 	// 			Timestamp:        req.GetTimestamp(),
// 	// 		})
// 	// 		if err != nil {
// 	// 			log.Printf("Couldn't send message to receiver %d: %v", receiver.ID, err)
// 	// 		}
// 	// 	} else {
// 	// 		log.Printf("Receiver %d is not connected", receiver.ID)
// 	// 	}
// 	// }
// }

func (s *chatService) Chat(stream pb.ChatService_ChatServer) error {
	senderId, ok := stream.Context().Value(middleware.TokenKey("user_id")).(uint64)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "User ID is missing in contenxt")
	}

	s.streams.Store(senderId, stream)
	defer s.streams.Delete(senderId)

	log.Printf("User %d connected to chat", senderId)

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

		receiver, err := s.userRepo.GetByUsername(stream.Context(), req.GetReceiverUsername())
		if err != nil || receiver == nil {
			return status.Errorf(codes.NotFound, "Receiver not found")
		}

		chatId, err := s.chatRepo.GetChatByUserIds(stream.Context(), senderId, receiver.ID)
		if err != nil {
			return status.Errorf(codes.Internal, "Error checking chat existance: %v", err)
		}

		message := entities.Message{
			ChatID:     chatId,
			SenderId:   senderId,
			ReceiverId: receiver.ID,
			Content:    content,
			Timestamp:  time.Now(),
		}

		err = s.messageRepo.SaveMessage(&message)
		if err != nil {
			return status.Errorf(codes.Internal, "Failed to save message: %v", err)
		}

		if recvStream, ok := s.streams.Load(receiver.ID); ok {
			recvStream := recvStream.(pb.ChatService_ChatServer)

			err = recvStream.Send(&pb.ChatResponse{
				SenderUsername: req.GetReceiverUsername(),
				Content:        content,
				Timestamp:      time.Now().Unix(),
			})

			if err != nil {
				log.Printf("Coundn't send message to receiver %d: %v", receiver.ID, err)
			} else {
				log.Printf("Message sent from user %d to user %d", senderId, receiver.ID)
			}
		} else {
			log.Printf("Receiver %d is not connected. Message will not be delivered in real-time.", receiver.ID)
		}
	}
}
