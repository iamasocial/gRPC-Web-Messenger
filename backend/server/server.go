package server

import (
	"fmt"
	pb "gRPCWebServer/backend/generated"
	"gRPCWebServer/backend/middleware"
	"gRPCWebServer/backend/transport"
	"log"
	"net"
	"net/http"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	grpcServer       *grpc.Server
	webSocketHandler *transport.WebSocketHandler
}

func NewServer(wsHandler *transport.WebSocketHandler) *Server {
	authMiddleWare := middleware.NewAuthInterceptor()
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(authMiddleWare.UnaryInterceptor()),
		grpc.StreamInterceptor(authMiddleWare.StreamInterceptor()),
	)

	return &Server{
		grpcServer:       grpcServer,
		webSocketHandler: wsHandler,
	}
}

func (s *Server) RegisterServices(userService pb.UserServiceServer, chatService pb.ChatServiceServer, fileService pb.FileServiceServer, keyExchangeService pb.KeyExchangeServiceServer) {
	pb.RegisterUserServiceServer(s.grpcServer, userService)
	pb.RegisterChatServiceServer(s.grpcServer, chatService)
	pb.RegisterFileServiceServer(s.grpcServer, fileService)
	pb.RegisterKeyExchangeServiceServer(s.grpcServer, keyExchangeService)
}

func (s *Server) Start(grpcAddr, httpAddr string) error {
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return fmt.Errorf("failed to listen %v", err)
	}

	wrappedGrpc := grpcweb.WrapServer(
		s.grpcServer,
		grpcweb.WithOriginFunc(func(origin string) bool {
			log.Printf("Origin: %s", origin)
			return origin == "http://localhost:8081"
		}),
		grpcweb.WithAllowedRequestHeaders([]string{
			"Content-Type", "Authorization", "X-Request-With", "Accept", "grpc-status", "grpc-message",
		}),
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
	)

	httpServer := &http.Server{
		Addr: httpAddr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Received request: %s %s", r.Method, r.URL.Path)

			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, grpc-status, grpc-message, grpc-web, x-grpc-web, x-user-agent")
			w.Header().Set("Access-Control-Expose-Headers", "grpc-status, grpc-message")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			if r.Method == http.MethodOptions {
				log.Printf("Request method = OPTIONS")
				w.WriteHeader(http.StatusNoContent)
				return
			}

			if r.URL.Path == "/ws" {
				log.Printf("Handling WebSocket request for %s", r.URL.Path)
				s.webSocketHandler.Handle(w, r)
				return
			}

			if wrappedGrpc.IsGrpcWebRequest(r) {
				log.Printf("Handling gRPC-Web request for %s", r.URL.Path)
				wrappedGrpc.ServeHTTP(w, r)
				return
			}

			log.Printf("Status not found")
			w.WriteHeader(http.StatusNotFound)

		}),
	}

	reflection.Register(s.grpcServer)
	go func() {
		log.Printf("Starting gRPC server on %s", grpcAddr)
		if err := s.grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	log.Printf("Starting gRPC-Web and WebSocket server on %s", httpAddr)
	return httpServer.ListenAndServe()
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
	log.Println("Server stopped")
}
