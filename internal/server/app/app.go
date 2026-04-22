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
)

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
	secretService := service.NewSecretService(secretRepo)

	authHandler := grpchandler.NewAuthHandler(authService)
	secretHandler := grpchandler.NewSecretHandler(secretService)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.LoggingInterceptor(logger),
			middleware.AuthInterceptor(cfg.JWTSecret, logger),
		),
	)

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

	a.logger.Infow("starting gRPC server", "port", a.cfg.ServerPort, "logLevel", a.cfg.LogLevel)

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
