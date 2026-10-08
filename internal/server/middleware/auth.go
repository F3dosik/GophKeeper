// Package middleware содержит gRPC унарные interceptor'ы сервера.
package middleware

import (
	"context"
	"strings"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/F3dosik/GophKeeper/internal/server/jwtutil"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type metadataKey string

const tokenKey metadataKey = "authorization"

// publicMethods содержит список методов не требующих аутентификации.
var publicMethods = map[string]bool{
	pb.Auth_GetSalt_FullMethodName:    true,
	pb.Auth_CreateUser_FullMethodName: true,
	pb.Auth_Login_FullMethodName:      true,
	// Административные методы не требуют токена: сервис Admin регистрируется только на
	// административном порту, доступном с localhost; на публичном порту его нет.
	pb.Admin_DeleteUser_FullMethodName: true,
}

// passwordChangeMethods — методы, доступные с токеном временного пароля.
var passwordChangeMethods = map[string]bool{
	pb.Auth_ChangePassword_FullMethodName: true,
	pb.Auth_Logout_FullMethodName:         true,
}

// AuthInterceptor возвращает gRPC унарный interceptor для аутентификации запросов.
// Пропускает публичные методы без проверки токена.
// Извлекает JWT токен из metadata заголовка "authorization",
// валидирует его, проверяет по tokens, что токен не отозван, и добавляет userID
// и claims в контекст запроса.
// Возвращает codes.Unauthenticated если токен отсутствует, невалиден или отозван.
func AuthInterceptor(secretKey string, tokens domain.TokenRepository, logger *zap.SugaredLogger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}
		ctx, err = authenticate(ctx, info.FullMethod, secretKey, tokens, logger)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// AuthStreamInterceptor — потоковый аналог AuthInterceptor: проверяет токен до начала
// обработки потока и передаёт обработчику контекст с userID и claims.
func AuthStreamInterceptor(secretKey string, tokens domain.TokenRepository, logger *zap.SugaredLogger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx, err := authenticate(ss.Context(), info.FullMethod, secretKey, tokens, logger)
		if err != nil {
			return err
		}
		return handler(srv, &contextStream{ServerStream: ss, ctx: ctx})
	}
}

// contextStream подменяет контекст серверного потока.
type contextStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context возвращает подменённый контекст.
func (s *contextStream) Context() context.Context {
	return s.ctx
}

// authenticate извлекает и проверяет токен запроса, включая отзыв, и возвращает
// контекст с userID и claims.
func authenticate(
	ctx context.Context, method, secretKey string, tokens domain.TokenRepository, logger *zap.SugaredLogger,
) (context.Context, error) {
	var token string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get(string(tokenKey))
		if len(values) > 0 {
			token = strings.TrimPrefix(values[0], "Bearer ")
		}
	}

	if len(token) == 0 {
		logger.Warnw("unauthenticated request", "method", method)
		return nil, status.Error(codes.Unauthenticated, "missing token")
	}

	claims, err := jwtutil.ParseToken(token, secretKey)
	if err != nil {
		logger.Warnw("invalid token", "method", method, "error", err)
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	active, err := tokens.IsActive(ctx, claims.UserID, claims.TokenVersion, claims.TokenID)
	if err != nil {
		logger.Errorw("check token revocation", "method", method, "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	if !active {
		logger.Warnw("revoked token", "method", method, "user_id", claims.UserID)
		return nil, status.Error(codes.Unauthenticated, "token revoked")
	}

	if claims.PasswordChangeOnly && !passwordChangeMethods[method] {
		return nil, status.Error(codes.PermissionDenied, domain.ErrPasswordChangeRequired.Error())
	}

	ctx = WithUserID(ctx, claims.UserID)
	return WithClaims(ctx, claims), nil
}
