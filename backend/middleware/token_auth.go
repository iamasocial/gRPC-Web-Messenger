package middleware

import (
	"context"
	"errors"
	"messenger/backend/utils"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type TokenKey string

type AuthInterceptor struct{}

func NewAuthInterceptor() *AuthInterceptor {
	return &AuthInterceptor{}
}

func (a *AuthInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if info.FullMethod == "/messenger.UserService/Login" || info.FullMethod == "/messenger.UserService/Register" {
			token, err := extractToken(ctx)
			if err == nil && token != "" {
				_, err := utils.ValidateToken(token)
				if err == nil {
					return nil, status.Errorf(codes.FailedPrecondition, "user is already logged in")
				}
			}
			return handler(ctx, req)
		}

		token, err := extractToken(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "unathorized: %v", err)
		}

		claims, err := utils.ValidateToken(token)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		ctx = context.WithValue(ctx, TokenKey("user_id"), claims.UserId)

		return handler(ctx, req)
	}
}

func extractToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("metadata is not provided")
	}

	authHeader, exists := md["authorization"]
	if !exists || len(authHeader) == 0 {
		return "", errors.New("authorization header is missing")
	}

	tokenParts := strings.SplitN(authHeader[0], " ", 2)
	if len(tokenParts) != 2 || strings.ToLower(tokenParts[0]) != "bearer" {
		return "", errors.New("invalid authorization header format")
	}

	return tokenParts[1], nil
}
