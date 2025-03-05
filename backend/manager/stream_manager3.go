package manager

import (
	"fmt"
	pb "gRPCWebServer/backend/generated"
	"sync"
)

type StreamManager3 interface {
	AddConnection(senderId, receiverId uint64) error
	AddStream(senderId uint64, stream pb.ChatService_ChatServer) error
	GetConnectionData(senderId uint64) (uint64, error)
	GetStream(senderId uint64) (pb.ChatService_ChatServer, error)
	ClearConnectionAndStream(senderId uint64)
}

type streamManager3 struct {
	connectionData sync.Map
	streams        sync.Map
	mu             sync.RWMutex
}

func NewStreamManager3() *streamManager3 {
	return &streamManager3{}
}

func (sm *streamManager3) AddConnection(senderId, receiverId uint64) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.connectionData.Load(senderId); exists {
		return fmt.Errorf("connection with senderID=%d and receiverID=%d already exists", senderId, receiverId)
	}

	sm.connectionData.Store(senderId, receiverId)

	return nil
}

func (sm *streamManager3) AddStream(senderId uint64, stream pb.ChatService_ChatServer) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.connectionData.Load(senderId); !exists {
		return fmt.Errorf("failed to add stream: no connection found for senderID=%d", senderId)
	}

	if _, exists := sm.streams.Load(senderId); exists {
		return fmt.Errorf("failed to add stream: stream for senderID=%d already exists", senderId)
	}

	sm.streams.Store(senderId, stream)

	return nil
}

func (sm *streamManager3) GetConnectionData(senderId uint64) (uint64, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	value, exists := sm.connectionData.Load(senderId)
	if !exists {
		return 0, fmt.Errorf("failed to get connection data: senderID=%d has no receiver", senderId)
	}

	receiverId, ok := value.(uint64)
	if !ok {
		return 0, fmt.Errorf("failed to get connection data: unexpected receiverID type: expected uint64, got %T", value)
	}

	return receiverId, nil
}

func (sm *streamManager3) GetStream(senderId uint64) (pb.ChatService_ChatServer, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	value, exists := sm.streams.Load(senderId)
	if !exists {
		return nil, fmt.Errorf("stream for userID=%d not found", senderId)
	}

	stream, ok := value.(pb.ChatService_ChatServer)
	if !ok {
		return nil, fmt.Errorf("unexpected stream type for userID=%d: expected pb.ChatService_ChatServer, got %T", senderId, value)
	}

	return stream, nil
}

func (sm *streamManager3) ClearConnectionAndStream(senderId uint64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.connectionData.Delete(senderId)
	sm.streams.Delete(senderId)
}
