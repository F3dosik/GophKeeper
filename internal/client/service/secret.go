package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/F3dosik/GophKeeper/internal/client/grpcclient"
	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/F3dosik/GophKeeper/pkg/crypto"
)

// SecretsService определяет интерфейс для работы с секретами пользователя.
type SecretsService interface {
	// ListSecrets возвращает список всех секретов пользователя.
	// Расшифровывает данные каждого секрета перед возвратом.
	ListSecrets(ctx context.Context) ([]*domain.SecretInfo, error)

	// CreateSecret создаёт новый секрет на сервере.
	// Шифрует payload и вычисляет blind index перед отправкой.
	CreateSecret(ctx context.Context, payload *domain.SecretPayload) error

	// UpdateSecret обновляет существующий секрет на сервере.
	// Шифрует payload и вычисляет blind index перед отправкой.
	UpdateSecret(ctx context.Context, payload *domain.SecretPayload) error

	// GetSecret возвращает секрет по имени и типу.
	// Расшифровывает данные перед возвратом.
	GetSecret(ctx context.Context, name string, secretType domain.SecretType) (*domain.SecretInfo, error)

	// DeleteSecret удаляет секрет по имени и типу.
	DeleteSecret(ctx context.Context, name string, secretType domain.SecretType) error
}

// secretsService реализует SecretsService.
type secretsService struct {
	// client используется для взаимодействия с gRPC сервером секретов.
	client grpcclient.SecretsClient
	// hmacKey используется для вычисления blind index через HMAC-SHA256.
	hmacKey []byte
	// cipher используется для шифрования и расшифровки данных секретов.
	cipher crypto.Cipher
}

// NewSecretsService создаёт новый secretsService.
// Деривирует ключ шифрования и ключ HMAC из masterKey через HKDF.
func NewSecretsService(client grpcclient.SecretsClient, masterKey []byte) (SecretsService, error) {
	hmacKey, err := crypto.HKDF(masterKey, crypto.InfoBlindIndex)
	if err != nil {
		return nil, fmt.Errorf("new secrets service: %w", err)
	}

	encKey, err := crypto.HKDF(masterKey, crypto.InfoEncryption)
	if err != nil {
		return nil, fmt.Errorf("new secrets service: %w", err)
	}
	cipher, err := crypto.NewAESCipher(encKey)
	if err != nil {
		return nil, fmt.Errorf("newSecretsService: %w", err)
	}
	return &secretsService{client: client, hmacKey: hmacKey, cipher: cipher}, nil
}

// encryptPayload сериализует payload в JSON и шифрует его через AES-256-GCM.
func (s *secretsService) encryptPayload(payload *domain.SecretPayload) ([]byte, error) {
	plaintext, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode payload: %w", err)
	}
	ciphertext, err := s.cipher.Encrypt(plaintext)
	if err != nil {
		return nil, fmt.Errorf("encrypt payload: %w", err)
	}
	return ciphertext, nil
}

// decryptPayload расшифровывает данные и десериализует их в SecretPayload.
func (s *secretsService) decryptPayload(data []byte) (*domain.SecretPayload, error) {
	plaintext, err := s.cipher.Decrypt(data)
	if err != nil {
		return nil, fmt.Errorf("decrypt payload: %w", err)
	}

	var payload domain.SecretPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	return &payload, nil
}

// ListSecrets возвращает список всех секретов пользователя.
// Расшифровывает данные каждого секрета и заполняет timestamps с сервера.
func (s *secretsService) ListSecrets(ctx context.Context) ([]*domain.SecretInfo, error) {
	secrets, err := s.client.ListSecrets(ctx)
	if err != nil {
		return nil, fmt.Errorf("secretsService.ListSecrets: %w", err)
	}
	secretInformations := make([]*domain.SecretInfo, 0, len(secrets))
	for _, secret := range secrets {
		payload, err := s.decryptPayload(secret.Data)
		if err != nil {
			return nil, fmt.Errorf("secretService.ListSecrets: %w", err)
		}
		secretInformations = append(secretInformations, &domain.SecretInfo{
			SecretPayload: *payload,
			CreatedAt:     secret.CreatedAt,
			UpdatedAt:     secret.UpdatedAt,
		})
	}
	return secretInformations, nil
}

// CreateSecret создаёт новый секрет на сервере.
// Вычисляет blind index, шифрует payload и отправляет на сервер.
func (s *secretsService) CreateSecret(
	ctx context.Context, payload *domain.SecretPayload,
) error {
	blindIndex := crypto.BlindIndex(payload.Name, payload.Type, s.hmacKey)

	ciphertext, err := s.encryptPayload(payload)
	if err != nil {
		return fmt.Errorf("secretService.CreateSecret: %w", err)
	}

	if err := s.client.CreateSecret(ctx, blindIndex, ciphertext); err != nil {
		return fmt.Errorf("secretService.CreateSecret: %w", err)
	}
	return nil
}

// UpdateSecret обновляет существующий секрет на сервере.
// Вычисляет blind index, шифрует payload и отправляет на сервер.
func (s *secretsService) UpdateSecret(ctx context.Context, payload *domain.SecretPayload) error {
	blindIndex := crypto.BlindIndex(payload.Name, payload.Type, s.hmacKey)

	ciphertext, err := s.encryptPayload(payload)
	if err != nil {
		return fmt.Errorf("secretService.UpdateSecret: %w", err)
	}

	if err := s.client.UpdateSecret(ctx, blindIndex, ciphertext); err != nil {
		return fmt.Errorf("secretService.UpdateSecret: %w", err)
	}
	return nil
}

// GetSecret возвращает секрет по имени и типу.
// Вычисляет blind index, получает зашифрованные данные и расшифровывает их.
func (s *secretsService) GetSecret(ctx context.Context, name string, secretType domain.SecretType) (*domain.SecretInfo, error) {
	blindIndex := crypto.BlindIndex(name, secretType, s.hmacKey)
	secret, err := s.client.GetSecret(ctx, blindIndex)
	if err != nil {
		return nil, fmt.Errorf("secretService.GetSecret: %w", err)
	}

	payload, err := s.decryptPayload(secret.Data)
	if err != nil {
		return nil, fmt.Errorf("secretService.GetSecret: %w", err)
	}

	return &domain.SecretInfo{
		SecretPayload: *payload,
		CreatedAt:     secret.CreatedAt,
		UpdatedAt:     secret.UpdatedAt,
	}, nil
}

// DeleteSecret удаляет секрет по имени и типу.
// Вычисляет blind index и отправляет запрос на удаление.
func (s *secretsService) DeleteSecret(ctx context.Context, name string, secretType domain.SecretType) error {
	blindIndex := crypto.BlindIndex(name, secretType, s.hmacKey)
	if err := s.client.DeleteSecret(ctx, blindIndex); err != nil {
		return fmt.Errorf("secretService.DeleteSecret: %w", err)
	}
	return nil
}
