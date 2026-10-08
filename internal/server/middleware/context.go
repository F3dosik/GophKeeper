package middleware

import (
	"context"

	"github.com/F3dosik/GophKeeper/internal/server/jwtutil"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type contextKey string

const (
	userIDKey contextKey = "user_id"
	claimsKey contextKey = "claims"
)

// UserIDFromContext извлекает userID из контекста установленного middleware.
// Возвращает codes.Unauthenticated если userID отсутствует в контексте.
func UserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, status.Error(codes.Unauthenticated, "unauthenticated")
	}
	return userID, nil
}

// WithUserID добавляет userID в контекст.
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// WithClaims добавляет claims проверенного токена в контекст.
func WithClaims(ctx context.Context, claims *jwtutil.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// ClaimsFromContext извлекает claims токена текущего запроса.
// Возвращает codes.Unauthenticated если claims отсутствуют в контексте.
func ClaimsFromContext(ctx context.Context) (*jwtutil.Claims, error) {
	claims, ok := ctx.Value(claimsKey).(*jwtutil.Claims)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthenticated")
	}
	return claims, nil
}
