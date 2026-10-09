package grpcclient

import (
	"context"

	"github.com/F3dosik/GophKeeper/internal/domain"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
)

// AdminClient определяет интерфейс административного сервиса. Работает только при
// подключении к административному порту сервера (ADMIN_PORT).
type AdminClient interface {
	// DeleteUser удаляет пользователя по логину вместе со всеми секретами.
	DeleteUser(ctx context.Context, login string) error

	// ListUsers возвращает всех пользователей по возрастанию логина (запрашивая страницы).
	ListUsers(ctx context.Context) ([]*domain.UserInfo, error)

	// GetUser возвращает сведения о пользователе.
	GetUser(ctx context.Context, login string) (*domain.UserInfo, error)

	// RevokeSessions отзывает все токены пользователя.
	RevokeSessions(ctx context.Context, login string) error
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

func (c *adminClient) ListUsers(ctx context.Context) ([]*domain.UserInfo, error) {
	var (
		users []*domain.UserInfo
		token string
		seen  = make(map[string]bool)
	)
	for {
		resp, err := c.client.ListUsers(ctx, pb.ListUsersRequest_builder{PageToken: &token}.Build())
		if err != nil {
			return nil, fromGRPCError(err)
		}
		for _, u := range resp.GetUsers() {
			users = append(users, fromPBUserInfo(u))
		}
		token = resp.GetNextPageToken()
		if token == "" {
			return users, nil
		}
		if seen[token] {
			return nil, ErrPaginationLoop
		}
		seen[token] = true
	}
}

func (c *adminClient) GetUser(ctx context.Context, login string) (*domain.UserInfo, error) {
	resp, err := c.client.GetUser(ctx, pb.GetUserRequest_builder{Login: &login}.Build())
	if err != nil {
		return nil, fromGRPCError(err)
	}
	return fromPBUserInfo(resp.GetUser()), nil
}

func (c *adminClient) RevokeSessions(ctx context.Context, login string) error {
	_, err := c.client.RevokeSessions(ctx, pb.RevokeSessionsRequest_builder{Login: &login}.Build())
	return fromGRPCError(err)
}

// fromPBUserInfo переводит сведения о пользователе из protobuf.
func fromPBUserInfo(u *pb.UserInfo) *domain.UserInfo {
	info := &domain.UserInfo{
		Login:       u.GetLogin(),
		CreatedAt:   u.GetCreatedAt().AsTime(),
		SecretCount: int(u.GetSecretCount()),
		KDF:         domain.KDFParams{Time: u.GetKdfTime(), MemoryKiB: u.GetKdfMemoryKib()},
	}
	if u.HasTemporaryPasswordExpiresAt() {
		expires := u.GetTemporaryPasswordExpiresAt().AsTime()
		info.PasswordExpiresAt = &expires
	}
	return info
}
