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

var (
	testID   = uuid.New()
	testData = []byte("Secret")
)

func TestSecretService_Create_EmptyBlindIndex(t *testing.T) {
	svc := NewSecretService(mocks.NewSecretRepository(t))
	err := svc.Create(context.Background(), testID, "", testData)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}

func TestSecretService_Create_EmptyData(t *testing.T) {
	svc := NewSecretService(mocks.NewSecretRepository(t))
	err := svc.Create(context.Background(), testID, "blindindex", []byte{})
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}

func TestSecretService_Create_AlreadyExists(t *testing.T) {
	mockRepo := mocks.NewSecretRepository(t)
	mockRepo.On("Create", mock.Anything, mock.Anything).
		Return(domain.ErrSecretAlreadyExists)

	svc := NewSecretService(mockRepo)
	err := svc.Create(context.Background(), testID, "blindindex", testData)
	assert.ErrorIs(t, err, domain.ErrSecretAlreadyExists)
}

func TestSecretService_Create_Success(t *testing.T) {
	mockRepo := mocks.NewSecretRepository(t)
	mockRepo.On("Create", mock.Anything, &domain.Secret{
		UserID:     testID,
		BlindIndex: "blindindex",
		Data:       testData,
	}).Return(nil)

	svc := NewSecretService(mockRepo)
	err := svc.Create(context.Background(), testID, "blindindex", testData)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
