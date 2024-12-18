package main

import (
	"log"
	"messenger/backend/repository"
	"messenger/backend/server"
	"messenger/backend/service"
	"messenger/backend/storage"
)

func main() {
	db, err := storage.ConnectDB(storage.Config{
		Host:     "localhost",
		Port:     5432,
		User:     "admin",
		Password: "topsecret",
		DBName:   "messenger_db",
		SSLMode:  "disable",
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	userRepo := repository.NewUserRepo(db)
	tokenRepo := repository.NewTokenRepository(db)
	userService := service.NewUserService(userRepo, tokenRepo)
	srv := server.NewServer(tokenRepo)

	// userService := service.NewUserService()
	chatService := service.NewChatService()
	srv.RegisterServices(userService, chatService)

	if err := srv.Start(":50051"); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
