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

var testLimits = SecretLimits{MaxSize: 64, MaxCount: 3}

func TestSecretService_Create_EmptyBlindIndex(t *testing.T) {
	svc := NewSecretService(mocks.NewSecretRepository(t), testLimits)
	err := svc.Create(context.Background(), testID, "", testData)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}

func TestSecretService_Create_EmptyData(t *testing.T) {
	svc := NewSecretService(mocks.NewSecretRepository(t), testLimits)
	err := svc.Create(context.Background(), testID, "blindindex", []byte{})
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}

func TestSecretService_Create_AlreadyExists(t *testing.T) {
	mockRepo := mocks.NewSecretRepository(t)
	mockRepo.On("CountByUserID", mock.Anything, testID).Return(0, nil)
	mockRepo.On("Create", mock.Anything, mock.Anything).
		Return(domain.ErrSecretAlreadyExists)

	svc := NewSecretService(mockRepo, testLimits)
	err := svc.Create(context.Background(), testID, "blindindex", testData)
	assert.ErrorIs(t, err, domain.ErrSecretAlreadyExists)
}

func TestSecretService_Create_Success(t *testing.T) {
	mockRepo := mocks.NewSecretRepository(t)
	mockRepo.On("CountByUserID", mock.Anything, testID).Return(testLimits.MaxCount-1, nil)
	mockRepo.On("Create", mock.Anything, &domain.Secret{
		UserID:     testID,
		BlindIndex: "blindindex",
		Data:       testData,
	}).Return(nil)

	svc := NewSecretService(mockRepo, testLimits)
	err := svc.Create(context.Background(), testID, "blindindex", testData)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSecretService_Create_TooLarge(t *testing.T) {
	svc := NewSecretService(mocks.NewSecretRepository(t), testLimits)
	err := svc.Create(context.Background(), testID, "blindindex", make([]byte, testLimits.MaxSize+1))
	assert.ErrorIs(t, err, domain.ErrSecretTooLarge)
}

func TestSecretService_Create_QuotaExceeded(t *testing.T) {
	mockRepo := mocks.NewSecretRepository(t)
	mockRepo.On("CountByUserID", mock.Anything, testID).Return(testLimits.MaxCount, nil)

	svc := NewSecretService(mockRepo, testLimits)
	err := svc.Create(context.Background(), testID, "blindindex", testData)
	assert.ErrorIs(t, err, domain.ErrSecretQuotaExceeded)
}

func TestSecretService_Update_TooLarge(t *testing.T) {
	svc := NewSecretService(mocks.NewSecretRepository(t), testLimits)
	err := svc.Update(context.Background(), testID, "blindindex", make([]byte, testLimits.MaxSize+1))
	assert.ErrorIs(t, err, domain.ErrSecretTooLarge)
}

func TestSecretService_Update_EmptyBlindIndex(t *testing.T) {
	svc := NewSecretService(mocks.NewSecretRepository(t), testLimits)
	err := svc.Update(context.Background(), testID, "", testData)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}

func TestSecretService_Update_EmptyData(t *testing.T) {
	svc := NewSecretService(mocks.NewSecretRepository(t), testLimits)
	err := svc.Update(context.Background(), testID, "blindindex", []byte{})
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}

func TestSecretService_Update_NotFound(t *testing.T) {
	mockRepo := mocks.NewSecretRepository(t)
	mockRepo.On("Update", mock.Anything, mock.Anything).
		Return(domain.ErrSecretNotFound)

	svc := NewSecretService(mockRepo, testLimits)
	err := svc.Update(context.Background(), testID, "blindindex", testData)
	assert.ErrorIs(t, err, domain.ErrSecretNotFound)
}

func TestSecretService_Update_Success(t *testing.T) {
	mockRepo := mocks.NewSecretRepository(t)
	mockRepo.On("Update", mock.Anything, &domain.Secret{
		UserID:     testID,
		BlindIndex: "blindindex",
		Data:       testData,
	}).Return(nil)

	svc := NewSecretService(mockRepo, testLimits)
	err := svc.Update(context.Background(), testID, "blindindex", testData)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSecretService_GetByBlindIndex_EmptyBlindIndex(t *testing.T) {
	svc := NewSecretService(mocks.NewSecretRepository(t), testLimits)
	_, err := svc.GetByBlindIndex(context.Background(), testID, "")
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}

func TestSecretService_GetByBlindIndex_NotFound(t *testing.T) {
	mockRepo := mocks.NewSecretRepository(t)
	mockRepo.On("GetByBlindIndex", mock.Anything, testID, "blindindex").
		Return(nil, domain.ErrSecretNotFound)

	svc := NewSecretService(mockRepo, testLimits)
	_, err := svc.GetByBlindIndex(context.Background(), testID, "blindindex")
	assert.ErrorIs(t, err, domain.ErrSecretNotFound)
}

func TestSecretService_GetByBlindIndex_Success(t *testing.T) {
	want := &domain.Secret{UserID: testID, BlindIndex: "blindindex", Data: testData}
	mockRepo := mocks.NewSecretRepository(t)
	mockRepo.On("GetByBlindIndex", mock.Anything, testID, "blindindex").
		Return(want, nil)

	svc := NewSecretService(mockRepo, testLimits)
	got, err := svc.GetByBlindIndex(context.Background(), testID, "blindindex")
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestSecretService_ListByUserID_Success(t *testing.T) {
	want := []*domain.Secret{
		{UserID: testID, BlindIndex: "a", Data: testData},
		{UserID: testID, BlindIndex: "b", Data: testData},
	}
	mockRepo := mocks.NewSecretRepository(t)
	mockRepo.On("ListByUserID", mock.Anything, testID).Return(want, nil)

	svc := NewSecretService(mockRepo, testLimits)
	got, err := svc.ListByUserID(context.Background(), testID)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestSecretService_ListByUserID_Empty(t *testing.T) {
	mockRepo := mocks.NewSecretRepository(t)
	mockRepo.On("ListByUserID", mock.Anything, testID).Return([]*domain.Secret{}, nil)

	svc := NewSecretService(mockRepo, testLimits)
	got, err := svc.ListByUserID(context.Background(), testID)
	assert.NoError(t, err)
	assert.Empty(t, got)
}

func TestSecretService_Delete_EmptyBlindIndex(t *testing.T) {
	svc := NewSecretService(mocks.NewSecretRepository(t), testLimits)
	err := svc.Delete(context.Background(), testID, "")
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}

func TestSecretService_Delete_NotFound(t *testing.T) {
	mockRepo := mocks.NewSecretRepository(t)
	mockRepo.On("Delete", mock.Anything, testID, "blindindex").
		Return(domain.ErrSecretNotFound)

	svc := NewSecretService(mockRepo, testLimits)
	err := svc.Delete(context.Background(), testID, "blindindex")
	assert.ErrorIs(t, err, domain.ErrSecretNotFound)
}

func TestSecretService_Delete_Success(t *testing.T) {
	mockRepo := mocks.NewSecretRepository(t)
	mockRepo.On("Delete", mock.Anything, testID, "blindindex").Return(nil)

	svc := NewSecretService(mockRepo, testLimits)
	err := svc.Delete(context.Background(), testID, "blindindex")
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
