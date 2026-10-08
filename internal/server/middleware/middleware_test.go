package middleware_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/F3dosik/GophKeeper/internal/server/jwtutil"
	"github.com/F3dosik/GophKeeper/internal/server/middleware"
	"github.com/F3dosik/GophKeeper/internal/server/mocks"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const testSecret = "test-jwt-secret"

// fakeHandler имитирует следующий handler в цепочке interceptor'ов.
func fakeHandler(ctx context.Context, req any) (any, error) {
	return "ok", nil
}

func TestAuthInterceptor_PublicMethod(t *testing.T) {
	interceptor := middleware.AuthInterceptor(testSecret, mocks.NewTokenRepository(t), zap.NewNop().Sugar())

	info := &grpc.UnaryServerInfo{FullMethod: pb.Auth_Login_FullMethodName}
	resp, err := interceptor(context.Background(), nil, info, fakeHandler)

	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestAuthInterceptor_MissingToken(t *testing.T) {
	interceptor := middleware.AuthInterceptor(testSecret, mocks.NewTokenRepository(t), zap.NewNop().Sugar())

	info := &grpc.UnaryServerInfo{FullMethod: pb.Secrets_GetSecret_FullMethodName}
	_, err := interceptor(context.Background(), nil, info, fakeHandler)

	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestAuthInterceptor_InvalidToken(t *testing.T) {
	interceptor := middleware.AuthInterceptor(testSecret, mocks.NewTokenRepository(t), zap.NewNop().Sugar())

	md := metadata.Pairs("authorization", "Bearer invalid-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	info := &grpc.UnaryServerInfo{FullMethod: pb.Secrets_GetSecret_FullMethodName}
	_, err := interceptor(ctx, nil, info, fakeHandler)

	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// tokenContext возвращает входящий контекст с новым токеном пользователя и его claims.
func tokenContext(t *testing.T, userID uuid.UUID, version int) (context.Context, *jwtutil.Claims) {
	t.Helper()

	token, err := jwtutil.GenerateToken(userID, version, testSecret, time.Hour)
	require.NoError(t, err)
	claims, err := jwtutil.ParseToken(token, testSecret)
	require.NoError(t, err)

	md := metadata.Pairs("authorization", "Bearer "+token)
	return metadata.NewIncomingContext(context.Background(), md), claims
}

func TestAuthInterceptor_ValidToken(t *testing.T) {
	userID := uuid.New()
	ctx, claims := tokenContext(t, userID, 3)

	tokens := mocks.NewTokenRepository(t)
	tokens.On("IsActive", mock.Anything, userID, 3, claims.TokenID).Return(true, nil)
	interceptor := middleware.AuthInterceptor(testSecret, tokens, zap.NewNop().Sugar())

	info := &grpc.UnaryServerInfo{FullMethod: pb.Secrets_GetSecret_FullMethodName}
	resp, err := interceptor(ctx, nil, info, func(ctx context.Context, req any) (any, error) {
		gotID, err := middleware.UserIDFromContext(ctx)
		require.NoError(t, err)
		assert.Equal(t, userID, gotID)

		gotClaims, err := middleware.ClaimsFromContext(ctx)
		require.NoError(t, err)
		assert.Equal(t, claims.TokenID, gotClaims.TokenID)
		return "ok", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestAuthInterceptor_RevokedToken(t *testing.T) {
	userID := uuid.New()
	ctx, _ := tokenContext(t, userID, 0)

	tokens := mocks.NewTokenRepository(t)
	tokens.On("IsActive", mock.Anything, userID, 0, mock.Anything).Return(false, nil)
	interceptor := middleware.AuthInterceptor(testSecret, tokens, zap.NewNop().Sugar())

	info := &grpc.UnaryServerInfo{FullMethod: pb.Secrets_GetSecret_FullMethodName}
	_, err := interceptor(ctx, nil, info, fakeHandler)

	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestAuthInterceptor_RevocationCheckFails(t *testing.T) {
	userID := uuid.New()
	ctx, _ := tokenContext(t, userID, 0)

	tokens := mocks.NewTokenRepository(t)
	tokens.On("IsActive", mock.Anything, userID, 0, mock.Anything).Return(false, errors.New("db down"))
	interceptor := middleware.AuthInterceptor(testSecret, tokens, zap.NewNop().Sugar())

	info := &grpc.UnaryServerInfo{FullMethod: pb.Secrets_GetSecret_FullMethodName}
	_, err := interceptor(ctx, nil, info, fakeHandler)

	assert.Equal(t, codes.Internal, status.Code(err), "fail closed when revocation state is unknown")
}

func TestLoggingInterceptor(t *testing.T) {
	interceptor := middleware.LoggingInterceptor(zap.NewNop().Sugar())

	info := &grpc.UnaryServerInfo{FullMethod: pb.Auth_Login_FullMethodName}
	resp, err := interceptor(context.Background(), nil, info, fakeHandler)

	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

// Токен временного пароля разрешает только смену пароля и выход.
func TestAuthInterceptor_PasswordChangeOnlyToken(t *testing.T) {
	userID := uuid.New()
	token, err := jwtutil.GeneratePasswordChangeToken(userID, 0, testSecret, time.Minute)
	require.NoError(t, err)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

	tokens := mocks.NewTokenRepository(t)
	tokens.On("IsActive", mock.Anything, userID, 0, mock.Anything).Return(true, nil)
	interceptor := middleware.AuthInterceptor(testSecret, tokens, zap.NewNop().Sugar())

	_, err = interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: pb.Secrets_ListSecrets_FullMethodName}, fakeHandler)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	_, err = interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: pb.Auth_Logout_FullMethodName}, fakeHandler)
	assert.NoError(t, err)
}
