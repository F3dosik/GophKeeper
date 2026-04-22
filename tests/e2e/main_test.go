//go:build e2e

// Package e2e содержит сквозные интеграционные тесты клиента и сервера GophKeeper.
// Запуск: go test -tags=e2e ./tests/e2e/...
// Требует установленного Docker (используется testcontainers-go для поднятия PostgreSQL).
package e2e

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/F3dosik/GophKeeper/internal/logger"
	"github.com/F3dosik/GophKeeper/internal/server/grpchandler"
	"github.com/F3dosik/GophKeeper/internal/server/middleware"
	"github.com/F3dosik/GophKeeper/internal/server/repository/postgres"
	"github.com/F3dosik/GophKeeper/internal/server/service"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"google.golang.org/grpc"
)

const (
	testJWTSecret = "e2e-jwt-secret-that-is-32-chars!!"
	migrationPath = "../../migrations/000001_init.up.sql"
)

// serverAddr — адрес in-process gRPC сервера, заполняется в TestMain.
var serverAddr string

// TestMain поднимает Postgres-контейнер, применяет миграции, запускает
// gRPC сервер in-process и сохраняет его адрес в serverAddr для использования в тестах.
func TestMain(m *testing.M) {
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:18-alpine",
		tcpostgres.WithDatabase("gophkeeper"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.WithInitScripts(migrationPath),
		tcpostgres.BasicWaitStrategies(),
		tcpostgres.WithSQLDriver("pgx"),
	)
	if err != nil {
		log.Fatalf("start postgres: %v", err)
	}
	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := pgContainer.Terminate(shutCtx); err != nil {
			log.Printf("terminate postgres: %v", err)
		}
	}()

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("pgxpool.New: %v", err)
	}
	defer pool.Close()

	stopServer, addr, err := startTestServer(pool)
	if err != nil {
		log.Fatalf("start server: %v", err)
	}
	defer stopServer()
	serverAddr = addr

	os.Exit(m.Run())
}

// startTestServer собирает gRPC сервер с реальными репозиториями и сервисами,
// запускает его на случайном TCP-порту и возвращает функцию остановки и адрес.
func startTestServer(pool *pgxpool.Pool) (stop func(), addr string, err error) {
	log := logger.New(logger.ModeDevelopment)

	userRepo := postgres.NewUserRepository(pool)
	secretRepo := postgres.NewSecretRepository(pool)

	authService := service.NewAuthService(userRepo, testJWTSecret, time.Hour)
	secretService := service.NewSecretService(secretRepo)

	authHandler := grpchandler.NewAuthHandler(authService)
	secretHandler := grpchandler.NewSecretHandler(secretService)

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.LoggingInterceptor(log),
			middleware.AuthInterceptor(testJWTSecret, log),
		),
	)
	pb.RegisterAuthServer(server, authHandler)
	pb.RegisterSecretsServer(server, secretHandler)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", fmt.Errorf("listen: %w", err)
	}

	go func() {
		if err := server.Serve(lis); err != nil {
			log.Errorf("serve: %v", err)
		}
	}()

	return server.GracefulStop, lis.Addr().String(), nil
}
