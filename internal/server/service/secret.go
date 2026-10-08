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
	// Возвращает ErrSecretTooLarge или ErrSecretQuotaExceeded при превышении квот.
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

// SecretLimits задаёт квоты на хранение секретов одного пользователя.
// Без них любой зарегистрировавшийся может заполнить диск сервера.
type SecretLimits struct {
	// MaxSize — максимальный размер зашифрованных данных одного секрета в байтах.
	MaxSize int
	// MaxCount — максимальное количество секретов у одного пользователя.
	MaxCount int
}

// secretService реализует SecretService.
type secretService struct {
	repo   domain.SecretRepository
	limits SecretLimits
}

// NewSecretService создаёт новый экземпляр secretService.
func NewSecretService(repo domain.SecretRepository, limits SecretLimits) SecretService {
	return &secretService{repo: repo, limits: limits}
}

// Create регистрирует новый секрет.
func (s *secretService) Create(ctx context.Context, userID uuid.UUID, blindIndex string, data []byte) error {
	if blindIndex == "" || len(data) == 0 {
		return domain.ErrInvalidArgument
	}
	if len(data) > s.limits.MaxSize {
		return domain.ErrSecretTooLarge
	}
	// Проверка не атомарна с вставкой: параллельные запросы могут превысить лимит
	// на единицы, что допустимо для защиты от заполнения диска.
	count, err := s.repo.CountByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if count >= s.limits.MaxCount {
		return domain.ErrSecretQuotaExceeded
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
	if len(data) > s.limits.MaxSize {
		return domain.ErrSecretTooLarge
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
