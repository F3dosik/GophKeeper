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
	"google.golang.org/protobuf/types/known/timestamppb"
)

// authHandler реализует интерфейс pb.AuthServer.
// Обрабатывает запросы аутентификации и регистрации пользователей.
type authHandler struct {
	pb.UnimplementedAuthServer
	authService service.AuthService
	opts        AuthHandlerOptions
}

// AuthHandlerOptions задаёт, какие регистрации принимает обработчик. Публичный и
// административный порты сервера используют разные настройки.
type AuthHandlerOptions struct {
	// AllowRegistration разрешает CreateUser.
	AllowRegistration bool
	// AllowTemporary разрешает регистрацию с временным паролем. Включается только на
	// административном порту, доступном с самой машины сервера.
	AllowTemporary bool
}

// NewAuthHandler создаёт новый экземпляр обработчика аутентификации.
// Возвращает pb.AuthServer, чтобы тип был именуемым вне пакета и легко подменялся
// в тестах и при регистрации в gRPC-сервере.
func NewAuthHandler(authService service.AuthService, opts AuthHandlerOptions) pb.AuthServer {
	return &authHandler{authService: authService, opts: opts}
}

// CreateUser обрабатывает запрос регистрации нового пользователя.
// Возвращает codes.AlreadyExists если пользователь с таким логином уже существует
// и codes.PermissionDenied если регистрация (или временный пароль) здесь не разрешена.
func (h *authHandler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	if !h.opts.AllowRegistration {
		return nil, toGRPCError(domain.ErrRegistrationDisabled)
	}
	if req.GetTemporary() && !h.opts.AllowTemporary {
		return nil, toGRPCError(fmt.Errorf("%w: temporary passwords are only allowed on the admin port",
			domain.ErrRegistrationDisabled))
	}
	if !req.HasKdf() {
		return nil, status.Error(codes.InvalidArgument, "kdf params are required, update the client")
	}

	expiresAt, err := h.authService.Create(ctx, domain.Registration{
		Login:     req.GetCredentials().GetLogin(),
		AuthKey:   req.GetCredentials().GetAuthKey(),
		Salt:      req.GetSalt(),
		KDF:       fromPBKDF(req.GetKdf()),
		Temporary: req.GetTemporary(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}

	resp := pb.CreateUserResponse_builder{}
	if expiresAt != nil {
		resp.TemporaryExpiresAt = timestamppb.New(*expiresAt)
	}
	return resp.Build(), nil
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
	result, err := h.authService.Login(
		ctx, req.GetCredentials().GetLogin(),
		req.GetCredentials().GetAuthKey(),
	)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return pb.LoginResponse_builder{
		Token:                  &result.Token,
		PasswordChangeRequired: &result.PasswordChangeRequired,
	}.Build(), nil
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

// DeleteAccount удаляет учётку текущего пользователя. Кроме токена требует ключ
// аутентификации от пароля, чтобы украденный токен не позволял удалить учётку.
func (h *authHandler) DeleteAccount(ctx context.Context, req *pb.DeleteAccountRequest) (*pb.DeleteAccountResponse, error) {
	claims, err := middleware.ClaimsFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.authService.DeleteAccount(ctx, claims.UserID, req.GetAuthKey()); err != nil {
		return nil, toGRPCError(err)
	}
	return pb.DeleteAccountResponse_builder{}.Build(), nil
}
