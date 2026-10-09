package grpchandler

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/F3dosik/GophKeeper/internal/server/mocks"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var (
	testLogin    = "test_user"
	testAuthKey  = []byte("secret")
	testWrongKey = []byte("wrongkey")
	testSalt     = []byte("salt")
)

func TestAuthHandler_CreateUser_Success(t *testing.T) {
	mockService := mocks.NewAuthService(t)
	mockService.On("Create", mock.Anything, testRegistration).
		Return(nil, nil)

	handler := NewAuthHandler(mockService, openRegistration)

	req := pb.CreateUserRequest_builder{
		Credentials: pb.Credentials_builder{
			Login:   &testLogin,
			AuthKey: testAuthKey,
		}.Build(),
		Salt: testSalt,
		Kdf:  toPBKDF(domain.DefaultKDFParams),
	}.Build()

	resp, err := handler.CreateUser(context.Background(), req)

	assert.NoError(t, err)
	assert.False(t, resp.HasTemporaryExpiresAt())
	mockService.AssertExpectations(t)
}

func TestAuthHandler_CreateUser_UserAlreadyExists(t *testing.T) {
	mockService := mocks.NewAuthService(t)
	mockService.On("Create", mock.Anything, testRegistration).
		Return(nil, domain.ErrUserAlreadyExists)

	handler := NewAuthHandler(mockService, openRegistration)

	req := pb.CreateUserRequest_builder{
		Credentials: pb.Credentials_builder{
			Login:   &testLogin,
			AuthKey: testAuthKey,
		}.Build(),
		Salt: testSalt,
		Kdf:  toPBKDF(domain.DefaultKDFParams),
	}.Build()

	_, err := handler.CreateUser(context.Background(), req)

	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	mockService.AssertExpectations(t)
}

func TestAuthHandler_GetSalt_Success(t *testing.T) {
	mockService := mocks.NewAuthService(t)
	mockService.On("GetSalt", mock.Anything, testLogin).
		Return(testSalt, domain.DefaultKDFParams, nil)

	handler := NewAuthHandler(mockService, openRegistration)

	req := pb.GetSaltRequest_builder{
		Login: &testLogin,
	}.Build()

	resp, err := handler.GetSalt(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, testSalt, resp.GetSalt())
	assert.Equal(t, domain.DefaultKDFParams, fromPBKDF(resp.GetKdf()))
	mockService.AssertExpectations(t)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	mockService := mocks.NewAuthService(t)
	mockService.On("Login", mock.Anything, testLogin, testAuthKey).
		Return(domain.LoginResult{Token: "jwt-token"}, nil)

	handler := NewAuthHandler(mockService, openRegistration)

	req := pb.LoginRequest_builder{
		Credentials: pb.Credentials_builder{
			Login:   proto.String(testLogin),
			AuthKey: testAuthKey,
		}.Build(),
	}.Build()

	resp, err := handler.Login(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, "jwt-token", resp.GetToken())
	mockService.AssertExpectations(t)
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	mockService := mocks.NewAuthService(t)
	mockService.On("Login", mock.Anything, testLogin, testWrongKey).
		Return(domain.LoginResult{}, domain.ErrInvalidCredentials)

	handler := NewAuthHandler(mockService, openRegistration)

	req := pb.LoginRequest_builder{
		Credentials: pb.Credentials_builder{
			Login:   proto.String(testLogin),
			AuthKey: testWrongKey,
		}.Build(),
	}.Build()

	_, err := handler.Login(context.Background(), req)

	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	mockService.AssertExpectations(t)
}

// Старый клиент без параметров Argon2id получает понятную ошибку, а не регистрацию
// с неизвестными параметрами.
func TestAuthHandler_CreateUser_RequiresKDF(t *testing.T) {
	handler := NewAuthHandler(mocks.NewAuthService(t), openRegistration)

	req := pb.CreateUserRequest_builder{
		Credentials: pb.Credentials_builder{Login: &testLogin, AuthKey: testAuthKey}.Build(),
		Salt:        testSalt,
	}.Build()

	_, err := handler.CreateUser(context.Background(), req)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

// openRegistration — настройки публичного порта с открытой регистрацией.
var openRegistration = AuthHandlerOptions{AllowRegistration: true}

// testRegistration — регистрация, которую собирает хендлер из тестового запроса.
var testRegistration = domain.Registration{
	Login: testLogin, AuthKey: testAuthKey, Salt: testSalt, KDF: domain.DefaultKDFParams,
}

// createUserRequest собирает корректный запрос регистрации.
func createUserRequest(temporary bool) *pb.CreateUserRequest {
	return pb.CreateUserRequest_builder{
		Credentials: pb.Credentials_builder{Login: &testLogin, AuthKey: testAuthKey}.Build(),
		Salt:        testSalt,
		Kdf:         toPBKDF(domain.DefaultKDFParams),
		Temporary:   &temporary,
	}.Build()
}

func TestAuthHandler_CreateUser_RegistrationDisabled(t *testing.T) {
	handler := NewAuthHandler(mocks.NewAuthService(t), AuthHandlerOptions{})

	_, err := handler.CreateUser(context.Background(), createUserRequest(false))
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestAuthHandler_CreateUser_TemporaryOnlyOnAdminPort(t *testing.T) {
	handler := NewAuthHandler(mocks.NewAuthService(t), openRegistration)

	_, err := handler.CreateUser(context.Background(), createUserRequest(true))
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestAuthHandler_CreateUser_TemporaryOnAdminPort(t *testing.T) {
	expiresAt := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	reg := testRegistration
	reg.Temporary = true

	mockService := mocks.NewAuthService(t)
	mockService.On("Create", mock.Anything, reg).Return(&expiresAt, nil)
	handler := NewAuthHandler(mockService, AuthHandlerOptions{AllowRegistration: true, AllowTemporary: true})

	resp, err := handler.CreateUser(context.Background(), createUserRequest(true))
	require.NoError(t, err)
	assert.Equal(t, expiresAt, resp.GetTemporaryExpiresAt().AsTime())
}

func TestAuthHandler_Login_PasswordChangeRequired(t *testing.T) {
	mockService := mocks.NewAuthService(t)
	mockService.On("Login", mock.Anything, testLogin, testAuthKey).
		Return(domain.LoginResult{Token: "restricted", PasswordChangeRequired: true}, nil)
	handler := NewAuthHandler(mockService, openRegistration)

	req := pb.LoginRequest_builder{
		Credentials: pb.Credentials_builder{Login: &testLogin, AuthKey: testAuthKey}.Build(),
	}.Build()
	resp, err := handler.Login(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.GetPasswordChangeRequired())
}

func TestAuthHandler_DeleteAccount(t *testing.T) {
	mockService := mocks.NewAuthService(t)
	mockService.On("DeleteAccount", mock.Anything, testUserID, testAuthKey).Return(nil)
	handler := NewAuthHandler(mockService, openRegistration)

	_, err := handler.DeleteAccount(claimsContext(), pb.DeleteAccountRequest_builder{AuthKey: testAuthKey}.Build())
	assert.NoError(t, err)

	_, err = handler.DeleteAccount(context.Background(), pb.DeleteAccountRequest_builder{AuthKey: testAuthKey}.Build())
	assert.Equal(t, codes.Unauthenticated, status.Code(err), "requires an authenticated request")
}
