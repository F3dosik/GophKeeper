package grpcclient

import (
	"errors"
	"context"

	"github.com/F3dosik/GophKeeper/internal/domain"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
)

// SecretsClient определяет интерфейс для взаимодействия с сервисом секретов.
type SecretsClient interface {
	// ListSecrets возвращает все секреты текущего пользователя.
	ListSecrets(ctx context.Context) ([]*domain.Secret, error)

	// CreateSecret создаёт новый секрет с заданным blindIndex и зашифрованными данными.
	CreateSecret(ctx context.Context, blindIndex string, data []byte) error

	// UpdateSecret обновляет зашифрованные данные секрета, идентифицируемого по blindIndex.
	UpdateSecret(ctx context.Context, blindIndex string, data []byte) error

	// GetSecret возвращает секрет по blindIndex.
	GetSecret(ctx context.Context, blindIndex string) (*domain.Secret, error)

	// DeleteSecret удаляет секрет по blindIndex.
	DeleteSecret(ctx context.Context, blindIndex string) error
}

type secretsClient struct {
	client pb.SecretsClient
}

// ErrPaginationLoop возвращается, если сервер повторно прислал уже встречавшийся курсор страницы.
var ErrPaginationLoop = errors.New("server returned a repeated page token")

// NewSecretsClient создаёт новый SecretsClient поверх сгенерированного gRPC клиента.
func NewSecretsClient(client pb.SecretsClient) SecretsClient {
	return &secretsClient{client: client}
}

// ListSecrets запрашивает страницы, пока сервер не вернёт пустой курсор.
// Повтор курсора означает ошибку сервера: без проверки клиент зациклился бы.
func (c *secretsClient) ListSecrets(ctx context.Context) ([]*domain.Secret, error) {
	var (
		secrets []*domain.Secret
		token   string
		seen    = make(map[string]bool)
	)
	for {
		req := pb.ListSecretsRequest_builder{PageToken: &token}.Build()
		resp, err := c.client.ListSecrets(ctx, req)
		if err != nil {
			return nil, fromGRPCError(err)
		}
		secrets = append(secrets, fromPBSecrets(resp.GetItems())...)

		token = resp.GetNextPageToken()
		if token == "" {
			return secrets, nil
		}
		if seen[token] {
			return nil, ErrPaginationLoop
		}
		seen[token] = true
	}
}

func (c *secretsClient) CreateSecret(ctx context.Context, blindIndex string, data []byte) error {
	item := pb.SecretData_builder{BlindIndex: &blindIndex, Data: data}.Build()
	req := pb.CreateSecretRequest_builder{Item: item}.Build()
	_, err := c.client.CreateSecret(ctx, req)
	return fromGRPCError(err)
}

func (c *secretsClient) UpdateSecret(ctx context.Context, blindIndex string, data []byte) error {
	item := pb.SecretData_builder{BlindIndex: &blindIndex, Data: data}.Build()
	req := pb.UpdateSecretRequest_builder{Item: item}.Build()
	_, err := c.client.UpdateSecret(ctx, req)
	return fromGRPCError(err)
}

func (c *secretsClient) GetSecret(ctx context.Context, blindIndex string) (*domain.Secret, error) {
	req := pb.GetSecretRequest_builder{BlindIndex: &blindIndex}.Build()
	resp, err := c.client.GetSecret(ctx, req)
	if err != nil {
		return nil, fromGRPCError(err)
	}
	return &domain.Secret{
		Data:      resp.GetData(),
		CreatedAt: resp.GetCreatedAt().AsTime(),
		UpdatedAt: resp.GetUpdatedAt().AsTime(),
	}, nil
}

func (c *secretsClient) DeleteSecret(ctx context.Context, blindIndex string) error {
	req := pb.DeleteSecretRequest_builder{BlindIndex: &blindIndex}.Build()
	_, err := c.client.DeleteSecret(ctx, req)
	return fromGRPCError(err)
}
