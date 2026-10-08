package middleware

import (
	"context"
	"net"
	"testing"
	"time"

	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestIPRateLimiter_BurstAndRefill(t *testing.T) {
	now := time.Unix(0, 0)
	l := NewIPRateLimiter(60, 2) // 1 запрос в секунду, 2 подряд
	l.now = func() time.Time { return now }

	assert.True(t, l.Allow("1.1.1.1"))
	assert.True(t, l.Allow("1.1.1.1"))
	assert.False(t, l.Allow("1.1.1.1"), "burst exhausted")
	assert.True(t, l.Allow("2.2.2.2"), "other IPs are limited independently")

	now = now.Add(time.Second)
	assert.True(t, l.Allow("1.1.1.1"), "token refilled after 1s")
	assert.False(t, l.Allow("1.1.1.1"))
}

func TestIPRateLimiter_SweepsIdleEntries(t *testing.T) {
	now := time.Unix(0, 0)
	l := NewIPRateLimiter(60, 1)
	l.now = func() time.Time { return now }

	l.Allow("1.1.1.1")
	now = now.Add(2 * limiterIdleTTL)
	l.Allow("2.2.2.2")

	assert.NotContains(t, l.limiters, "1.1.1.1")
	assert.Contains(t, l.limiters, "2.2.2.2")
}

func TestRateLimitInterceptor(t *testing.T) {
	l := NewIPRateLimiter(60, 1)
	interceptor := RateLimitInterceptor(l, zap.NewNop().Sugar())
	handler := func(ctx context.Context, req any) (any, error) { return "ok", nil }
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP("10.0.0.1"), Port: 1234},
	})
	login := &grpc.UnaryServerInfo{FullMethod: pb.Auth_Login_FullMethodName}

	_, err := interceptor(ctx, nil, login, handler)
	assert.NoError(t, err)

	_, err = interceptor(ctx, nil, login, handler)
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))

	t.Run("non-auth methods are not limited", func(t *testing.T) {
		info := &grpc.UnaryServerInfo{FullMethod: pb.Secrets_ListSecrets_FullMethodName}
		for range 5 {
			_, err := interceptor(ctx, nil, info, handler)
			assert.NoError(t, err)
		}
	})
}
