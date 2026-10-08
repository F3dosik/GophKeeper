package service

import (
	"context"
	"fmt"

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

	// ListPage возвращает страницу секретов пользователя начиная с курсора pageToken
	// (пустой — с начала). pageSize 0 означает размер по умолчанию.
	// Возвращает ErrInvalidArgument при неверном pageSize или pageToken.
	ListPage(ctx context.Context, userID uuid.UUID, pageToken string, pageSize int) (*domain.SecretPage, error)

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
	if err := validateBlindIndex(blindIndex); err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("%w: secret data is empty", domain.ErrInvalidArgument)
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
	if err := validateBlindIndex(blindIndex); err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("%w: secret data is empty", domain.ErrInvalidArgument)
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
	if err := validateBlindIndex(blindIndex); err != nil {
		return nil, err
	}
	return s.repo.GetByBlindIndex(ctx, userID, blindIndex)
}

// Ограничения страницы списка секретов.
const (
	// MaxPageSize — максимум секретов на странице.
	MaxPageSize = 100
	// MaxPageBytes — ориентир на суммарный размер данных страницы; страница всегда
	// содержит хотя бы один секрет, даже если он больше.
	MaxPageBytes = 4 << 20
)

// ListPage возвращает страницу секретов. Страница ограничена и количеством, и суммарным
// размером данных, чтобы ответ помещался в лимит размера gRPC-сообщения клиента.
func (s *secretService) ListPage(
	ctx context.Context, userID uuid.UUID, pageToken string, pageSize int,
) (*domain.SecretPage, error) {
	if pageSize == 0 {
		pageSize = MaxPageSize
	}
	if pageSize < 0 || pageSize > MaxPageSize {
		return nil, fmt.Errorf("%w: page size must be 1-%d", domain.ErrInvalidArgument, MaxPageSize)
	}

	after := uuid.Nil
	if pageToken != "" {
		var err error
		if after, err = uuid.Parse(pageToken); err != nil {
			return nil, fmt.Errorf("%w: bad page token", domain.ErrInvalidArgument)
		}
	}

	// Запрашиваем на один больше, чтобы понять, есть ли следующая страница.
	rows, err := s.repo.ListByUserID(ctx, userID, after, pageSize+1)
	if err != nil {
		return nil, err
	}

	page := &domain.SecretPage{Secrets: make([]*domain.Secret, 0, min(len(rows), pageSize))}
	size := 0
	for _, secret := range rows {
		full := len(page.Secrets) == pageSize ||
			(len(page.Secrets) > 0 && size+len(secret.Data) > MaxPageBytes)
		if full {
			page.NextPageToken = page.Secrets[len(page.Secrets)-1].ID.String()
			break
		}
		page.Secrets = append(page.Secrets, secret)
		size += len(secret.Data)
	}
	return page, nil
}

// Delete удаляет секрет по userID и blindIndex.
func (s *secretService) Delete(ctx context.Context, userID uuid.UUID, blindIndex string) error {
	if err := validateBlindIndex(blindIndex); err != nil {
		return err
	}
	return s.repo.Delete(ctx, userID, blindIndex)
}
