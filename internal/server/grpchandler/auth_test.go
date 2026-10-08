package grpchandler

import (
	"context"
	"testing"

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
	mockService.On("Create", mock.Anything, testLogin, testAuthKey, testSalt, domain.DefaultKDFParams).
		Return(nil)

	handler := NewAuthHandler(mockService)

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
	assert.Equal(t, pb.CreateUserResponse_builder{}.Build(), resp)
	mockService.AssertExpectations(t)
}

func TestAuthHandler_CreateUser_UserAlreadyExists(t *testing.T) {
	mockService := mocks.NewAuthService(t)
	mockService.On("Create", mock.Anything, testLogin, testAuthKey, testSalt, domain.DefaultKDFParams).
		Return(domain.ErrUserAlreadyExists)

	handler := NewAuthHandler(mockService)

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

	handler := NewAuthHandler(mockService)

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
		Return("jwt-token", nil)

	handler := NewAuthHandler(mockService)

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
		Return("", domain.ErrInvalidCredentials)

	handler := NewAuthHandler(mockService)

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
	handler := NewAuthHandler(mocks.NewAuthService(t))

	req := pb.CreateUserRequest_builder{
		Credentials: pb.Credentials_builder{Login: &testLogin, AuthKey: testAuthKey}.Build(),
		Salt:        testSalt,
	}.Build()

	_, err := handler.CreateUser(context.Background(), req)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}
