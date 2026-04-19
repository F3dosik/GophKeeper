package service

import (
	"context"
	"testing"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/F3dosik/GophKeeper/internal/server/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAuthService_Login_Success(t *testing.T) {
	mockRepo := mocks.NewUserRepository(t)
	mockRepo.On("GetByLogin", mock.Anything, "user").
		Return(&domain.User{
			ID:           uuid.New(),
			PasswordHash: []byte("masterkey123"),
		}, nil)
	svc := NewAuthService(mockRepo, "jwt-secret")

	token, err := svc.Login(context.Background(), "user", []byte("masterkey123"))

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	mockRepo := mocks.NewUserRepository(t)
	mockRepo.On("GetByLogin", mock.Anything, "user").
		Return(&domain.User{
			PasswordHash: []byte("correcthash"),
		}, nil)

	svc := NewAuthService(mockRepo, "jwt-secret")

	_, err := svc.Login(context.Background(), "user", []byte("wronghash"))

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	mockRepo := mocks.NewUserRepository(t)
	mockRepo.On("GetByLogin", mock.Anything, "user").
		Return(nil, domain.ErrUserNotFound)

	svc := NewAuthService(mockRepo, "jwt-secret")

	_, err := svc.Login(context.Background(), "user", []byte("masterkey123"))

	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestAuthService_Create_Success(t *testing.T) {
	mockRepo := mocks.NewUserRepository(t)
	mockRepo.On("Create", mock.Anything, &domain.User{
		Login:        "user",
		PasswordHash: []byte("masterkey"),
		PasswordSalt: []byte("salt"),
	}).Return(nil)

	svc := NewAuthService(mockRepo, "jwt-secret")
	err := svc.Create(context.Background(), "user", []byte("masterkey"), []byte("salt"))

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_Create_AlreadyExists(t *testing.T) {
	mockRepo := mocks.NewUserRepository(t)
	mockRepo.On("Create", mock.Anything, mock.Anything).
		Return(domain.ErrUserAlreadyExists)

	svc := NewAuthService(mockRepo, "jwt-secret")
	err := svc.Create(context.Background(), "user", []byte("masterkey"), []byte("salt"))

	assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)
}

func TestAuthService_GetSalt_Success(t *testing.T) {
	mockRepo := mocks.NewUserRepository(t)
	mockRepo.On("GetByLogin", mock.Anything, "user").
		Return(&domain.User{PasswordSalt: []byte("salt")}, nil)

	svc := NewAuthService(mockRepo, "jwt-secret")
	salt, err := svc.GetSalt(context.Background(), "user")

	assert.NoError(t, err)
	assert.Equal(t, []byte("salt"), salt)
}

func TestAuthService_GetSalt_UserNotFound(t *testing.T) {
	mockRepo := mocks.NewUserRepository(t)
	mockRepo.On("GetByLogin", mock.Anything, "user").
		Return(nil, domain.ErrUserNotFound)

	svc := NewAuthService(mockRepo, "jwt-secret")
	_, err := svc.GetSalt(context.Background(), "user")

	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}
