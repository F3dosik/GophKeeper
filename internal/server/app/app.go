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
	// adminServer — административный сервер (только Auth с разрешённой регистрацией
	// и временными паролями); nil, если ADMIN_PORT не задан.
	adminServer *grpc.Server
	db          *pgxpool.Pool
	cfg         *Config
	logger      *zap.SugaredLogger
}

// New создает новый экемпляр App.
func New(ctx context.Context, cfg *Config, logger *zap.SugaredLogger) (*App, error) {
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("app: new pool: %w", err)
	}

	userRepo := postgres.NewUserRepository(pool)
	secretRepo := postgres.NewSecretRepository(pool)
	tokenRepo := postgres.NewTokenRepository(pool)

	keys, err := service.DeriveServerKeys(cfg.JWTSecret)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("app: %w", err)
	}
	secretLimits := service.SecretLimits{MaxSize: cfg.SecretMaxSize, MaxCount: cfg.SecretMaxCount}
	authService := service.NewAuthService(userRepo, tokenRepo, service.AuthConfig{
		Keys:            keys,
		TokenTTL:        cfg.TokenTTL,
		SecretLimits:    secretLimits,
		TempPasswordTTL: cfg.TempPasswordTTL,
	})
	secretService := service.NewSecretService(secretRepo, secretLimits)

	var transport []grpc.ServerOption
	if cfg.TLSEnabled() {
		creds, err := credentials.NewServerTLSFromFile(cfg.TLSCertFile, cfg.TLSKeyFile)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("app: load TLS credentials: %w", err)
		}
		transport = append(transport, grpc.Creds(creds))
	} else {
		logger.Warn("TLS is disabled: traffic (including JWT tokens) is sent in plaintext; use only for local development")
	}

	newServer := func(unary ...grpc.UnaryServerInterceptor) *grpc.Server {
		opts := append([]grpc.ServerOption{
			grpc.ChainUnaryInterceptor(append(unary,
				middleware.AuthInterceptor(keys.TokenSigning, tokenRepo, logger))...),
			grpc.ChainStreamInterceptor(
				middleware.LoggingStreamInterceptor(logger),
				middleware.AuthStreamInterceptor(keys.TokenSigning, tokenRepo, logger),
			),
			// Сообщения больше секрета максимального размера (плюс запас на служебные поля)
			// отклоняются до разбора и не расходуют память сервера.
			grpc.MaxRecvMsgSize(cfg.SecretMaxSize + maxMessageOverhead),
		}, transport...)
		return grpc.NewServer(opts...)
	}

	grpcServer := newServer(
		middleware.LoggingInterceptor(logger),
		middleware.RateLimitInterceptor(middleware.NewIPRateLimiter(cfg.AuthRateLimit, cfg.AuthRateBurst), logger),
	)
	pb.RegisterAuthServer(grpcServer, grpchandler.NewAuthHandler(authService, grpchandler.AuthHandlerOptions{
		AllowRegistration: cfg.RegistrationEnabled,
	}))
	pb.RegisterSecretsServer(grpcServer, grpchandler.NewSecretHandler(secretService))

	a := &App{
		grpcServer: grpcServer,
		db:         pool,
		cfg:        cfg,
		logger:     logger,
	}

	if cfg.AdminPort != "" {
		// Административный порт должен быть доступен только с машины сервера
		// (в docker-compose он публикуется на 127.0.0.1), поэтому лимит частоты не нужен.
		a.adminServer = newServer(middleware.LoggingInterceptor(logger))
		pb.RegisterAuthServer(a.adminServer, grpchandler.NewAuthHandler(authService, grpchandler.AuthHandlerOptions{
			AllowRegistration: true,
			AllowTemporary:    true,
		}))
		pb.RegisterAdminServer(a.adminServer, grpchandler.NewAdminHandler(service.NewAdminService(userRepo, tokenRepo)))
	}

	return a, nil
}

// Run запускает gRPC сервер (и административный, если настроен) и блокирует
// до остановки. Возвращает первую ошибку любого из серверов.
func (a *App) Run() error {
	listen, err := net.Listen("tcp", a.cfg.ServerPort)
	if err != nil {
		return fmt.Errorf("app.Run: listen: %w", err)
	}

	errs := make(chan error, 2)
	running := 1
	go func() { errs <- a.grpcServer.Serve(listen) }()
	a.logger.Infow("starting gRPC server",
		"port", a.cfg.ServerPort, "logLevel", a.cfg.LogLevel, "tls", a.cfg.TLSEnabled(),
		"registration", a.cfg.RegistrationEnabled)

	if a.adminServer != nil {
		adminListen, err := net.Listen("tcp", a.cfg.AdminPort)
		if err != nil {
			a.grpcServer.Stop()
			return fmt.Errorf("app.Run: listen admin: %w", err)
		}
		running++
		go func() { errs <- a.adminServer.Serve(adminListen) }()
		a.logger.Infow("starting admin gRPC server", "port", a.cfg.AdminPort)
	}

	for range running {
		if err := <-errs; err != nil {
			return fmt.Errorf("app.Run: serve: %w", err)
		}
	}
	return nil
}

// Stop gracefully останавливает gRPC серверы и закрывает соединение с БД.
func (a *App) Stop() {
	a.grpcServer.GracefulStop()
	if a.adminServer != nil {
		a.adminServer.GracefulStop()
	}
	a.db.Close()
	a.logger.Info("server stopped")
}
