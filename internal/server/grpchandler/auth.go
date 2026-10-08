// Package grpc содержит gRPC обработчики сервера.
package grpchandler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

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

// changePasswordTimeout ограничивает время смены пароля: транзакция держит блокировку
// строки пользователя, пока клиент передаёт секреты, и не должна висеть бесконечно.
const changePasswordTimeout = 5 * time.Minute

// ChangePassword принимает поток: header, затем перешифрованные секреты, — и передаёт
// секреты сервису по мере получения, не накапливая их в памяти.
func (h *authHandler) ChangePassword(stream pb.Auth_ChangePasswordServer) error {
	ctx, cancel := context.WithTimeout(stream.Context(), changePasswordTimeout)
	defer cancel()

	claims, err := middleware.ClaimsFromContext(ctx)
	if err != nil {
		return err
	}

	first, err := stream.Recv()
	if err != nil {
		return err
	}
	header := first.GetHeader()
	if header == nil {
		return status.Error(codes.InvalidArgument, "first message must be a header")
	}

	// recvErr сохраняет ошибку транспорта, чтобы вернуть её клиенту как есть,
	// а не как внутреннюю ошибку сервиса.
	var recvErr error
	next := func() (*domain.ReencryptedSecret, error) {
		msg, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		if err != nil {
			recvErr = err
			return nil, err
		}
		secret := msg.GetSecret()
		if secret == nil {
			return nil, fmt.Errorf("%w: expected a secret after the header", domain.ErrInvalidArgument)
		}
		return &domain.ReencryptedSecret{
			OldBlindIndex:     secret.GetOldBlindIndex(),
			NewBlindIndex:     secret.GetNewBlindIndex(),
			Data:              secret.GetData(),
			ExpectedUpdatedAt: secret.GetExpectedUpdatedAt().AsTime(),
		}, nil
	}

	token, err := h.authService.ChangePassword(ctx, claims.UserID, domain.PasswordChange{
		OldAuthKey: header.GetOldAuthKey(),
		NewSalt:    header.GetNewSalt(),
		NewKDF:     fromPBKDF(header.GetNewKdf()),
		NewAuthKey: header.GetNewAuthKey(),
	}, next)
	if recvErr != nil {
		return recvErr
	}
	if err != nil {
		return toGRPCError(err)
	}

	return stream.SendAndClose(pb.ChangePasswordResponse_builder{Token: &token}.Build())
}
