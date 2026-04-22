package service_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/F3dosik/GophKeeper/internal/client/mocks"
	"github.com/F3dosik/GophKeeper/internal/client/service"
	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/F3dosik/GophKeeper/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var testMasterKey = make([]byte, 32)

func TestSecretsService_CreateSecret(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSecrets := mocks.NewSecretsClient(t)
		mockSecrets.On("CreateSecret", mock.Anything, mock.AnythingOfType("string"),
			mock.AnythingOfType("[]uint8")).Return(nil)

		svc, err := service.NewSecretsService(mockSecrets, testMasterKey)
		require.NoError(t, err)

		payload := &domain.SecretPayload{
			Name: "github",
			Type: domain.SecretTypeCredentials,
			Data: json.RawMessage(`{"login":"user","password":"pass"}`),
		}
		err = svc.CreateSecret(context.Background(), payload)

		require.NoError(t, err)
		mockSecrets.AssertExpectations(t)
	})

	t.Run("already exists", func(t *testing.T) {
		mockSecrets := mocks.NewSecretsClient(t)
		mockSecrets.On("CreateSecret", mock.Anything, mock.Anything, mock.Anything).
			Return(domain.ErrAlreadyExists)

		svc, err := service.NewSecretsService(mockSecrets, testMasterKey)
		require.NoError(t, err)

		err = svc.CreateSecret(context.Background(), &domain.SecretPayload{
			Name: "github",
			Type: domain.SecretTypeCredentials,
		})

		assert.ErrorIs(t, err, domain.ErrAlreadyExists)
	})
}

