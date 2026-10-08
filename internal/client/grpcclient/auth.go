package grpcclient

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/F3dosik/GophKeeper/internal/domain"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AuthClient определяет интерфейс для взаимодействия с сервисом аутентификации.
type AuthClient interface {
	// CreateUser регистрирует нового пользователя на сервере.
	// salt должен быть сгенерирован клиентом перед деривацией ключей.
	// kdf — параметры Argon2id, с которыми выведен ключ аутентификации.
	// temporary — пароль временный (только на административном порту); тогда
	// возвращается время, до которого им можно войти.
	CreateUser(ctx context.Context, creds domain.Credentials, salt []byte, kdf domain.KDFParams, temporary bool) (*time.Time, error)

	// GetSalt возвращает соль пользователя по логину.
	// Используется для деривации ключей перед аутентификацией.
	// Вместе с солью возвращает параметры Argon2id пользователя.
	GetSalt(ctx context.Context, login string) ([]byte, domain.KDFParams, error)

	// Login аутентифицирует пользователя и возвращает JWT токен. Для временного пароля
	// возвращает passwordChangeRequired == true и токен, пригодный только для смены пароля.
	Login(ctx context.Context, creds domain.Credentials) (token string, passwordChangeRequired bool, err error)

	// Logout отзывает на сервере текущий токен или, если allSessions == true,
	// все токены пользователя.
	Logout(ctx context.Context, allSessions bool) error

	// ChangePassword передаёт серверу новые учётные данные и все перешифрованные
	// секреты одним потоком и возвращает новый токен.
	ChangePassword(ctx context.Context, change domain.PasswordChange, secrets []domain.ReencryptedSecret) (string, error)

	// DeleteAccount удаляет учётку текущего пользователя; authKey подтверждает пароль.
	DeleteAccount(ctx context.Context, authKey []byte) error
}

type authClient struct {
	client pb.AuthClient
}

// NewAuthClient создаёт новый AuthClient поверх сгенерированного gRPC клиента.
func NewAuthClient(client pb.AuthClient) AuthClient {
	return &authClient{client: client}
}

func (c *authClient) CreateUser(
	ctx context.Context, creds domain.Credentials, salt []byte, kdf domain.KDFParams, temporary bool,
) (*time.Time, error) {
	req := pb.CreateUserRequest_builder{
		Credentials: toPBCredentials(creds),
		Salt:        salt,
		Kdf:         toPBKDF(kdf),
		Temporary:   &temporary,
	}.Build()
	resp, err := c.client.CreateUser(ctx, req)
	if err != nil {
		return nil, fromGRPCError(err)
	}
	if !resp.HasTemporaryExpiresAt() {
		return nil, nil
	}
	expiresAt := resp.GetTemporaryExpiresAt().AsTime()
	return &expiresAt, nil
}

func (c *authClient) GetSalt(ctx context.Context, login string) ([]byte, domain.KDFParams, error) {
	req := pb.GetSaltRequest_builder{Login: &login}.Build()
	resp, err := c.client.GetSalt(ctx, req)
	if err != nil {
		return nil, domain.KDFParams{}, fromGRPCError(err)
	}
	return resp.GetSalt(), fromPBKDF(resp.GetKdf()), nil
}

func (c *authClient) Login(ctx context.Context, creds domain.Credentials) (string, bool, error) {
	req := pb.LoginRequest_builder{
		Credentials: toPBCredentials(creds),
	}.Build()
	resp, err := c.client.Login(ctx, req)
	if err != nil {
		return "", false, fromGRPCError(err)
	}
	return resp.GetToken(), resp.GetPasswordChangeRequired(), nil
}

func (c *authClient) Logout(ctx context.Context, allSessions bool) error {
	req := pb.LogoutRequest_builder{AllSessions: &allSessions}.Build()
	_, err := c.client.Logout(ctx, req)
	return fromGRPCError(err)
}

func (c *authClient) ChangePassword(
	ctx context.Context, change domain.PasswordChange, secrets []domain.ReencryptedSecret,
) (string, error) {
	stream, err := c.client.ChangePassword(ctx)
	if err != nil {
		return "", fromGRPCError(err)
	}

	header := pb.ChangePasswordHeader_builder{
		OldAuthKey: change.OldAuthKey,
		NewSalt:    change.NewSalt,
		NewAuthKey: change.NewAuthKey,
		NewKdf:     toPBKDF(change.NewKDF),
	}.Build()
	if err := stream.Send(pb.ChangePasswordRequest_builder{Header: header}.Build()); err != nil {
		return "", c.streamError(stream, err)
	}

	for _, secret := range secrets {
		item := pb.ReencryptedSecret_builder{
			OldBlindIndex:     &secret.OldBlindIndex,
			NewBlindIndex:     &secret.NewBlindIndex,
			Data:              secret.Data,
			ExpectedUpdatedAt: timestamppb.New(secret.ExpectedUpdatedAt),
		}.Build()
		if err := stream.Send(pb.ChangePasswordRequest_builder{Secret: item}.Build()); err != nil {
			return "", c.streamError(stream, err)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return "", fromGRPCError(err)
	}
	return resp.GetToken(), nil
}

// streamError возвращает настоящую причину ошибки отправки в клиентский поток.
// Если сервер завершил поток с ошибкой, Send возвращает io.EOF, а сам статус
// доступен только через CloseAndRecv.
func (c *authClient) streamError(
	stream grpc.ClientStreamingClient[pb.ChangePasswordRequest, pb.ChangePasswordResponse], err error,
) error {
	if errors.Is(err, io.EOF) {
		_, err = stream.CloseAndRecv()
	}
	return fromGRPCError(err)
}

func (c *authClient) DeleteAccount(ctx context.Context, authKey []byte) error {
	_, err := c.client.DeleteAccount(ctx, pb.DeleteAccountRequest_builder{AuthKey: authKey}.Build())
	return fromGRPCError(err)
}
