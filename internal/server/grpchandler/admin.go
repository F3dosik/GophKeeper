package grpchandler

import (
	"context"

	"github.com/F3dosik/GophKeeper/internal/server/service"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
)

// adminHandler реализует pb.AdminServer. Регистрируется только на административном
// сервере: методы не проверяют токен, доступ ограничен тем, что порт открыт лишь на localhost.
type adminHandler struct {
	pb.UnimplementedAdminServer
	authService service.AuthService
}

// NewAdminHandler создаёт обработчик административного сервиса.
func NewAdminHandler(authService service.AuthService) pb.AdminServer {
	return &adminHandler{authService: authService}
}

// DeleteUser удаляет пользователя по логину вместе со всеми секретами.
// Возвращает codes.NotFound, если пользователя нет.
func (h *adminHandler) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	if err := h.authService.DeleteUser(ctx, req.GetLogin()); err != nil {
		return nil, toGRPCError(err)
	}
	return pb.DeleteUserResponse_builder{}.Build(), nil
}
