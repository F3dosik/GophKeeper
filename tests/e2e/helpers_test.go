//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/F3dosik/GophKeeper/internal/client/grpcclient"
	"github.com/F3dosik/GophKeeper/internal/client/service"
	"github.com/F3dosik/GophKeeper/internal/client/session"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// clientKit — набор клиентских зависимостей для одного тестового пользователя.
// Каждый тест создаёт свой kit через newClientKit, чтобы пользователи не пересекались.
type clientKit struct {
	Login       string
	Password    string
	SessionPath string
	Auth        service.AuthService
	Secrets     service.SecretsService
	conn        *grpc.ClientConn
}

// newClientKit поднимает gRPC-соединение с тестовым сервером без TLS,
// создаёт AuthService и возвращает kit с уникальным логином. SecretsService
// заполняется после вызова Login через initSecretsService.
func newClientKit(t *testing.T) *clientKit {
	t.Helper()

	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	sessionPath := filepath.Join(t.TempDir(), "session")
	login := fmt.Sprintf("user-%s", uuid.NewString())

	authClient := grpcclient.NewAuthClient(pb.NewAuthClient(conn))
	return &clientKit{
		Login:       login,
		Password:    "test-password-123",
		SessionPath: sessionPath,
		Auth:        service.NewAuthService(authClient, sessionPath),
		conn:        conn,
	}
}

// initSecretsService создаёт SecretsService с использованием мастер-ключа,
// деривированного из пароля. Должен вызываться после Login.
// Для secrets-операций нужен новый conn с авторизационным интерцептором,
// поэтому подменяем conn внутри kit.
func (k *clientKit) initSecretsService(ctx context.Context, t *testing.T) {
	t.Helper()

	sess, err := session.Load(k.SessionPath)
	require.NoError(t, err)

	authConn, err := grpcclient.Dial(serverAddr, "", sess.Token)
	require.NoError(t, err)
	t.Cleanup(func() { _ = authConn.Close() })

	masterKey, err := k.Auth.DeriveMasterKey(ctx, k.Login, k.Password)
	require.NoError(t, err)

	secretsClient := grpcclient.NewSecretsClient(pb.NewSecretsClient(authConn))
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
