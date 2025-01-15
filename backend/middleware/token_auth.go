package middleware

import (
	"context"
	"errors"
	"gRPCWebServer/backend/utils"
	"log"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type TokenKey string

type AuthInterceptor struct {
	skippedMethods map[string]bool
}

func NewAuthInterceptor() *AuthInterceptor {
	return &AuthInterceptor{
		skippedMethods: map[string]bool{
			"/messenger.UserService/Login":    true,
			"/messenger.UserService/Register": true,
		},
	}
}

func (a *AuthInterceptor) isMethodSkipped(fullMethod string) bool {
	return strings.HasPrefix(fullMethod, "/grpc.reflection.") || a.skippedMethods[fullMethod]
}

func (a *AuthInterceptor) authenticate(ctx context.Context) (context.Context, error) {
	token, err := extractToken(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	claims, err := utils.ValidateToken(token)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	ctx = context.WithValue(ctx, TokenKey("user_id"), claims.UserId)
	return ctx, nil
}

func (a *AuthInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		log.Printf("[Unary Interceptor] Handling method: %s", info.FullMethod)

		if a.isMethodSkipped(info.FullMethod) {
			return handler(ctx, req)
		}

		ctx, err := a.authenticate(ctx)
		if err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

func (a *AuthInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		log.Printf("[Stream Interceptor] Handling method: %s", info.FullMethod)

		if a.isMethodSkipped(info.FullMethod) {
			return handler(srv, stream)
		}

		ctx, err := a.authenticate(stream.Context())
		if err != nil {
			return err
		}

		wrappedStream := &wrappedServerStream{ServerStream: stream, ctx: ctx}
		return handler(srv, wrappedStream)
	}
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
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

// package middleware

// import (
// 	"context"
// 	"errors"
// 	"log"
// 	"gRPCWebServer/backend/utils"
// 	"strings"

// 	"google.golang.org/grpc"
// 	"google.golang.org/grpc/codes"
// 	"google.golang.org/grpc/metadata"
// 	"google.golang.org/grpc/status"
// )

// type TokenKey string

// type AuthInterceptor struct {
// 	skippedMethods map[string]bool
// }

// func NewAuthInterceptor() *AuthInterceptor {
// 	return &AuthInterceptor{
// 		skippedMethods: map[string]bool{
// 			"/messenger.UserService/Login":    true,
// 			"/messenger.UserService/Register": true,
// 		},
// 	}
// }

// func (a *AuthInterceptor) isSkippedMethod(method string) bool {
// 	// return a.skippedMethods[method] || strings.HasPrefix(method, "/grpc.reflection.v1.")
// 	return a.skippedMethods[method]
// }

// type wrappedServerStream struct {
// 	grpc.ServerStream
// 	ctx context.Context
// }

// func (w *wrappedServerStream) Context() context.Context {
// 	return w.ctx
// }

// func (a *AuthInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
// 	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
// 		log.Printf("[Unary Interceptor] Full method: %s", info.FullMethod)
// 		if info.FullMethod == "/messenger.UserService/Login" || info.FullMethod == "/messenger.UserService/Register" {
// 			token, err := extractToken(ctx)
// 			if err == nil && token != "" {
// 				_, err := utils.ValidateToken(token)
// 				if err == nil {
// 					return nil, status.Errorf(codes.FailedPrecondition, "user is already logged in")
// 				}
// 			}
// 			return handler(ctx, req)
// 		}
// 		// if a.isSkippedMethod(info.FullMethod) {
// 		// 	if token, _ := extractToken(ctx); token != "" {
// 		// 		return nil, status.Errorf(codes.InvalidArgument, "token should not be provided for %s", info.FullMethod)
// 		// 	}
// 		// 	return handler(ctx, req)
// 		// }

// 		token, err := extractToken(ctx)
// 		if err != nil {
// 			return nil, status.Errorf(codes.Unauthenticated, "unathorized: %v", err)
// 		}

// 		claims, err := utils.ValidateToken(token)
// 		if err != nil {
// 			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
// 		}

// 		ctx = context.WithValue(ctx, TokenKey("user_id"), claims.UserId)

// 		return handler(ctx, req)
// 	}
// }

// func (a *AuthInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
// 	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
// 		// if info.FullMethod == "/messenger.UserService/Login" || info.FullMethod == "/messenger.UserService/Register" {
// 		// 	token, err := extractToken(stream.Context())
// 		// 	if err == nil && token != "" {
// 		// 		_, err := utils.ValidateToken(token)
// 		// 		if err == nil {
// 		// 			return status.Errorf(codes.FailedPrecondition, "user is already logged in")
// 		// 		}
// 		// 	}
// 		// 	return handler(srv, stream)
// 		// }
// 		log.Printf("[Stream Interceptor] Full method: %s, IsClientStream: %v, IsServerStream: %v",
// 			info.FullMethod, info.IsClientStream, info.IsServerStream)

// 		if strings.HasPrefix(info.FullMethod, "/grpc.reflection.") {
// 			log.Printf("[Stream Interceptor] Skipping reflection method: %s", info.FullMethod)
// 			return handler(srv, stream)
// 		}

// 		if !info.IsServerStream && !info.IsClientStream {
// 			log.Printf("[Stream Interceptor] Skipping non-streaming method: %s", info.FullMethod)
// 			return handler(srv, stream)
// 		}

// 		// log.Printf("[Stream Interceptor] Full method: %s", info.FullMethod)
// 		if a.isSkippedMethod(info.FullMethod) {
// 			if token, _ := extractToken(stream.Context()); token != "" {
// 				return status.Errorf(codes.InvalidArgument, "token should not be provided for %s", info.FullMethod)
// 			}
// 			return handler(srv, stream)
// 		}

// 		token, err := extractToken(stream.Context())
// 		if err != nil {
// 			return status.Errorf(codes.Unauthenticated, "unathorized: %v", err)
// 		}

// 		claims, err := utils.ValidateToken(token)
// 		if err != nil {
// 			return status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
// 		}

// 		ctx := context.WithValue(stream.Context(), TokenKey("user_id"), claims.UserId)
// 		wrappedStream := &wrappedServerStream{ServerStream: stream, ctx: ctx}
// 		// return srv.(grpc.ServiceRegistrar).(grpc.ServerStream).RecvMsg(stream)
// 		return handler(srv, wrappedStream)
// 	}
// }

// func extractToken(ctx context.Context) (string, error) {
// 	md, ok := metadata.FromIncomingContext(ctx)
// 	if !ok {
// 		return "", errors.New("metadata is not provided")
// 	}

// 	authHeader, exists := md["authorization"]
// 	if !exists || len(authHeader) == 0 {
// 		return "", errors.New("authorization header is missing")
// 	}

// 	tokenParts := strings.SplitN(authHeader[0], " ", 2)
// 	if len(tokenParts) != 2 || strings.ToLower(tokenParts[0]) != "bearer" {
// 		return "", errors.New("invalid authorization header format")
// 	}

// 	return tokenParts[1], nil
// }
