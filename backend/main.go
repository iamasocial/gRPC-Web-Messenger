package main

import (
	"gRPCWebServer/backend/broker"
	"gRPCWebServer/backend/repository"
	"gRPCWebServer/backend/server"
	"gRPCWebServer/backend/service"
	"gRPCWebServer/backend/storage"
	"gRPCWebServer/backend/transport"
	"log"
	"os"
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

	// Создаем базовый каталог для файлов
	baseFilePath := "./storage"
	os.MkdirAll(baseFilePath+"/files", 0755)
	os.MkdirAll(baseFilePath+"/temp", 0755)

	// Инициализируем репозитории
	userRepo := repository.NewUserRepo(db)
	chatRepo := repository.NewChatRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	fileRepo := repository.NewFileRepository(db)
	keyExchangeRepo := repository.NewKeyExchangeRepository(db)

	// Инициализируем сервисы
	userService := service.NewUserService(userRepo)
	chatService := service.NewChatService(chatRepo, userRepo, messageRepo, broker)
	fileService := service.NewFileService(fileRepo, userRepo, chatRepo, baseFilePath)
	keyExchangeService := service.NewKeyExchangeService(keyExchangeRepo, chatRepo, userRepo)

	// Создаем и запускаем сервер
	websocket := transport.NewWebSocketHandler()
	srv := server.NewServer(websocket)
	srv.RegisterServices(userService, chatService, fileService, keyExchangeService)

	if err := srv.Start(":50051", ":8888"); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
