//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/F3dosik/GophKeeper/internal/client/grpcclient"
	"github.com/F3dosik/GophKeeper/internal/client/service"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// clientKit — набор клиентских зависимостей для одного тестового пользователя.
// Каждый тест создаёт свой kit через newClientKit, чтобы пользователи не пересекались.
type clientKit struct {
	Login       string
	Password    string
	SessionPath string
	Auth        service.AuthService
	Secrets     service.SecretsService
	// Tokens — токен соединения; как и в клиенте, общий для Auth и Secrets.
	Tokens *grpcclient.TokenStore
	conn   *grpc.ClientConn
}

// newClientKit поднимает gRPC-соединение с тестовым сервером без TLS так же,
// как cmd/client, создаёт AuthService и возвращает kit с уникальным логином.
// SecretsService заполняется после вызова Login через initSecretsService.
func newClientKit(t *testing.T) *clientKit {
	t.Helper()

	tokens := grpcclient.NewTokenStore("")
	conn, err := grpcclient.Dial(serverAddr, "", true, tokens)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	sessionPath := filepath.Join(t.TempDir(), "session")
	login := fmt.Sprintf("user-%s", uuid.NewString())

	authClient := grpcclient.NewAuthClient(pb.NewAuthClient(conn))
	return &clientKit{
		Login:       login,
		Password:    "test-password-123",
		SessionPath: sessionPath,
		Auth:        service.NewAuthService(authClient, sessionPath, tokens),
		Tokens:      tokens,
		conn:        conn,
	}
}

// initSecretsService проверяет пароль через Unlock и создаёт SecretsService
// на том же соединении, что и AuthService. Должен вызываться после Login.
func (k *clientKit) initSecretsService(ctx context.Context, t *testing.T) {
	t.Helper()

	masterKey, err := k.Auth.Unlock(ctx, k.Login, k.Password)
	require.NoError(t, err)

	secretsClient := grpcclient.NewSecretsClient(pb.NewSecretsClient(k.conn))
	secretsSvc, err := service.NewSecretsService(secretsClient, masterKey)
	require.NoError(t, err)
	k.Secrets = secretsSvc
}

// registerAndLogin регистрирует пользователя и выполняет вход.
// Инициализирует Secrets-сервис, готовый для CRUD-операций.
func (k *clientKit) registerAndLogin(ctx context.Context, t *testing.T) {
	t.Helper()
	require.NoError(t, k.Auth.CreateUser(ctx, k.Login, k.Password))
	require.NoError(t, k.Auth.Login(ctx, k.Login, k.Password))
	k.initSecretsService(ctx, t)
}
