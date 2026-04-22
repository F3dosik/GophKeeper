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
	testLogin     = "test_user"
	testMasterKey = []byte("secret")
	testWrongKey  = []byte("wrongkey")
	testSalt      = []byte("salt")
)

func TestAuthHandler_CreateUser_Success(t *testing.T) {
	mockService := mocks.NewAuthService(t)
	mockService.On("Create", mock.Anything, testLogin, testMasterKey, testSalt).
		Return(nil)

	handler := NewAuthHandler(mockService)

	req := pb.CreateUserRequest_builder{
		Credentials: pb.Credentials_builder{
			Login:     &testLogin,
			MasterKey: testMasterKey,
		}.Build(),
		Salt: testSalt,
	}.Build()

	resp, err := handler.CreateUser(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, pb.CreateUserResponse_builder{}.Build(), resp)
	mockService.AssertExpectations(t)
}

func TestAuthHandler_CreateUser_UserAlreadyExists(t *testing.T) {
	mockService := mocks.NewAuthService(t)
	mockService.On("Create", mock.Anything, testLogin, testMasterKey, testSalt).
		Return(domain.ErrUserAlreadyExists)

	handler := NewAuthHandler(mockService)

	req := pb.CreateUserRequest_builder{
		Credentials: pb.Credentials_builder{
			Login:     &testLogin,
			MasterKey: testMasterKey,
		}.Build(),
		Salt: testSalt,
	}.Build()

	_, err := handler.CreateUser(context.Background(), req)

	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	mockService.AssertExpectations(t)
}

func TestAuthHandler_GetSalt_Success(t *testing.T) {
	mockService := mocks.NewAuthService(t)
	mockService.On("GetSalt", mock.Anything, testLogin).
		Return(testSalt, nil)

	handler := NewAuthHandler(mockService)

	req := pb.GetSaltRequest_builder{
		Login: &testLogin,
	}.Build()

	resp, err := handler.GetSalt(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, testSalt, resp.GetSalt())
	mockService.AssertExpectations(t)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	mockService := mocks.NewAuthService(t)
	mockService.On("Login", mock.Anything, testLogin, testMasterKey).
		Return("jwt-token", nil)

	handler := NewAuthHandler(mockService)

	req := pb.LoginRequest_builder{
		Credentials: pb.Credentials_builder{
			Login:     proto.String(testLogin),
			MasterKey: testMasterKey,
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
			Login:     proto.String(testLogin),
			MasterKey: testWrongKey,
		}.Build(),
	}.Build()

	_, err := handler.Login(context.Background(), req)

	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	mockService.AssertExpectations(t)
}
