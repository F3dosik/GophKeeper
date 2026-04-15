package middleware

import (
	"context"
	"time"

	"go.uber.org/zap"
	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

// LoggingInterceptor возвращает gRPC унарный interceptor для логирования запросов.
// Логирует метод, IP клиента, длительность и ошибку каждого запроса.
func LoggingInterceptor(logger *zap.SugaredLogger) grpclib.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpclib.UnaryServerInfo, handler grpclib.UnaryHandler) (any, error) {
		start := time.Now()

		var clientIP string
		if p, ok := peer.FromContext(ctx); ok {
			clientIP = p.Addr.String()
		}
		resp, err := handler(ctx, req)

		logger.Infow("request",
			"method", info.FullMethod,
			"duration", time.Since(start),
			"client_ip", clientIP,
			"error", err,
		)

		return resp, err
	}
}
