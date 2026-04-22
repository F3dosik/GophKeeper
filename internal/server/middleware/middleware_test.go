package middleware_test

import (
	"context"
	"testing"
	"time"

	"github.com/F3dosik/GophKeeper/internal/server/jwtutil"
	"github.com/F3dosik/GophKeeper/internal/server/middleware"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
	interceptor := middleware.AuthInterceptor(testSecret, zap.NewNop().Sugar())

	info := &grpc.UnaryServerInfo{FullMethod: pb.Auth_Login_FullMethodName}
	resp, err := interceptor(context.Background(), nil, info, fakeHandler)

	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestAuthInterceptor_MissingToken(t *testing.T) {
	interceptor := middleware.AuthInterceptor(testSecret, zap.NewNop().Sugar())

	info := &grpc.UnaryServerInfo{FullMethod: pb.Secrets_GetSecret_FullMethodName}
	_, err := interceptor(context.Background(), nil, info, fakeHandler)

	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestAuthInterceptor_InvalidToken(t *testing.T) {
	interceptor := middleware.AuthInterceptor(testSecret, zap.NewNop().Sugar())

	md := metadata.Pairs("authorization", "Bearer invalid-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	info := &grpc.UnaryServerInfo{FullMethod: pb.Secrets_GetSecret_FullMethodName}
	_, err := interceptor(ctx, nil, info, fakeHandler)

	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestAuthInterceptor_ValidToken(t *testing.T) {
	interceptor := middleware.AuthInterceptor(testSecret, zap.NewNop().Sugar())

	userID := uuid.New()
	token, err := jwtutil.GenerateToken(userID, testSecret, time.Hour)
	assert.NoError(t, err)

	md := metadata.Pairs("authorization", "Bearer "+token)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	info := &grpc.UnaryServerInfo{FullMethod: pb.Secrets_GetSecret_FullMethodName}
	resp, err := interceptor(ctx, nil, info, fakeHandler)

	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestLoggingInterceptor(t *testing.T) {
	interceptor := middleware.LoggingInterceptor(zap.NewNop().Sugar())

	info := &grpc.UnaryServerInfo{FullMethod: pb.Auth_Login_FullMethodName}
	resp, err := interceptor(context.Background(), nil, info, fakeHandler)

	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
}
