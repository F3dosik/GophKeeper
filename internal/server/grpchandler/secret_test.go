package grpchandler

import (
	"context"
	"testing"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/F3dosik/GophKeeper/internal/server/middleware"
	"github.com/F3dosik/GophKeeper/internal/server/mocks"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	testBlindIndex = "blindIndex"
	testData       = []byte("data")
	emptyData      = []byte{}
	testUserID     = uuid.New()
	testSecret     = &domain.Secret{
		ID:         uuid.New(),
		UserID:     testUserID,
		BlindIndex: testBlindIndex,
		Data:       testData,
	}
)

func TestSecretHandler_CreateSecret_Success(t *testing.T) {
	mockService := mocks.NewSecretService(t)
	mockService.On("Create", mock.Anything, testUserID, testBlindIndex, testData).
		Return(nil)

	handler := NewSecretHandler(mockService)

	req := pb.CreateSecretRequest_builder{
		Item: pb.SecretItem_builder{
			BlindIndex: &testBlindIndex,
			Data:       testData,
		}.Build(),
	}.Build()

	_, err := handler.CreateSecret(middleware.WithUserID(context.Background(), testUserID), req)
	assert.NoError(t, err)
	mockService.AssertExpectations(t)
}

func TestSecretHandler_CreateSecret_BlindIndexAlreadyExist(t *testing.T) {
	mockService := mocks.NewSecretService(t)
	mockService.On("Create", mock.Anything, testUserID, testBlindIndex, testData).
		Return(domain.ErrSecretAlreadyExists)

	handler := NewSecretHandler(mockService)

	req := pb.CreateSecretRequest_builder{
		Item: pb.SecretItem_builder{
			BlindIndex: &testBlindIndex,
			Data:       testData,
		}.Build(),
	}.Build()

	_, err := handler.CreateSecret(middleware.WithUserID(context.Background(), testUserID), req)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	mockService.AssertExpectations(t)

}

func TestSecretHandler_CreateSecret_InvalidArgument(t *testing.T) {
	mockService := mocks.NewSecretService(t)
	mockService.On("Create", mock.Anything, testUserID, testBlindIndex, emptyData).
		Return(domain.ErrInvalidArgument)

	handler := NewSecretHandler(mockService)

	req := pb.CreateSecretRequest_builder{
		Item: pb.SecretItem_builder{
			BlindIndex: &testBlindIndex,
			Data:       emptyData,
		}.Build(),
	}.Build()

	_, err := handler.CreateSecret(middleware.WithUserID(context.Background(), testUserID), req)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	mockService.AssertExpectations(t)
}

func TestSecretHandler_CreateSecret_Unauthenticated(t *testing.T) {
	mockService := mocks.NewSecretService(t)
	handler := NewSecretHandler(mockService)

	req := pb.CreateSecretRequest_builder{
		Item: pb.SecretItem_builder{
			BlindIndex: &testBlindIndex,
			Data:       testData,
		}.Build(),
	}.Build()

	_, err := handler.CreateSecret(context.Background(), req)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	mockService.AssertExpectations(t)
}

func TestSecretHandler_UpdateSecret_Success(t *testing.T) {
	mockService := mocks.NewSecretService(t)
	mockService.On("Update", mock.Anything, testUserID, testBlindIndex, testData).
		Return(nil)

	handler := NewSecretHandler(mockService)

	req := pb.UpdateSecretRequest_builder{
		Item: pb.SecretItem_builder{
			BlindIndex: &testBlindIndex,
			Data:       testData,
		}.Build(),
	}.Build()

	_, err := handler.UpdateSecret(middleware.WithUserID(context.Background(), testUserID), req)
	assert.NoError(t, err)
	mockService.AssertExpectations(t)
}

func TestSecretHandler_UpdateSecret_NotFound(t *testing.T) {
	mockService := mocks.NewSecretService(t)
	mockService.On("Update", mock.Anything, testUserID, testBlindIndex, testData).
		Return(domain.ErrSecretNotFound)

	handler := NewSecretHandler(mockService)

	req := pb.UpdateSecretRequest_builder{
		Item: pb.SecretItem_builder{
			BlindIndex: &testBlindIndex,
			Data:       testData,
		}.Build(),
	}.Build()

	_, err := handler.UpdateSecret(middleware.WithUserID(context.Background(), testUserID), req)
	assert.Equal(t, codes.NotFound, status.Code(err))
	mockService.AssertExpectations(t)
}

func TestSecretHandler_UpdateSecret_InvalidArgument(t *testing.T) {
	mockService := mocks.NewSecretService(t)
	mockService.On("Update", mock.Anything, testUserID, testBlindIndex, emptyData).
		Return(domain.ErrInvalidArgument)

	handler := NewSecretHandler(mockService)

	req := pb.UpdateSecretRequest_builder{
		Item: pb.SecretItem_builder{
			BlindIndex: &testBlindIndex,
			Data:       emptyData,
		}.Build(),
	}.Build()

	_, err := handler.UpdateSecret(middleware.WithUserID(context.Background(), testUserID), req)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	mockService.AssertExpectations(t)
}

