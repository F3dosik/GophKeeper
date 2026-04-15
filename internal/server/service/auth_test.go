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
			PasswordHash: "masterkey123",
		}, nil)
	svc := NewAuthService(mockRepo, "jwt-secret")

	token, err := svc.Login(context.Background(), "user", "masterkey123")

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	mockRepo.AssertExpectations(t)
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	mockRepo := mocks.NewUserRepository(t)
	mockRepo.On("GetByLogin", mock.Anything, "user").
		Return(&domain.User{
			PasswordHash: "correcthash",
		}, nil)

	svc := NewAuthService(mockRepo, "jwt-secret")

	_, err := svc.Login(context.Background(), "user", "wronghash")

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	mockRepo.AssertExpectations(t)
}
