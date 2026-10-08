package grpcclient

import (
	"context"

	"github.com/F3dosik/GophKeeper/internal/domain"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
)

// AuthClient определяет интерфейс для взаимодействия с сервисом аутентификации.
type AuthClient interface {
	// CreateUser регистрирует нового пользователя на сервере.
	// salt должен быть сгенерирован клиентом перед деривацией ключей.
	// kdf — параметры Argon2id, с которыми выведен ключ аутентификации.
	CreateUser(ctx context.Context, creds domain.Credentials, salt []byte, kdf domain.KDFParams) error

	// GetSalt возвращает соль пользователя по логину.
	// Используется для деривации ключей перед аутентификацией.
	// Вместе с солью возвращает параметры Argon2id пользователя.
	GetSalt(ctx context.Context, login string) ([]byte, domain.KDFParams, error)

	// Login аутентифицирует пользователя и возвращает JWT токен.
	Login(ctx context.Context, creds domain.Credentials) (string, error)

	// Logout отзывает на сервере текущий токен или, если allSessions == true,
	// все токены пользователя.
	Logout(ctx context.Context, allSessions bool) error
}

type authClient struct {
	client pb.AuthClient
}

// NewAuthClient создаёт новый AuthClient поверх сгенерированного gRPC клиента.
func NewAuthClient(client pb.AuthClient) AuthClient {
	return &authClient{client: client}
}

func (c *authClient) CreateUser(ctx context.Context, creds domain.Credentials, salt []byte, kdf domain.KDFParams) error {
	req := pb.CreateUserRequest_builder{
		Credentials: toPBCredentials(creds),
		Salt:        salt,
		Kdf:         toPBKDF(kdf),
	}.Build()
	_, err := c.client.CreateUser(ctx, req)
	return fromGRPCError(err)
}

func (c *authClient) GetSalt(ctx context.Context, login string) ([]byte, domain.KDFParams, error) {
	req := pb.GetSaltRequest_builder{Login: &login}.Build()
	resp, err := c.client.GetSalt(ctx, req)
	if err != nil {
		return nil, domain.KDFParams{}, fromGRPCError(err)
	}
	return resp.GetSalt(), fromPBKDF(resp.GetKdf()), nil
}

func (c *authClient) Login(ctx context.Context, creds domain.Credentials) (string, error) {
	req := pb.LoginRequest_builder{
		Credentials: toPBCredentials(creds),
	}.Build()
	resp, err := c.client.Login(ctx, req)
	if err != nil {
		return "", fromGRPCError(err)
	}
	return resp.GetToken(), nil
}

func (c *authClient) Logout(ctx context.Context, allSessions bool) error {
	req := pb.LogoutRequest_builder{AllSessions: &allSessions}.Build()
	_, err := c.client.Logout(ctx, req)
	return fromGRPCError(err)
}
