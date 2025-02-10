package main

import (
	"gRPCWebServer/backend/broker"
	"gRPCWebServer/backend/manager"
	"gRPCWebServer/backend/repository"
	"gRPCWebServer/backend/server"
	"gRPCWebServer/backend/service"
	"gRPCWebServer/backend/storage"
	"log"
)

func main() {
	db, err := storage.ConnectDB(storage.Config{
		Host:     "db",
		Port:     5432,
		User:     "admin",
		Password: "topsecret",
		DBName:   "messenger_db",
		SSLMode:  "disable",
	})
	if err != nil {
		log.Fatal(err)
	}

	broker, err := broker.NewMessageBroker("amqp://admin:topsecret@rabbitmq:5672/")
	if err != nil {
		log.Fatal(err)
	}

	streamManager := manager.NewStreamManager()

	userRepo := repository.NewUserRepo(db)
	chatRepo := repository.NewChatRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	userService := service.NewUserService(userRepo)
	chatService := service.NewChatService(chatRepo, userRepo, messageRepo, broker, streamManager)
	srv := server.NewServer()

	srv.RegisterServices(userService, chatService)

	if err := srv.Start(":50051", ":8888"); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
