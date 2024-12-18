package service

import "messenger/proto"

type ChatService struct {
	proto.UnimplementedChatServiceServer
}

func NewChatService() *ChatService {
	return &ChatService{}
}
