package middleware

import (
	"context"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// limiterIdleTTL — через сколько неактивный IP удаляется из таблицы лимитеров.
const limiterIdleTTL = 10 * time.Minute

// IPRateLimiter ограничивает частоту запросов с одного IP-адреса (token bucket на каждый IP).
// Неактивные записи периодически удаляются, чтобы таблица не росла бесконечно.
type IPRateLimiter struct {
	mu        sync.Mutex
	limiters  map[string]*ipLimiter
	limit     rate.Limit
	burst     int
	lastSweep time.Time
	now       func() time.Time
}

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewIPRateLimiter создаёт лимитер: perMinute запросов в минуту в среднем
// и не более burst запросов подряд.
func NewIPRateLimiter(perMinute, burst int) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*ipLimiter),
		limit:    rate.Limit(float64(perMinute) / 60),
		burst:    burst,
		now:      time.Now,
	}
}

// Allow сообщает, можно ли выполнить ещё один запрос с адреса ip.
func (l *IPRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if now.Sub(l.lastSweep) > limiterIdleTTL {
		for key, entry := range l.limiters {
			if now.Sub(entry.lastSeen) > limiterIdleTTL {
				delete(l.limiters, key)
			}
		}
		l.lastSweep = now
	}

	entry, ok := l.limiters[ip]
	if !ok {
		entry = &ipLimiter{limiter: rate.NewLimiter(l.limit, l.burst)}
		l.limiters[ip] = entry
	}
	entry.lastSeen = now
	return entry.limiter.AllowN(now, 1)
}

// RateLimitInterceptor возвращает gRPC унарный interceptor, ограничивающий частоту
// вызовов публичных методов аутентификации (GetSalt, CreateUser, Login) с одного IP.
// Это замедляет онлайн-перебор паролей и массовую регистрацию.
// При превышении лимита возвращает codes.ResourceExhausted.
func RateLimitInterceptor(limiter *IPRateLimiter, logger *zap.SugaredLogger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		ip := clientIP(ctx)
		if !limiter.Allow(ip) {
			logger.Warnw("rate limit exceeded", "method", info.FullMethod, "client_ip", ip)
			return nil, status.Error(codes.ResourceExhausted, "too many requests, try again later")
		}
		return handler(ctx, req)
	}
}

// clientIP возвращает IP-адрес клиента без порта.
func clientIP(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok || p.Addr == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(p.Addr.String())
	if err != nil {
		return p.Addr.String()
	}
	return host
}
