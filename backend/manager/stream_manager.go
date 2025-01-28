package manager

import (
	"fmt"
	pb "gRPCWebServer/backend/generated"
	"sync"
)

type StreamManager interface {
	AddConnection(senderId uint64, receiverUsername string) error
	AddStream(senderId uint64, stream pb.ChatService_ChatServer) error
	GetStream(senderId uint64) (pb.ChatService_ChatServer, error)
	GetReceiverUsername(senderId uint64) (string, error)
	RemoveConnection(senderId uint64) error
}

type connectionData struct {
	receiverUsername string
}

type streamManager struct {
	connections sync.Map
	streams     sync.Map
	mu          sync.RWMutex
}

func NewStreamManager() *streamManager {
	return &streamManager{}
}

func (sm *streamManager) AddConnection(senderId uint64, receiverUsername string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.connections.Load(senderId); exists {
		return fmt.Errorf("connection already exists for sender %d", senderId)
	}

	sm.connections.Store(senderId, &connectionData{receiverUsername: receiverUsername})

	return nil
}

func (sm *streamManager) RemoveConnection(senderId uint64) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.connections.Load(senderId); !exists {
		return fmt.Errorf("no connection found for sender %d", senderId)
	}

	if _, exists := sm.streams.Load(senderId); !exists {
		return fmt.Errorf("no stream found for sender %d", senderId)
	}

	sm.connections.Delete(senderId)
	sm.streams.Delete(senderId)
	return nil
}

func (sm *streamManager) AddStream(senderId uint64, stream pb.ChatService_ChatServer) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.connections.Load(senderId); !exists {
		return fmt.Errorf("no connection found for sender %d", senderId)
	}

	if _, exists := sm.streams.Load(senderId); exists {
		return fmt.Errorf("stream for sender %d already exists", senderId)
	}

	sm.streams.Store(senderId, stream)
	return nil
}

func (sm *streamManager) GetStream(senderId uint64) (pb.ChatService_ChatServer, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stream, exists := sm.streams.Load(senderId)
	if !exists {
		return nil, fmt.Errorf("stream for user %d not found", senderId)
	}

	chatStream, ok := stream.(pb.ChatService_ChatServer)
	if !ok {
		return nil, fmt.Errorf("unexpected type for stream, got %T", stream)
	}

	return chatStream, nil
}

func (sm *streamManager) GetReceiverUsername(senderId uint64) (string, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	data, ok := sm.connections.Load(senderId)
	if !ok {
		return "", fmt.Errorf("receiver username not found")
	}

	connData, ok := data.(*connectionData)
	if !ok {
		return "", fmt.Errorf("unexpected type for connection data, got %T", data)
	}

	return connData.receiverUsername, nil
}
