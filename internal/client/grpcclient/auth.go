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
	CreateUser(ctx context.Context, creds domain.Credentials, salt []byte) error

	// GetSalt возвращает соль пользователя по логину.
	// Используется для деривации ключей перед аутентификацией.
	GetSalt(ctx context.Context, login string) ([]byte, error)

	// Login аутентифицирует пользователя и возвращает JWT токен.
	Login(ctx context.Context, creds domain.Credentials) (string, error)
}

type authClient struct {
	client pb.AuthClient
}

// NewAuthClient создаёт новый AuthClient поверх сгенерированного gRPC клиента.
func NewAuthClient(client pb.AuthClient) AuthClient {
	return &authClient{client: client}
}

func (c *authClient) CreateUser(ctx context.Context, creds domain.Credentials, salt []byte) error {
	req := pb.CreateUserRequest_builder{
		Credentials: toPBCredentials(creds),
		Salt:        salt,
	}.Build()
	_, err := c.client.CreateUser(ctx, req)
	return fromGRPCError(err)
}

func (c *authClient) GetSalt(ctx context.Context, login string) ([]byte, error) {
	req := pb.GetSaltRequest_builder{Login: &login}.Build()
	resp, err := c.client.GetSalt(ctx, req)
	if err != nil {
		return nil, fromGRPCError(err)
	}
	return resp.GetSalt(), nil
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
