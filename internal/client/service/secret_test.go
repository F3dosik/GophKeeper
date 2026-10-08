package service_test

import (
	"bytes"
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
			{BlindIndex: blindIndexForTest(t, payload1), Data: encrypted1, CreatedAt: now, UpdatedAt: now},
			{BlindIndex: blindIndexForTest(t, payload2), Data: encrypted2, CreatedAt: now, UpdatedAt: now},
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

// blindIndexForTest вычисляет blind index payload так же, как сервис.
func blindIndexForTest(t *testing.T, payload *domain.SecretPayload) string {
	t.Helper()

	hmacKey, err := crypto.HKDF(testMasterKey, crypto.InfoBlindIndex)
	require.NoError(t, err)
	return crypto.BlindIndex(payload.Name, payload.Type, hmacKey)
}

// Сервер может вернуть под одним blind index шифротекст другого секрета того же
// пользователя: AES-GCM его расшифрует, поэтому клиент обязан сверить имя и тип.
func TestSecretsService_DetectsSwappedSecret(t *testing.T) {
	bank := &domain.SecretPayload{
		Name: "bank", Type: domain.SecretTypeCredentials,
		Data: json.RawMessage(`{"login":"real","password":"real"}`),
	}
	decoy := &domain.SecretPayload{
		Name: "decoy", Type: domain.SecretTypeCredentials,
		Data: json.RawMessage(`{"login":"attacker","password":"phish"}`),
	}
	decoyData := encryptPayloadForTest(t, testMasterKey, decoy)

	t.Run("get", func(t *testing.T) {
		mockSecrets := mocks.NewSecretsClient(t)
		mockSecrets.On("GetSecret", mock.Anything, blindIndexForTest(t, bank)).
			Return(&domain.Secret{Data: decoyData}, nil)

		svc, err := service.NewSecretsService(mockSecrets, testMasterKey)
		require.NoError(t, err)

		_, err = svc.GetSecret(context.Background(), "bank", domain.SecretTypeCredentials)
		assert.ErrorIs(t, err, service.ErrIntegrity)
	})

	t.Run("list", func(t *testing.T) {
		mockSecrets := mocks.NewSecretsClient(t)
		mockSecrets.On("ListSecrets", mock.Anything).Return([]*domain.Secret{
			{BlindIndex: blindIndexForTest(t, bank), Data: decoyData},
		}, nil)

		svc, err := service.NewSecretsService(mockSecrets, testMasterKey)
		require.NoError(t, err)

		_, err = svc.ListSecrets(context.Background())
		assert.ErrorIs(t, err, service.ErrIntegrity)
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

func TestSecretsService_Reencrypt(t *testing.T) {
	payload := &domain.SecretPayload{
		Name: "bank", Type: domain.SecretTypeCredentials,
		Data: json.RawMessage(`{"login":"u","password":"p"}`),
	}
	updatedAt := time.Date(2026, 1, 2, 3, 4, 5, 6000, time.UTC)

	mockSecrets := mocks.NewSecretsClient(t)
	mockSecrets.On("ListSecrets", mock.Anything).Return([]*domain.Secret{{
		BlindIndex: blindIndexForTest(t, payload),
		Data:       encryptPayloadForTest(t, testMasterKey, payload),
		UpdatedAt:  updatedAt,
	}}, nil)

	svc, err := service.NewSecretsService(mockSecrets, testMasterKey)
	require.NoError(t, err)

	newMasterKey := bytes.Repeat([]byte{9}, 32)
	items, err := svc.Reencrypt(context.Background(), newMasterKey)
	require.NoError(t, err)
	require.Len(t, items, 1)

	item := items[0]
	assert.Equal(t, blindIndexForTest(t, payload), item.OldBlindIndex)
	assert.Equal(t, updatedAt, item.ExpectedUpdatedAt)

	// Новые blind index и данные соответствуют новому ключу: их принимает сервис нового ключа.
	newHMAC, err := crypto.HKDF(newMasterKey, crypto.InfoBlindIndex)
	require.NoError(t, err)
	assert.Equal(t, crypto.BlindIndex("bank", domain.SecretTypeCredentials, newHMAC), item.NewBlindIndex)

	newClient := mocks.NewSecretsClient(t)
	newClient.On("GetSecret", mock.Anything, item.NewBlindIndex).Return(&domain.Secret{Data: item.Data}, nil)
	newSvc, err := service.NewSecretsService(newClient, newMasterKey)
	require.NoError(t, err)
	got, err := newSvc.GetSecret(context.Background(), "bank", domain.SecretTypeCredentials)
	require.NoError(t, err)
	assert.JSONEq(t, string(payload.Data), string(got.Data))
}
