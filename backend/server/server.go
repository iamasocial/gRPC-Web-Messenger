package server

import (
	"fmt"
	"log"
	"messenger/backend/middleware"
	"messenger/backend/repository"
	"messenger/proto"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	grpcServer *grpc.Server
}

func NewServer(tr repository.TokenRepository) *Server {
	// excludeMethods := []string{
	// 	"/messenger.UserService/Login",
	// 	"/messenger.UserService/Register",
	// }
	authMiddleWare := middleware.NewAuthInterceptor(tr)
	grpcServer := grpc.NewServer(
		// grpc.UnaryInterceptor(middleware.AuthInterceptor(excludeMethods)),
		grpc.UnaryInterceptor(authMiddleWare.UnaryInterceptor()),
	)
	return &Server{
		grpcServer: grpcServer,
	}
}

// func NewServer() *Server {
// 	return &Server{
// 		grpcServer: grpc.NewServer(),
// 	}
// }

func (s *Server) RegisterServices(userService proto.UserServiceServer, chatService proto.ChatServiceServer) {
	proto.RegisterUserServiceServer(s.grpcServer, userService)
	proto.RegisterChatServiceServer(s.grpcServer, chatService)
}

func (s *Server) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen %v", err)
	}
	reflection.Register(s.grpcServer)
	log.Printf("Server listening on %s", addr)
	return s.grpcServer.Serve(lis)
}

func (s *Server) Stop() {
	s.grpcServer.Stop()
	log.Println("Server stopped")
}

// package main

// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	pb "messenger/proto"
// 	"net"
// 	"time"

// 	"google.golang.org/grpc"
// 	"google.golang.org/grpc/reflection"
// )

// type server struct {
// 	pb.UnimplementedMessengerServer
// }

// func (s *server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
// 	userID := fmt.Sprintf("user-%d", time.Now().UnixNano())
// 	fmt.Println("Registering user:", req.GetUsername())
// 	return &pb.RegisterResponse{UserId: userID}, nil
// }

// func (s *server) CreateChat(ctx context.Context, req *pb.CreateChatRequest) (*pb.CreateChatResponse, error) {
// 	chatID := fmt.Sprintf("chat-%d", time.Now().UnixNano())
// 	return &pb.CreateChatResponse{ChatId: chatID}, nil
// }

// func (s *server) JoinChat(ctx context.Context, req *pb.JoinChatRequest) (*pb.JoinChatResponse, error) {
// 	return &pb.JoinChatResponse{ChatId: req.ChatId, UserId: req.UserId}, nil
// }

// func (s *server) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
// 	messageID := fmt.Sprintf("msg-%d", time.Now().UnixNano())
// 	return &pb.SendMessageResponse{MessageId: messageID}, nil
// }

// func (s *server) ReceiveMessages(req *pb.ReceiveMessagesRequest, stream pb.Messenger_ReceiveMessagesServer) error {
// 	for {
// 		message := &pb.Message{
// 			UserId:    "user-1",
// 			Content:   "Hello, world!",
// 			Timestamp: time.Now().Format(time.RFC3339),
// 		}

// 		if err := stream.Send(message); err != nil {
// 			return err
// 		}
// 		time.Sleep(2 * time.Second)
// 	}
// }

// func main() {
// 	lis, err := net.Listen("tcp", ":50051")
// 	if err != nil {
// 		log.Fatalf("failed to listen: %v", err)
// 	}

// 	grpcServer := grpc.NewServer()

// 	pb.RegisterMessengerServer(grpcServer, &server{})

// 	reflection.Register(grpcServer)

// 	log.Println("Server is running at :50051")
// 	if err := grpcServer.Serve(lis); err != nil {
// 		log.Fatalf("failed to serve: %v", err)
// 	}
// }
