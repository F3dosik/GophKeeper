package service

import (
	"context"
	"strings"
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

// testTokenVersion — версия токена запроса, которая должна дойти до репозитория.
const testTokenVersion = 7

// testBlindIndex — корректный blind index (64 hex-символа).
var testBlindIndex = strings.Repeat("ab", 32)

func TestSecretService_Create_EmptyBlindIndex(t *testing.T) {
	svc := NewSecretService(mocks.NewSecretRepository(t), testLimits)
	err := svc.Create(context.Background(), testID, testTokenVersion, "", testData)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}

func TestSecretService_Create_EmptyData(t *testing.T) {
	svc := NewSecretService(mocks.NewSecretRepository(t), testLimits)
	err := svc.Create(context.Background(), testID, testTokenVersion, testBlindIndex, []byte{})
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}

func TestSecretService_Create_AlreadyExists(t *testing.T) {
	mockRepo := mocks.NewSecretRepository(t)
	mockRepo.On("CountByUserID", mock.Anything, testID).Return(0, nil)
	mockRepo.On("Create", mock.Anything, mock.Anything, testTokenVersion).
		Return(domain.ErrSecretAlreadyExists)

	svc := NewSecretService(mockRepo, testLimits)
	err := svc.Create(context.Background(), testID, testTokenVersion, testBlindIndex, testData)
	assert.ErrorIs(t, err, domain.ErrSecretAlreadyExists)
}

func TestSecretService_Create_Success(t *testing.T) {
	mockRepo := mocks.NewSecretRepository(t)
	mockRepo.On("CountByUserID", mock.Anything, testID).Return(testLimits.MaxCount-1, nil)
	mockRepo.On("Create", mock.Anything, &domain.Secret{
		UserID:     testID,
		BlindIndex: testBlindIndex,
		Data:       testData,
	}, testTokenVersion).Return(nil)

	svc := NewSecretService(mockRepo, testLimits)
	err := svc.Create(context.Background(), testID, testTokenVersion, testBlindIndex, testData)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSecretService_Create_TooLarge(t *testing.T) {
	svc := NewSecretService(mocks.NewSecretRepository(t), testLimits)
	err := svc.Create(context.Background(), testID, testTokenVersion, testBlindIndex, make([]byte, testLimits.MaxSize+1))
	assert.ErrorIs(t, err, domain.ErrSecretTooLarge)
}

func TestSecretService_Create_QuotaExceeded(t *testing.T) {
	mockRepo := mocks.NewSecretRepository(t)
	mockRepo.On("CountByUserID", mock.Anything, testID).Return(testLimits.MaxCount, nil)

	svc := NewSecretService(mockRepo, testLimits)
	err := svc.Create(context.Background(), testID, testTokenVersion, testBlindIndex, testData)
	assert.ErrorIs(t, err, domain.ErrSecretQuotaExceeded)
}

func TestSecretService_Update_TooLarge(t *testing.T) {
	svc := NewSecretService(mocks.NewSecretRepository(t), testLimits)
	err := svc.Update(context.Background(), testID, testTokenVersion, testBlindIndex, make([]byte, testLimits.MaxSize+1))
	assert.ErrorIs(t, err, domain.ErrSecretTooLarge)
}

func TestSecretService_Update_EmptyBlindIndex(t *testing.T) {
	svc := NewSecretService(mocks.NewSecretRepository(t), testLimits)
	err := svc.Update(context.Background(), testID, testTokenVersion, "", testData)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}

func TestSecretService_Update_EmptyData(t *testing.T) {
	svc := NewSecretService(mocks.NewSecretRepository(t), testLimits)
	err := svc.Update(context.Background(), testID, testTokenVersion, testBlindIndex, []byte{})
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}

func TestSecretService_Update_NotFound(t *testing.T) {
	mockRepo := mocks.NewSecretRepository(t)
	mockRepo.On("Update", mock.Anything, mock.Anything, testTokenVersion).
		Return(domain.ErrSecretNotFound)

	svc := NewSecretService(mockRepo, testLimits)
	err := svc.Update(context.Background(), testID, testTokenVersion, testBlindIndex, testData)
	assert.ErrorIs(t, err, domain.ErrSecretNotFound)
}

func TestSecretService_Update_Success(t *testing.T) {
	mockRepo := mocks.NewSecretRepository(t)
	mockRepo.On("Update", mock.Anything, &domain.Secret{
		UserID:     testID,
		BlindIndex: testBlindIndex,
		Data:       testData,
	}, testTokenVersion).Return(nil)

	svc := NewSecretService(mockRepo, testLimits)
	err := svc.Update(context.Background(), testID, testTokenVersion, testBlindIndex, testData)
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
	mockRepo.On("GetByBlindIndex", mock.Anything, testID, testBlindIndex).
		Return(nil, domain.ErrSecretNotFound)

	svc := NewSecretService(mockRepo, testLimits)
	_, err := svc.GetByBlindIndex(context.Background(), testID, testBlindIndex)
	assert.ErrorIs(t, err, domain.ErrSecretNotFound)
}

func TestSecretService_GetByBlindIndex_Success(t *testing.T) {
	want := &domain.Secret{UserID: testID, BlindIndex: testBlindIndex, Data: testData}
	mockRepo := mocks.NewSecretRepository(t)
	mockRepo.On("GetByBlindIndex", mock.Anything, testID, testBlindIndex).
		Return(want, nil)

	svc := NewSecretService(mockRepo, testLimits)
	got, err := svc.GetByBlindIndex(context.Background(), testID, testBlindIndex)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

// secretsWithData возвращает n секретов с упорядоченными ID и данными размера size.
func secretsWithData(n, size int) []*domain.Secret {
	secrets := make([]*domain.Secret, n)
	for i := range secrets {
		id := uuid.UUID{}
		id[15] = byte(i + 1)
		secrets[i] = &domain.Secret{ID: id, UserID: testID, Data: make([]byte, size)}
	}
	return secrets
}

func TestSecretService_ListPage(t *testing.T) {
	ctx := context.Background()

	t.Run("last page has no token", func(t *testing.T) {
		rows := secretsWithData(2, 10)
		mockRepo := mocks.NewSecretRepository(t)
		mockRepo.On("ListByUserID", mock.Anything, testID, uuid.Nil, MaxPageSize+1).Return(rows, nil)

		page, err := NewSecretService(mockRepo, testLimits).ListPage(ctx, testID, "", 0)
		assert.NoError(t, err)
		assert.Equal(t, rows, page.Secrets)
		assert.Empty(t, page.NextPageToken)
	})

	t.Run("page size limit sets token to last returned id", func(t *testing.T) {
		rows := secretsWithData(3, 10)
		mockRepo := mocks.NewSecretRepository(t)
		mockRepo.On("ListByUserID", mock.Anything, testID, uuid.Nil, 3).Return(rows, nil)

		page, err := NewSecretService(mockRepo, testLimits).ListPage(ctx, testID, "", 2)
		assert.NoError(t, err)
		assert.Len(t, page.Secrets, 2)
		assert.Equal(t, rows[1].ID.String(), page.NextPageToken)
	})

	t.Run("token continues after cursor", func(t *testing.T) {
		after := secretsWithData(1, 0)[0].ID
		mockRepo := mocks.NewSecretRepository(t)
		mockRepo.On("ListByUserID", mock.Anything, testID, after, 3).Return([]*domain.Secret{}, nil)

		page, err := NewSecretService(mockRepo, testLimits).ListPage(ctx, testID, after.String(), 2)
		assert.NoError(t, err)
		assert.Empty(t, page.Secrets)
	})

	t.Run("byte budget splits pages but never returns an empty page", func(t *testing.T) {
		rows := secretsWithData(3, MaxPageBytes) // каждый секрет сам по себе на весь бюджет
		mockRepo := mocks.NewSecretRepository(t)
		mockRepo.On("ListByUserID", mock.Anything, testID, uuid.Nil, MaxPageSize+1).Return(rows, nil)

		page, err := NewSecretService(mockRepo, testLimits).ListPage(ctx, testID, "", 0)
		assert.NoError(t, err)
		assert.Len(t, page.Secrets, 1)
		assert.Equal(t, rows[0].ID.String(), page.NextPageToken)
	})

	t.Run("invalid arguments", func(t *testing.T) {
		svc := NewSecretService(mocks.NewSecretRepository(t), testLimits)
		_, err := svc.ListPage(ctx, testID, "", MaxPageSize+1)
		assert.ErrorIs(t, err, domain.ErrInvalidArgument)
		_, err = svc.ListPage(ctx, testID, "", -1)
		assert.ErrorIs(t, err, domain.ErrInvalidArgument)
		_, err = svc.ListPage(ctx, testID, "not-a-uuid", 0)
		assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	})
}

func TestSecretService_Delete_EmptyBlindIndex(t *testing.T) {
	svc := NewSecretService(mocks.NewSecretRepository(t), testLimits)
	err := svc.Delete(context.Background(), testID, "")
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}

func TestSecretService_Delete_NotFound(t *testing.T) {
	mockRepo := mocks.NewSecretRepository(t)
	mockRepo.On("Delete", mock.Anything, testID, testBlindIndex).
		Return(domain.ErrSecretNotFound)

	svc := NewSecretService(mockRepo, testLimits)
	err := svc.Delete(context.Background(), testID, testBlindIndex)
	assert.ErrorIs(t, err, domain.ErrSecretNotFound)
}

func TestSecretService_Delete_Success(t *testing.T) {
	mockRepo := mocks.NewSecretRepository(t)
	mockRepo.On("Delete", mock.Anything, testID, testBlindIndex).Return(nil)

	svc := NewSecretService(mockRepo, testLimits)
	err := svc.Delete(context.Background(), testID, testBlindIndex)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
