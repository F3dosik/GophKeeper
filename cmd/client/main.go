// Package main — точка входа клиента Shron. Загружает конфигурацию,
// устанавливает gRPC-соединение с сервером и запускает CLI-команды.
package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/F3dosik/Shron/internal/client/command"
	"github.com/F3dosik/Shron/internal/client/config"
	"github.com/F3dosik/Shron/internal/client/grpcclient"
	"github.com/F3dosik/Shron/internal/client/kdfpin"
	"github.com/F3dosik/Shron/internal/client/service"
	"github.com/F3dosik/Shron/internal/client/session"
	pb "github.com/F3dosik/Shron/proto/gen"
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

	tokens := grpcclient.NewTokenStore(token)
	conn, err := grpcclient.Dial(cfg.ServerAddress, cfg.TLSCertPath, cfg.Insecure, tokens)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	if cfg.Insecure {
		fmt.Fprintln(os.Stderr, "ВНИМАНИЕ: TLS отключён (SHRON_INSECURE), данные передаются открытым текстом")
	}

	authClient := grpcclient.NewAuthClient(pb.NewAuthClient(conn))
	secretsClient := grpcclient.NewSecretsClient(pb.NewSecretsClient(conn))
	adminClient := grpcclient.NewAdminClient(pb.NewAdminClient(conn))
	// Параметры Argon2id учёток запоминаются рядом с файлом сессии (см. kdfpin).
	authSvc := service.NewAuthService(authClient, cfg.SessionPath, tokens,
		service.WithKDFPins(kdfpin.NextTo(cfg.SessionPath), cfg.ServerAddress))

	if command.New(authSvc, secretsClient, adminClient, cfg).Execute() != nil {
		os.Exit(1)
	}
}
