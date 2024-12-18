package middleware

import (
	"context"
	"errors"
	"messenger/backend/repository"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// type UserIdKey string
type TokenKey string

type AuthInterceptor struct {
	tokenRepo repository.TokenRepository
}

func NewAuthInterceptor(tr repository.TokenRepository) *AuthInterceptor {
	return &AuthInterceptor{tokenRepo: tr}
}

func (a *AuthInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if info.FullMethod == "/messenger.UserService/Login" || info.FullMethod == "/messenger.UserService/Register" {
			tokenString, err := extractToken(ctx)
			if err == nil && tokenString != "" {
				tokenEntity, err := a.tokenRepo.GetByToken(ctx, tokenString)
				if err == nil && tokenEntity != nil {
					activeToken, err := a.tokenRepo.GetActiveToken(ctx, tokenEntity.UserID)
					if err == nil && activeToken != nil {
						return nil, status.Errorf(codes.FailedPrecondition, "user is already logged in")
					}
				}
			}
			return handler(ctx, req)
		}

		token, err := extractToken(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "unathorized: %v", err)
		}

		tokenEntity, err := a.tokenRepo.GetByToken(ctx, token)
		if err != nil || token == "" {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		ctx = context.WithValue(ctx, TokenKey("token"), tokenEntity.Token)

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

// func AuthInterceptor(excludeMethods []string) grpc.UnaryServerInterceptor {
// 	return func(
// 		ctx context.Context,
// 		req any,
// 		info *grpc.UnaryServerInfo,
// 		handler grpc.UnaryHandler,
// 	) (resp any, err error) {
// 		for _, method := range excludeMethods {
// 			if info.FullMethod == method {
// 				return handler(ctx, req)
// 			}
// 		}

// 		md, ok := metadata.FromIncomingContext(ctx)
// 		if !ok {
// 			return nil, errors.New("missing metadata")
// 		}

// 		token := ""
// 		if val := md["authorization"]; len(val) > 0 {
// 			token = val[0]
// 		}

// 		if token == "" {
// 			return nil, errors.New("missing token")
// 		}

// 		valid, err := utils.ValidateToken(token)
// 		if err != nil {
// 			return nil, fmt.Errorf("token validation failed: %w", err)
// 		}

// 		if valid != nil {
// 			return nil, errors.New("invalid token")
// 		}
// 		// if token == "" || !utils.ValidateToken(token) {
// 		// 	return nil, errors.New("invalid or missing token")
// 		// }

// 		return handler(ctx, req)
// 	}
// }

// func TokenAuthMiddleware(
// 	ctx context.Context,
// 	req interface{},
// 	info *grpc.UnaryServerInfo,
// 	handler grpc.UnaryHandler,
// ) (interface{}, error) {
// 	md, ok := metadata.FromIncomingContext(ctx)
// 	if !ok {
// 		return nil, errors.New("missing metadata")
// 	}

// 	tokens := md.Get("authorization")
// 	if len(tokens) == 0 {
// 		return nil, errors.New("authorization token is required")
// 	}

// 	token := tokens[0]
// 	if !strings.HasPrefix(token, "Bearer ") {
// 		return nil, fmt.Errorf("invalid token format")
// 	}
// 	token = token[7:]

// 	_, err := utils.ValidateToken(token)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid token: %v", err)
// 	}

// 	return handler(ctx, req)
// }
