// Package app содержит конфигурацию и инициализацию сервера.
package app

import (
	"context"
	"fmt"
	"net"

	"github.com/F3dosik/GophKeeper/internal/server/grpchandler"
	"github.com/F3dosik/GophKeeper/internal/server/middleware"
	"github.com/F3dosik/GophKeeper/internal/server/repository/postgres"
	"github.com/F3dosik/GophKeeper/internal/server/service"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// maxMessageOverhead — запас размера gRPC-сообщения сверх данных секрета (blind index и т.п.).
const maxMessageOverhead = 64 << 10

// App содержит все зависимости и конфигурацию gRPC сервера.
type App struct {
	grpcServer *grpc.Server
	db         *pgxpool.Pool
	cfg        *Config
	logger     *zap.SugaredLogger
}

// New создает новый экемпляр App.
func New(ctx context.Context, cfg *Config, logger *zap.SugaredLogger) (*App, error) {
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("app: new pool: %w", err)
	}

	userRepo := postgres.NewUserRepository(pool)
	secretRepo := postgres.NewSecretRepository(pool)

	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.TokenTTL)
	secretService := service.NewSecretService(secretRepo, service.SecretLimits{
		MaxSize:  cfg.SecretMaxSize,
		MaxCount: cfg.SecretMaxCount,
	})

	authHandler := grpchandler.NewAuthHandler(authService)
	secretHandler := grpchandler.NewSecretHandler(secretService)

	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			middleware.LoggingInterceptor(logger),
			middleware.RateLimitInterceptor(
				middleware.NewIPRateLimiter(cfg.AuthRateLimit, cfg.AuthRateBurst), logger,
			),
			middleware.AuthInterceptor(cfg.JWTSecret, logger),
		),
		// Сообщения больше секрета максимального размера (плюс запас на служебные поля)
		// отклоняются до разбора и не расходуют память сервера.
		grpc.MaxRecvMsgSize(cfg.SecretMaxSize + maxMessageOverhead),
	}
	if cfg.TLSEnabled() {
		creds, err := credentials.NewServerTLSFromFile(cfg.TLSCertFile, cfg.TLSKeyFile)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("app: load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.Creds(creds))
	} else {
		logger.Warn("TLS is disabled: traffic (including JWT tokens) is sent in plaintext; use only for local development")
	}

	grpcServer := grpc.NewServer(opts...)

	pb.RegisterAuthServer(grpcServer, authHandler)
	pb.RegisterSecretsServer(grpcServer, secretHandler)

	return &App{
		grpcServer: grpcServer,
		db:         pool,
		cfg:        cfg,
		logger:     logger,
	}, nil
}

// Run запускает gRPC сервер и блокирует до его остановки.
func (a *App) Run() error {
	listen, err := net.Listen("tcp", a.cfg.ServerPort)
	if err != nil {
		return fmt.Errorf("app.Run: listen: %w", err)
	}

	a.logger.Infow("starting gRPC server",
		"port", a.cfg.ServerPort, "logLevel", a.cfg.LogLevel, "tls", a.cfg.TLSEnabled())

	if err := a.grpcServer.Serve(listen); err != nil {
		return fmt.Errorf("app.Run: serve: %w", err)
	}
	return nil
}

// Stop gracefully останавливает gRPC сервер и закрывает соединение с БД.
func (a *App) Stop() {
	a.grpcServer.GracefulStop()
	a.db.Close()
	a.logger.Info("server stopped")
}
