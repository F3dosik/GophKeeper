package middleware

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type contextKey string

const userIDKey contextKey = "user_id"

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
