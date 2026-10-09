package grpchandler

import (
	"context"
	"time"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/F3dosik/GophKeeper/internal/server/service"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// adminHandler реализует pb.AdminServer. Регистрируется только на административном
// сервере: методы не проверяют токен, доступ ограничен тем, что порт открыт лишь на localhost.
type adminHandler struct {
	pb.UnimplementedAdminServer
	adminService service.AdminService
}

// NewAdminHandler создаёт обработчик административного сервиса.
func NewAdminHandler(adminService service.AdminService) pb.AdminServer {
	return &adminHandler{adminService: adminService}
}

// DeleteUser удаляет пользователя по логину вместе со всеми секретами.
// Возвращает codes.NotFound, если пользователя нет.
func (h *adminHandler) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	if err := h.adminService.DeleteUser(ctx, req.GetLogin()); err != nil {
		return nil, toGRPCError(err)
	}
	return pb.DeleteUserResponse_builder{}.Build(), nil
}

// ListUsers возвращает страницу пользователей.
func (h *adminHandler) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	page, err := h.adminService.ListUsers(ctx, req.GetPageToken(), int(req.GetPageSize()))
	if err != nil {
		return nil, toGRPCError(err)
	}
	users := make([]*pb.UserInfo, 0, len(page.Users))
	for _, u := range page.Users {
		users = append(users, toPBUserInfo(u))
	}
	return pb.ListUsersResponse_builder{Users: users, NextPageToken: &page.NextPageToken}.Build(), nil
}

// GetUser возвращает сведения о пользователе.
func (h *adminHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	user, err := h.adminService.GetUser(ctx, req.GetLogin())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return pb.GetUserResponse_builder{User: toPBUserInfo(user)}.Build(), nil
}

// RevokeSessions отзывает все токены пользователя.
func (h *adminHandler) RevokeSessions(ctx context.Context, req *pb.RevokeSessionsRequest) (*pb.RevokeSessionsResponse, error) {
	if err := h.adminService.RevokeSessions(ctx, req.GetLogin()); err != nil {
		return nil, toGRPCError(err)
	}
	return pb.RevokeSessionsResponse_builder{}.Build(), nil
}

// toPBUserInfo переводит сведения о пользователе в protobuf.
func toPBUserInfo(u *domain.UserInfo) *pb.UserInfo {
	count := int32(u.SecretCount)
	b := pb.UserInfo_builder{
		Login:        &u.Login,
		CreatedAt:    timestamppb.New(u.CreatedAt),
		SecretCount:  &count,
		KdfTime:      &u.KDF.Time,
		KdfMemoryKib: &u.KDF.MemoryKiB,
	}
	if u.PasswordExpiresAt != nil {
		b.TemporaryPasswordExpiresAt = timestamppb.New(u.PasswordExpiresAt.In(time.UTC))
	}
	return b.Build()
}
