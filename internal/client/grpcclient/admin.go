package grpcclient

import (
	"context"

	pb "github.com/F3dosik/GophKeeper/proto/gen"
)

// AdminClient определяет интерфейс административного сервиса. Работает только при
// подключении к административному порту сервера (ADMIN_PORT).
type AdminClient interface {
	// DeleteUser удаляет пользователя по логину вместе со всеми секретами.
	DeleteUser(ctx context.Context, login string) error
}

type adminClient struct {
	client pb.AdminClient
}

// NewAdminClient создаёт AdminClient поверх сгенерированного gRPC клиента.
func NewAdminClient(client pb.AdminClient) AdminClient {
	return &adminClient{client: client}
}

func (c *adminClient) DeleteUser(ctx context.Context, login string) error {
	_, err := c.client.DeleteUser(ctx, pb.DeleteUserRequest_builder{Login: &login}.Build())
	return fromGRPCError(err)
}
