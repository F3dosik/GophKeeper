// Package middleware содержит gRPC унарные interceptor'ы сервера.
package middleware

import (
	"context"
	"strings"

	"github.com/F3dosik/GophKeeper/internal/server/jwtutil"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type metadataKey string

var tokenKey metadataKey = "authorization"

// publicMethods содержит список методов не требующих аутентификации.
var publicMethods = map[string]bool{
	pb.Auth_GetSalt_FullMethodName:    true,
	pb.Auth_CreateUser_FullMethodName: true,
	pb.Auth_Login_FullMethodName:      true,
}

// AuthInterceptor возвращает gRPC унарный interceptor для аутентификации запросов.
// Пропускает публичные методы без проверки токена.
// Извлекает JWT токен из metadata заголовка "authorization",
// валидирует его и добавляет userID в контекст запроса.
// Возвращает codes.Unauthenticated если токен отсутствует или невалиден.
func AuthInterceptor(secretKey string, logger *zap.SugaredLogger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		var token string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			values := md.Get(string(tokenKey))
			if len(values) > 0 {
				token = strings.TrimPrefix(values[0], "Bearer ")
			}
		}

		if len(token) == 0 {
			logger.Warnw("unauthenticated request",
				"method", info.FullMethod,
			)
			return nil, status.Error(codes.Unauthenticated, "missing token")
		}

		claims, err := jwtutil.ParseToken(token, secretKey)
		if err != nil {
			logger.Warnw("invalid token",
				"method", info.FullMethod,
				"error", err,
			)
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		return handler(WithUserID(ctx, claims.UserID), req)
	}
}
