package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/F3dosik/GophKeeper/internal/logger"
	"github.com/F3dosik/GophKeeper/internal/server/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := app.Load()
	if err != nil {
		log.Fatal(err)
	}

	logger := logger.New(logger.Mode(cfg.LogLevel))
	defer func() { _ = logger.Sync() }()

	application, err := app.New(ctx, cfg, logger)
	if err != nil {
		logger.Fatalw("failed to create app", "error", err)
	}

	go func() {
		if err := application.Run(); err != nil {
			logger.Fatalw("server error", "error", err)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down...")
	application.Stop()
}
