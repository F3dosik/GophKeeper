// Package grpc содержит gRPC обработчики сервера.
package grpchandler

import (
	"context"

	"github.com/F3dosik/GophKeeper/internal/server/service"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
)

// authHandler реализует интерфейс pb.AuthServer.
// Обрабатывает запросы аутентификации и регистрации пользователей.
type authHandler struct {
	pb.UnimplementedAuthServer
	authService service.AuthService
}

// NewAuthHandler создаёт новый экземпляр authHandler.
func NewAuthHandler(authService service.AuthService) *authHandler {
	return &authHandler{authService: authService}
}

// CreateUser обрабатывает запрос регистрации нового пользователя.
// Возвращает codes.AlreadyExists если пользователь с таким логином уже существует.
func (h *authHandler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	if err := h.authService.Create(
		ctx, req.GetCredentials().GetLogin(),
		req.GetCredentials().GetMasterKey(), req.GetSalt(),
	); err != nil {
		return nil, toGRPCError(err)
	}
	return pb.CreateUserResponse_builder{}.Build(), nil
}

// GetSalt обрабатывает запрос получения соли пользователя по логину.
// Возвращает codes.NotFound если пользователь не найден.
func (h *authHandler) GetSalt(ctx context.Context, req *pb.GetSaltRequest) (*pb.GetSaltResponse, error) {
	salt, err := h.authService.GetSalt(ctx, req.GetLogin())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return pb.GetSaltResponse_builder{Salt: salt}.Build(), nil
}

// Login обрабатывает запрос аутентификации пользователя.
// Возвращает codes.NotFound если пользователь не найден.
// Возвращает codes.Unauthenticated если masterKey неверный.
func (h *authHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	token, err := h.authService.Login(
		ctx, req.GetCredentials().GetLogin(),
		req.GetCredentials().GetMasterKey(),
	)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return pb.LoginResponse_builder{Token: &token}.Build(), nil
}
