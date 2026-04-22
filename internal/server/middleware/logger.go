package middleware

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

// LoggingInterceptor возвращает gRPC унарный interceptor для логирования запросов.
// Логирует метод, IP клиента, длительность и ошибку (при ее наличии) каждого запроса.
func LoggingInterceptor(logger *zap.SugaredLogger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		var clientIP string
		if p, ok := peer.FromContext(ctx); ok {
			clientIP = p.Addr.String()
		}
		resp, err := handler(ctx, req)

		fields := []any{
			"method", info.FullMethod,
			"duration", time.Since(start),
			"client_ip", clientIP,
		}
		if err != nil {
			fields = append(fields, "error", err)
		}
		logger.Infow("request", fields...)

		return resp, err
	}
}
