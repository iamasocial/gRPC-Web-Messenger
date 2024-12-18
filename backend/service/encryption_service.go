package service

import (
	"context"
	pb "messenger/proto"
)

type EncryptService interface {
	Encrypt(ctx context.Context, req *pb.EncryptRequest) (*pb.EncryptResponse, error)
	Decrypt(ctx context.Context, req *pb.DecryptRequest) (*pb.DecryptResponse, error)
}

type encryptionService struct{}

func (es *encryptionService) Encrypt(ctx context.Context, req *pb.EncryptRequest) (*pb.EncryptResponse, error) {
	// need implementation
	return &pb.EncryptResponse{
		EncryptedMessage: req.Message,
	}, nil
}

func (es *encryptionService) Decrypt(ctx context.Context, req *pb.DecryptRequest) (*pb.DecryptResponse, error) {
	// need implementation
	return &pb.DecryptResponse{
		DecryptedMessage: req.EncryptedMessage,
	}, nil
}