func TestSecretHandler_UpdateSecret_Unauthenticated(t *testing.T) {
	mockService := mocks.NewSecretService(t)
	handler := NewSecretHandler(mockService)

	req := pb.UpdateSecretRequest_builder{
		Item: pb.SecretItem_builder{
			BlindIndex: &testBlindIndex,
			Data:       testData,
		}.Build(),
	}.Build()

	_, err := handler.UpdateSecret(context.Background(), req)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	mockService.AssertExpectations(t)
}

func TestSecretHandler_GetSecret_Success(t *testing.T) {
	mockService := mocks.NewSecretService(t)
	mockService.On("GetByBlindIndex", mock.Anything, testUserID, testBlindIndex).
		Return(testSecret, nil)

	handler := NewSecretHandler(mockService)

	req := pb.GetSecretRequest_builder{
		BlindIndex: &testBlindIndex,
	}.Build()

	resp, err := handler.GetSecret(middleware.WithUserID(context.Background(), testUserID), req)
	assert.NoError(t, err)
	assert.Equal(t, testSecret.Data, resp.GetData())
	mockService.AssertExpectations(t)
}

func TestSecretHandler_GetSecret_NotFound(t *testing.T) {
	mockService := mocks.NewSecretService(t)
	mockService.On("GetByBlindIndex", mock.Anything, testUserID, testBlindIndex).
		Return(nil, domain.ErrSecretNotFound)

	handler := NewSecretHandler(mockService)

	req := pb.GetSecretRequest_builder{
		BlindIndex: &testBlindIndex,
	}.Build()

	_, err := handler.GetSecret(middleware.WithUserID(context.Background(), testUserID), req)
	assert.Equal(t, codes.NotFound, status.Code(err))
	mockService.AssertExpectations(t)
}

func TestSecretHandler_GetSecret_Unauthenticated(t *testing.T) {
	mockService := mocks.NewSecretService(t)
	handler := NewSecretHandler(mockService)

	req := pb.GetSecretRequest_builder{
		BlindIndex: &testBlindIndex,
	}.Build()

	_, err := handler.GetSecret(context.Background(), req)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	mockService.AssertExpectations(t)
}

func TestSecretHandler_DeleteSecret_Success(t *testing.T) {
	mockService := mocks.NewSecretService(t)
	mockService.On("Delete", mock.Anything, testUserID, testBlindIndex).
		Return(nil)

	handler := NewSecretHandler(mockService)

	req := pb.DeleteSecretRequest_builder{
		BlindIndex: &testBlindIndex,
	}.Build()

	_, err := handler.DeleteSecret(middleware.WithUserID(context.Background(), testUserID), req)
	assert.NoError(t, err)
	mockService.AssertExpectations(t)
}

func TestSecretHandler_DeleteSecret_NotFound(t *testing.T) {
	mockService := mocks.NewSecretService(t)
	mockService.On("Delete", mock.Anything, testUserID, testBlindIndex).
		Return(domain.ErrSecretNotFound)

	handler := NewSecretHandler(mockService)

	req := pb.DeleteSecretRequest_builder{
		BlindIndex: &testBlindIndex,
	}.Build()

	_, err := handler.DeleteSecret(middleware.WithUserID(context.Background(), testUserID), req)
	assert.Equal(t, codes.NotFound, status.Code(err))
	mockService.AssertExpectations(t)
}

func TestSecretHandler_DeleteSecret_Unauthenticated(t *testing.T) {
	mockService := mocks.NewSecretService(t)
	handler := NewSecretHandler(mockService)

	req := pb.DeleteSecretRequest_builder{
		BlindIndex: &testBlindIndex,
	}.Build()

	_, err := handler.DeleteSecret(context.Background(), req)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	mockService.AssertExpectations(t)
}

func TestSecretHandler_ListSecrets_Success(t *testing.T) {
	mockService := mocks.NewSecretService(t)
	mockService.On("ListByUserID", mock.Anything, testUserID).
		Return([]*domain.Secret{testSecret}, nil)

	handler := NewSecretHandler(mockService)

	req := pb.ListSecretsRequest_builder{}.Build()

	resp, err := handler.ListSecrets(middleware.WithUserID(context.Background(), testUserID), req)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resp.GetItems()))
	mockService.AssertExpectations(t)
}

func TestSecretHandler_ListSecrets_Unauthenticated(t *testing.T) {
	mockService := mocks.NewSecretService(t)
	handler := NewSecretHandler(mockService)

	req := pb.ListSecretsRequest_builder{}.Build()

	_, err := handler.ListSecrets(context.Background(), req)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	mockService.AssertExpectations(t)
}
