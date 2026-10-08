// Package grpc содержит gRPC обработчики сервера.
package grpchandler

import (
	"context"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/F3dosik/GophKeeper/internal/server/middleware"
	"github.com/F3dosik/GophKeeper/internal/server/service"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// authHandler реализует интерфейс pb.AuthServer.
// Обрабатывает запросы аутентификации и регистрации пользователей.
type authHandler struct {
	pb.UnimplementedAuthServer
	authService service.AuthService
}

// NewAuthHandler создаёт новый экземпляр обработчика аутентификации.
// Возвращает pb.AuthServer, чтобы тип был именуемым вне пакета и легко подменялся
// в тестах и при регистрации в gRPC-сервере.
func NewAuthHandler(authService service.AuthService) pb.AuthServer {
	return &authHandler{authService: authService}
}

// CreateUser обрабатывает запрос регистрации нового пользователя.
// Возвращает codes.AlreadyExists если пользователь с таким логином уже существует.
func (h *authHandler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	if !req.HasKdf() {
		return nil, status.Error(codes.InvalidArgument, "kdf params are required, update the client")
	}
	if err := h.authService.Create(
		ctx, req.GetCredentials().GetLogin(),
		req.GetCredentials().GetAuthKey(), req.GetSalt(), fromPBKDF(req.GetKdf()),
	); err != nil {
		return nil, toGRPCError(err)
	}
	return pb.CreateUserResponse_builder{}.Build(), nil
}

// GetSalt обрабатывает запрос получения соли пользователя по логину.
// Для несуществующего логина возвращает детерминированную фиктивную соль,
// чтобы не раскрывать факт наличия пользователя (защита от перечисления).
func (h *authHandler) GetSalt(ctx context.Context, req *pb.GetSaltRequest) (*pb.GetSaltResponse, error) {
	salt, kdf, err := h.authService.GetSalt(ctx, req.GetLogin())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return pb.GetSaltResponse_builder{Salt: salt, Kdf: toPBKDF(kdf)}.Build(), nil
}

// Login обрабатывает запрос аутентификации пользователя.
// Возвращает codes.Unauthenticated если authKey неверный или логин не существует
// (единый код ответа скрывает факт наличия пользователя).
func (h *authHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	token, err := h.authService.Login(
		ctx, req.GetCredentials().GetLogin(),
		req.GetCredentials().GetAuthKey(),
	)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return pb.LoginResponse_builder{Token: &token}.Build(), nil
}

// Logout отзывает токен, с которым выполнен запрос, или все токены пользователя.
// Метод не публичный: AuthInterceptor уже проверил токен и положил claims в контекст.
func (h *authHandler) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	claims, err := middleware.ClaimsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.authService.Logout(
		ctx, claims.UserID, claims.TokenID, claims.ExpiresAt.Time, req.GetAllSessions(),
	); err != nil {
		return nil, toGRPCError(err)
	}
	return pb.LogoutResponse_builder{}.Build(), nil
}

// fromPBKDF переводит параметры Argon2id из protobuf. Значения вне диапазона uint8
// для потоков насыщаются, чтобы их отклонила проверка KDFParams.Validate.
func fromPBKDF(p *pb.KDFParams) domain.KDFParams {
	threads := p.GetThreads()
	if threads > 255 {
		threads = 255
	}
	return domain.KDFParams{Time: p.GetTime(), MemoryKiB: p.GetMemoryKib(), Threads: uint8(threads)}
}

// toPBKDF переводит параметры Argon2id в protobuf.
func toPBKDF(p domain.KDFParams) *pb.KDFParams {
	threads := uint32(p.Threads)
	return pb.KDFParams_builder{Time: &p.Time, MemoryKib: &p.MemoryKiB, Threads: &threads}.Build()
}
