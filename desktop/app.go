package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/F3dosik/GophKeeper/internal/client/grpcclient"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
)

// App — методы, доступные интерфейсу через привязки Wails.
type App struct {
	ctx context.Context
}

// NewApp создаёт App.
func NewApp() *App {
	return &App{}
}

// startup вызывается Wails при запуске; контекст нужен для runtime-функций Wails.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// CheckServer проверяет, что сервер доступен по адресу address и его сертификат
// подписан CA из файла caCertPath (пустой путь — системные корневые сертификаты).
// Возвращает параметры Argon2id, которые сервер отдаёт для нового логина.
func (a *App) CheckServer(address, caCertPath string) (string, error) {
	var caPEM []byte
	if caCertPath != "" {
		var err error
		if caPEM, err = os.ReadFile(caCertPath); err != nil {
			return "", fmt.Errorf("не удалось прочитать сертификат: %w", err)
		}
	}

	conn, err := grpcclient.DialWithOptions(address, grpcclient.DialOptions{CACertPEM: caPEM})
	if err != nil {
		return "", err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, kdf, err := grpcclient.NewAuthClient(pb.NewAuthClient(conn)).GetSalt(ctx, "connection-check")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Сервер доступен, TLS в порядке (Argon2id: t=%d, %d MiB)", kdf.Time, kdf.MemoryKiB/1024), nil
}
