package server

import (
	"fmt"
	"log"
	"messenger/backend/middleware"
	"messenger/proto"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	grpcServer *grpc.Server
}

func NewServer() *Server {
	authMiddleWare := middleware.NewAuthInterceptor()
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(authMiddleWare.UnaryInterceptor()),
	)
	return &Server{
		grpcServer: grpcServer,
	}
}

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
