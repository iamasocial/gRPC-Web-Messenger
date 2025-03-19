package service

import (
	"context"
	"gRPCWebServer/backend/middleware"
	"gRPCWebServer/backend/repository"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "gRPCWebServer/backend/generated"
)

// KeyExchangeService реализует протокол Диффи-Хеллмана для безопасного обмена ключами
type KeyExchangeService struct {
	pb.UnimplementedKeyExchangeServiceServer
	keyExchangeRepo repository.KeyExchangeRepository
	chatRepo        repository.ChatRepository
	userRepo        repository.UserRepository
}

// NewKeyExchangeService создает новый экземпляр сервиса обмена ключами
func NewKeyExchangeService(
	keyExchangeRepo repository.KeyExchangeRepository,
	chatRepo repository.ChatRepository,
	userRepo repository.UserRepository,
) *KeyExchangeService {
	return &KeyExchangeService{
		keyExchangeRepo: keyExchangeRepo,
		chatRepo:        chatRepo,
		userRepo:        userRepo,
	}
}

// InitKeyExchange инициирует процесс обмена ключами
func (s *KeyExchangeService) InitKeyExchange(ctx context.Context, req *pb.InitKeyExchangeRequest) (*pb.InitKeyExchangeResponse, error) {
	// Получаем ID инициатора (текущего пользователя) из контекста
	initiatorID, ok := ctx.Value(middleware.TokenKey("user_id")).(uint64)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "User ID is missing in context")
	}

	// Получаем пользователя-получателя по имени
	receiverUsername := req.GetUsername()
	if receiverUsername == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Receiver username is required")
	}

	receiver, err := s.userRepo.GetByUsername(ctx, receiverUsername)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "User '%s' not found", receiverUsername)
	}

	// Проверяем, существует ли чат между пользователями
	chatID, err := s.chatRepo.GetChatByUserIds(ctx, initiatorID, receiver.ID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Chat with '%s' not found", receiverUsername)
	}

	// Проверяем, существуют ли активные обмены ключами
	existingExchange, err := s.keyExchangeRepo.GetKeyExchangeByChatID(ctx, chatID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to check existing key exchanges: %v", err)
	}

	// Если есть незавершенный обмен ключами и этот пользователь не инициатор, отклоняем запрос
	if existingExchange != nil && existingExchange.Status == "INITIATED" && existingExchange.InitiatorID != initiatorID {
		return nil, status.Errorf(codes.FailedPrecondition, "There is an ongoing key exchange initiated by %s", receiverUsername)
	}

	// Проверяем параметры Диффи-Хеллмана
	if req.GetDhG() == "" || req.GetDhP() == "" || req.GetDhAPublic() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Missing Diffie-Hellman parameters")
	}

	// Создаем новую запись об обмене ключами
	_, err = s.keyExchangeRepo.CreateKeyExchange(
		ctx,
		chatID,
		initiatorID,
		receiver.ID,
		req.GetDhG(),
		req.GetDhP(),
		req.GetDhAPublic(),
	)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create key exchange: %v", err)
	}

	return &pb.InitKeyExchangeResponse{
		Success: true,
	}, nil
}

// CompleteKeyExchange завершает обмен ключами
func (s *KeyExchangeService) CompleteKeyExchange(ctx context.Context, req *pb.CompleteKeyExchangeRequest) (*pb.CompleteKeyExchangeResponse, error) {
	// Получаем ID получателя (текущего пользователя) из контекста
	recipientID, ok := ctx.Value(middleware.TokenKey("user_id")).(uint64)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "User ID is missing in context")
	}

	// Получаем пользователя-инициатора по имени
	initiatorUsername := req.GetUsername()
	if initiatorUsername == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Initiator username is required")
	}

	initiator, err := s.userRepo.GetByUsername(ctx, initiatorUsername)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "User '%s' not found", initiatorUsername)
	}

	// Проверяем, существует ли чат между пользователями
	chatID, err := s.chatRepo.GetChatByUserIds(ctx, recipientID, initiator.ID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Chat with '%s' not found", initiatorUsername)
	}

	// Получаем активный обмен ключами
	exchange, err := s.keyExchangeRepo.GetKeyExchangeByChatID(ctx, chatID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get key exchange: %v", err)
	}

	if exchange == nil {
		return nil, status.Errorf(codes.NotFound, "No active key exchange found")
	}

	// Проверяем, что обмен ключами находится в статусе "INITIATED"
	if exchange.Status != "INITIATED" {
		return nil, status.Errorf(codes.FailedPrecondition, "Key exchange is in %s status and cannot be completed", exchange.Status)
	}

	// Проверяем, что текущий пользователь является получателем
	if exchange.RecipientID != recipientID {
		return nil, status.Errorf(codes.PermissionDenied, "Only the recipient can complete the key exchange")
	}

	// Проверяем параметр публичного ключа B
	if req.GetDhBPublic() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Missing public key B")
	}

	// Обновляем запись обмена ключами с ключом B
	err = s.keyExchangeRepo.CompleteKeyExchange(ctx, exchange.ID, req.GetDhBPublic())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to complete key exchange: %v", err)
	}

	return &pb.CompleteKeyExchangeResponse{
		Success: true,
	}, nil
}

// GetKeyExchangeParams получает параметры обмена ключами
func (s *KeyExchangeService) GetKeyExchangeParams(ctx context.Context, req *pb.GetKeyExchangeParamsRequest) (*pb.GetKeyExchangeParamsResponse, error) {
	// Получаем ID текущего пользователя из контекста
	userID, ok := ctx.Value(middleware.TokenKey("user_id")).(uint64)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "User ID is missing in context")
	}

	// Получаем пользователя-собеседника по имени
	peerUsername := req.GetUsername()
	if peerUsername == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Peer username is required")
	}

	peer, err := s.userRepo.GetByUsername(ctx, peerUsername)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "User '%s' not found", peerUsername)
	}

	// Проверяем, существует ли чат между пользователями
	chatID, err := s.chatRepo.GetChatByUserIds(ctx, userID, peer.ID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Chat with '%s' not found", peerUsername)
	}

	// Получаем активный обмен ключами
	exchange, err := s.keyExchangeRepo.GetKeyExchangeByChatID(ctx, chatID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get key exchange: %v", err)
	}

	response := &pb.GetKeyExchangeParamsResponse{
		Success: exchange != nil,
	}

	if exchange == nil {
		response.Status = pb.KeyExchangeStatus_NOT_STARTED
		return response, nil
	}

	// Преобразуем статус
	switch exchange.Status {
	case "NOT_STARTED":
		response.Status = pb.KeyExchangeStatus_NOT_STARTED
	case "INITIATED":
		response.Status = pb.KeyExchangeStatus_INITIATED
	case "COMPLETED":
		response.Status = pb.KeyExchangeStatus_COMPLETED
	case "FAILED":
		response.Status = pb.KeyExchangeStatus_FAILED
	default:
		response.Status = pb.KeyExchangeStatus_NOT_STARTED
	}

	// Заполняем параметры Диффи-Хеллмана
	if exchange.DHG.Valid {
		response.DhG = exchange.DHG.String
	}

	if exchange.DHP.Valid {
		response.DhP = exchange.DHP.String
	}

	if exchange.DHA.Valid {
		response.DhAPublic = exchange.DHA.String
	}

	if exchange.DHB.Valid {
		response.DhBPublic = exchange.DHB.String
	}

	return response, nil
}
