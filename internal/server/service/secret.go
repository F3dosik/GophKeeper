package service

import (
	"context"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/google/uuid"
)

type SecretService interface {
	// Create регистрирует новый секрет.
	// Возвращает ErrSecretAlreadyExists если секрет с указанным blindIndex уже существует.
	// Возвращает ErrInvalidArgument если blind index или data пустые.
	Create(ctx context.Context, userID uuid.UUID, blindIndex string, data []byte) error

	// Update изменяет существующую приватную информацию.
	// Возвращает ErrSecretNotFound если секрет с указанным blindIndex не существует.
	// Возвращает codes.InvalidArgument если blind index или data пустые.
	Update(ctx context.Context, userID uuid.UUID, blindIndex string, data []byte) error

	// GetByBlindIndex получает секрет по userID и blindIndex.
	// Возвращает ErrSecretNotFound если секрет с указанным blindIndex не существует.
	GetByBlindIndex(ctx context.Context, userID uuid.UUID, blindIndex string) (*domain.Secret, error)

	// ListByUserID возвращает список всех секретов для указанного пользователя.
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Secret, error)

	// Delete удаляет секрет по userID и blindIndex.
	// Возвращает ErrSecretNotFound если секрет с указанным blindIndex не существует.
	Delete(ctx context.Context, userID uuid.UUID, blindIndex string) error
}

// secretService реализует SecretService.
type secretService struct {
	repo domain.SecretRepository
}

// NewSecretService создаёт новый экземпляр secretService.
func NewSecretService(repo domain.SecretRepository) SecretService {
	return &secretService{repo: repo}
}

// Create регистрирует новый секрет.
func (s *secretService) Create(ctx context.Context, userID uuid.UUID, blindIndex string, data []byte) error {
	if blindIndex == "" || len(data) == 0 {
		return domain.ErrInvalidArgument
	}
	return s.repo.Create(ctx, &domain.Secret{
		UserID:     userID,
		BlindIndex: blindIndex,
		Data:       data,
	})
}

// Update изменяет существующую приватную информацию.
func (s *secretService) Update(ctx context.Context, userID uuid.UUID, blindIndex string, data []byte) error {
	if blindIndex == "" || len(data) == 0 {
		return domain.ErrInvalidArgument
	}
	return s.repo.Update(ctx, &domain.Secret{
		UserID:     userID,
		BlindIndex: blindIndex,
		Data:       data,
	})
}

// GetByBlindIndex получает секрет по userID и blindIndex.
func (s *secretService) GetByBlindIndex(ctx context.Context, userID uuid.UUID, blindIndex string) (*domain.Secret, error) {
	if blindIndex == "" {
		return nil, domain.ErrInvalidArgument
	}
	return s.repo.GetByBlindIndex(ctx, userID, blindIndex)
}

// ListByUserID возвращает список всех секретов для указанного пользователя.
func (s *secretService) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Secret, error) {
	return s.repo.ListByUserID(ctx, userID)
}

// Delete удаляет секрет по userID и blindIndex.
func (s *secretService) Delete(ctx context.Context, userID uuid.UUID, blindIndex string) error {
	if blindIndex == "" {
		return domain.ErrInvalidArgument
	}
	return s.repo.Delete(ctx, userID, blindIndex)
}