func TestSecretsService_GetSecret(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, err := service.NewSecretsService(nil, testMasterKey)
		require.NoError(t, err)

		// шифруем тестовый payload
		payload := &domain.SecretPayload{
			Name:     "github",
			Type:     domain.SecretTypeCredentials,
			Data:     json.RawMessage(`{"login":"user","password":"pass"}`),
			Metadata: "personal",
		}

		mockSecrets := mocks.NewSecretsClient(t)
		// получаем зашифрованные данные через сам сервис
		// используем реальное шифрование

		now := time.Now()
		mockSecrets.On("GetSecret", mock.Anything, mock.AnythingOfType("string")).
			Return(&domain.Secret{
				Data:      encryptPayloadForTest(t, testMasterKey, payload),
				CreatedAt: now,
				UpdatedAt: now,
			}, nil)

		svc2, err := service.NewSecretsService(mockSecrets, testMasterKey)
		require.NoError(t, err)

		info, err := svc2.GetSecret(context.Background(), "github", domain.SecretTypeCredentials)
		require.NoError(t, err)
		assert.Equal(t, "github", info.Name)
		assert.Equal(t, domain.SecretTypeCredentials, info.Type)
		assert.Equal(t, now.Unix(), info.CreatedAt.Unix())
		_ = svc
	})

	t.Run("not found", func(t *testing.T) {
		mockSecrets := mocks.NewSecretsClient(t)
		mockSecrets.On("GetSecret", mock.Anything, mock.Anything).
			Return(nil, domain.ErrNotFound)

		svc, err := service.NewSecretsService(mockSecrets, testMasterKey)
		require.NoError(t, err)

		_, err = svc.GetSecret(context.Background(), "github", domain.SecretTypeCredentials)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestSecretsService_DeleteSecret(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSecrets := mocks.NewSecretsClient(t)
		mockSecrets.On("DeleteSecret", mock.Anything, mock.AnythingOfType("string")).
			Return(nil)

		svc, err := service.NewSecretsService(mockSecrets, testMasterKey)
		require.NoError(t, err)

		err = svc.DeleteSecret(context.Background(), "github", domain.SecretTypeCredentials)
		require.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		mockSecrets := mocks.NewSecretsClient(t)
		mockSecrets.On("DeleteSecret", mock.Anything, mock.Anything).
			Return(domain.ErrNotFound)

		svc, err := service.NewSecretsService(mockSecrets, testMasterKey)
		require.NoError(t, err)

		err = svc.DeleteSecret(context.Background(), "github", domain.SecretTypeCredentials)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestSecretsService_ListSecrets(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		payload1 := &domain.SecretPayload{
			Name: "github",
			Type: domain.SecretTypeCredentials,
			Data: json.RawMessage(`{"login":"user","password":"pass"}`),
		}
		payload2 := &domain.SecretPayload{
			Name: "note",
			Type: domain.SecretTypeText,
			Data: json.RawMessage(`{"text":"some text"}`),
		}

		now := time.Now()
		encrypted1 := encryptPayloadForTest(t, testMasterKey, payload1)
		encrypted2 := encryptPayloadForTest(t, testMasterKey, payload2)

		mockSecrets := mocks.NewSecretsClient(t)
		mockSecrets.On("ListSecrets", mock.Anything).Return([]*domain.Secret{
			{Data: encrypted1, CreatedAt: now, UpdatedAt: now},
			{Data: encrypted2, CreatedAt: now, UpdatedAt: now},
		}, nil)

		svc, err := service.NewSecretsService(mockSecrets, testMasterKey)
		require.NoError(t, err)

		infos, err := svc.ListSecrets(context.Background())
		require.NoError(t, err)
		require.Len(t, infos, 2)
		assert.Equal(t, "github", infos[0].Name)
		assert.Equal(t, domain.SecretTypeCredentials, infos[0].Type)
		assert.Equal(t, "note", infos[1].Name)
		assert.Equal(t, domain.SecretTypeText, infos[1].Type)
		assert.Equal(t, now.Unix(), infos[0].CreatedAt.Unix())
	})

	t.Run("empty list", func(t *testing.T) {
		mockSecrets := mocks.NewSecretsClient(t)
		mockSecrets.On("ListSecrets", mock.Anything).
			Return([]*domain.Secret{}, nil)

		svc, err := service.NewSecretsService(mockSecrets, testMasterKey)
		require.NoError(t, err)

		infos, err := svc.ListSecrets(context.Background())
		require.NoError(t, err)
		assert.Empty(t, infos)
	})

	t.Run("client error", func(t *testing.T) {
		mockSecrets := mocks.NewSecretsClient(t)
		mockSecrets.On("ListSecrets", mock.Anything).
			Return(nil, domain.ErrInvalidCredentials)

		svc, err := service.NewSecretsService(mockSecrets, testMasterKey)
		require.NoError(t, err)

		_, err = svc.ListSecrets(context.Background())
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("decrypt error", func(t *testing.T) {
		mockSecrets := mocks.NewSecretsClient(t)
		mockSecrets.On("ListSecrets", mock.Anything).Return([]*domain.Secret{
			{Data: []byte("invalid ciphertext"), CreatedAt: time.Now()},
		}, nil)

		svc, err := service.NewSecretsService(mockSecrets, testMasterKey)
		require.NoError(t, err)

		_, err = svc.ListSecrets(context.Background())
		assert.Error(t, err)
	})
}

func TestSecretsService_UpdateSecret(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSecrets := mocks.NewSecretsClient(t)
		mockSecrets.On("UpdateSecret", mock.Anything, mock.AnythingOfType("string"),
			mock.AnythingOfType("[]uint8")).Return(nil)

		svc, err := service.NewSecretsService(mockSecrets, testMasterKey)
		require.NoError(t, err)

		payload := &domain.SecretPayload{
			Name: "github",
			Type: domain.SecretTypeCredentials,
			Data: json.RawMessage(`{"login":"user","password":"newpass"}`),
		}
		err = svc.UpdateSecret(context.Background(), payload)

		require.NoError(t, err)
		mockSecrets.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockSecrets := mocks.NewSecretsClient(t)
		mockSecrets.On("UpdateSecret", mock.Anything, mock.Anything, mock.Anything).
			Return(domain.ErrNotFound)

		svc, err := service.NewSecretsService(mockSecrets, testMasterKey)
		require.NoError(t, err)

		err = svc.UpdateSecret(context.Background(), &domain.SecretPayload{
			Name: "github",
			Type: domain.SecretTypeCredentials,
		})
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("same blind index for same name and type", func(t *testing.T) {
		var capturedIndex1, capturedIndex2 string

		mockSecrets := mocks.NewSecretsClient(t)
		mockSecrets.On("UpdateSecret", mock.Anything, mock.MatchedBy(func(idx string) bool {
			capturedIndex1 = idx
			return true
		}), mock.Anything).Return(nil).Once()
		mockSecrets.On("UpdateSecret", mock.Anything, mock.MatchedBy(func(idx string) bool {
			capturedIndex2 = idx
			return true
		}), mock.Anything).Return(nil).Once()

		svc, err := service.NewSecretsService(mockSecrets, testMasterKey)
		require.NoError(t, err)

		payload := &domain.SecretPayload{
			Name: "github",
			Type: domain.SecretTypeCredentials,
			Data: json.RawMessage(`{"login":"user","password":"pass1"}`),
		}
		err = svc.UpdateSecret(context.Background(), payload)
		require.NoError(t, err)

		payload.Data = json.RawMessage(`{"login":"user","password":"pass2"}`)
		err = svc.UpdateSecret(context.Background(), payload)
		require.NoError(t, err)

		assert.Equal(t, capturedIndex1, capturedIndex2)
	})
}

// encryptPayloadForTest шифрует payload для использования в тестах.
func encryptPayloadForTest(t *testing.T, masterKey []byte, payload *domain.SecretPayload) []byte {
	t.Helper()

	encKey, err := crypto.HKDF(masterKey, crypto.InfoEncryption)
	require.NoError(t, err)

	cipher, err := crypto.NewAESCipher(encKey)
	require.NoError(t, err)

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	ciphertext, err := cipher.Encrypt(data)
	require.NoError(t, err)

	return ciphertext
}
