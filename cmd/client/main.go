// Package main — точка входа клиента GophKeeper. Загружает конфигурацию,
// устанавливает gRPC-соединение с сервером и запускает CLI-команды.
package main

import (
	"errors"
	"log"
	"os"

	"github.com/F3dosik/GophKeeper/internal/client/command"
	"github.com/F3dosik/GophKeeper/internal/client/config"
	"github.com/F3dosik/GophKeeper/internal/client/grpcclient"
	"github.com/F3dosik/GophKeeper/internal/client/service"
	"github.com/F3dosik/GophKeeper/internal/client/session"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	token := ""
	sess, err := session.Load(cfg.SessionPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatal(err)
	}
	if sess != nil {
		token = sess.Token
	}

	conn, err := grpcclient.Dial(cfg.ServerAddress, cfg.TLSCertPath, token)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	authClient := grpcclient.NewAuthClient(pb.NewAuthClient(conn))
	secretsClient := grpcclient.NewSecretsClient(pb.NewSecretsClient(conn))
	authSvc := service.NewAuthService(authClient, cfg.SessionPath)

	if command.New(authSvc, secretsClient, cfg).Execute() != nil {
		os.Exit(1)
	}
}
