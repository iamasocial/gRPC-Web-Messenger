package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
	"sync"
	"time"

	pb "gRPCWebServer/backend/generated"
)

// KeyExchangeService реализует протокол Диффи-Хеллмана для безопасного обмена ключами
type KeyExchangeService struct {
	pb.UnimplementedKeyExchangeServiceServer
	mu sync.RWMutex

	// Хранилище активных сессий обмена ключами
	sessions map[string]*keyExchangeSession
}

// keyExchangeSession хранит информацию о сессии обмена ключами
type keyExchangeSession struct {
	// Приватный ключ сервера (a)
	serverPrivateKey *big.Int
	// Публичный ключ сервера (A = g^a mod p)
	serverPublicKey *big.Int
	// Время создания сессии
	createdAt time.Time
	// Время жизни сессии (24 часа)
	expiresAt time.Time
}

// Глобальные параметры для Диффи-Хеллмана
var (
	// Большое простое число (2048 бит)
	prime, _ = new(big.Int).SetString("FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF", 16)
	// Генератор группы (2)
	generator = big.NewInt(2)
)

// NewKeyExchangeService создает новый экземпляр сервиса обмена ключами
func NewKeyExchangeService() *KeyExchangeService {
	return &KeyExchangeService{
		sessions: make(map[string]*keyExchangeSession),
	}
}

// generateSessionID создает уникальный идентификатор сессии
func generateSessionID() string {
	// Генерируем случайные байты
	bytes := make([]byte, 16)
	rand.Read(bytes)

	// Создаем хеш SHA-256
	hash := sha256.Sum256(bytes)

	// Возвращаем первые 32 символа хеша в hex-формате
	return hex.EncodeToString(hash[:16])
}

// InitiateKeyExchange начинает процесс обмена ключами
func (s *KeyExchangeService) InitiateKeyExchange(ctx context.Context, req *pb.InitiateKeyExchangeRequest) (*pb.InitiateKeyExchangeResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Генерируем случайное число a (приватный ключ сервера)
	privateKey := new(big.Int)
	privateKey, err := rand.Int(rand.Reader, prime)
	if err != nil {
		return nil, err
	}

	// Вычисляем публичный ключ сервера: A = g^a mod p
	publicKey := new(big.Int).Exp(generator, privateKey, prime)

	// Создаем новую сессию
	sessionID := generateSessionID()
	session := &keyExchangeSession{
		serverPrivateKey: privateKey,
		serverPublicKey:  publicKey,
		createdAt:        time.Now(),
		expiresAt:        time.Now().Add(24 * time.Hour),
	}
	s.sessions[sessionID] = session

	return &pb.InitiateKeyExchangeResponse{
		Prime:        prime.Bytes(),
		Generator:    generator.Bytes(),
		ServerPublic: publicKey.Bytes(),
		SessionId:    sessionID,
	}, nil
}

// CompleteKeyExchange завершает процесс обмена ключами
func (s *KeyExchangeService) CompleteKeyExchange(ctx context.Context, req *pb.CompleteKeyExchangeRequest) (*pb.CompleteKeyExchangeResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Получаем сессию
	session, exists := s.sessions[req.SessionId]
	if !exists {
		return nil, errors.New("session not found")
	}

	// Проверяем срок действия сессии
	if time.Now().After(session.expiresAt) {
		delete(s.sessions, req.SessionId)
		return nil, errors.New("session expired")
	}

	// Преобразуем публичный ключ клиента в big.Int
	clientPublicKey := new(big.Int).SetBytes(req.ClientPublic)

	// Вычисляем общий секрет: K = (g^b)^a mod p = g^(ab) mod p
	sharedSecret := new(big.Int).Exp(clientPublicKey, session.serverPrivateKey, prime)

	// Удаляем сессию после успешного обмена
	delete(s.sessions, req.SessionId)

	return &pb.CompleteKeyExchangeResponse{
		Success:      true,
		SharedSecret: sharedSecret.Bytes(),
	}, nil
}

// cleanupExpiredSessions удаляет истекшие сессии
func (s *KeyExchangeService) cleanupExpiredSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for sessionID, session := range s.sessions {
		if now.After(session.expiresAt) {
			delete(s.sessions, sessionID)
		}
	}
}
