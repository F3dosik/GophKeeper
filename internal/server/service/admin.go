package service

import (
	"context"
	"fmt"

	"github.com/F3dosik/GophKeeper/internal/domain"
)

// MaxUsersPageSize — максимум пользователей на странице списка.
const MaxUsersPageSize = 100

// AdminService — административные операции. Доступен только на административном
// порту сервера. Содержимое секретов администратору недоступно: оно зашифровано
// ключами пользователей.
type AdminService interface {
	// ListUsers возвращает страницу пользователей по возрастанию логина.
	// pageSize 0 означает размер по умолчанию; pageToken — курсор предыдущей страницы.
	ListUsers(ctx context.Context, pageToken string, pageSize int) (*domain.UserPage, error)

	// GetUser возвращает сведения о пользователе. Возвращает ErrUserNotFound, если его нет.
	GetUser(ctx context.Context, login string) (*domain.UserInfo, error)

	// RevokeSessions отзывает все токены пользователя. Возвращает ErrUserNotFound, если его нет.
	RevokeSessions(ctx context.Context, login string) error

	// DeleteUser удаляет пользователя со всеми секретами. Возвращает ErrUserNotFound, если его нет.
	DeleteUser(ctx context.Context, login string) error
}

type adminService struct {
	users  domain.UserRepository
	tokens domain.TokenRepository
}

// NewAdminService создаёт AdminService.
func NewAdminService(users domain.UserRepository, tokens domain.TokenRepository) AdminService {
	return &adminService{users: users, tokens: tokens}
}

// ListUsers возвращает страницу пользователей. Курсор — логин последнего
// пользователя предыдущей страницы.
func (s *adminService) ListUsers(ctx context.Context, pageToken string, pageSize int) (*domain.UserPage, error) {
	if pageSize == 0 {
		pageSize = MaxUsersPageSize
	}
	if pageSize < 0 || pageSize > MaxUsersPageSize {
		return nil, fmt.Errorf("%w: page size must be 1-%d", domain.ErrInvalidArgument, MaxUsersPageSize)
	}
	if pageToken != "" {
		if err := validateLogin(pageToken); err != nil {
			return nil, fmt.Errorf("%w: bad page token", domain.ErrInvalidArgument)
		}
	}

	// Запрашиваем на одного больше, чтобы понять, есть ли следующая страница.
	users, err := s.users.ListInfo(ctx, pageToken, pageSize+1)
	if err != nil {
		return nil, err
	}
	page := &domain.UserPage{Users: users}
	if len(users) > pageSize {
		page.Users = users[:pageSize]
		page.NextPageToken = page.Users[pageSize-1].Login
	}
	return page, nil
}

// GetUser возвращает сведения о пользователе.
func (s *adminService) GetUser(ctx context.Context, login string) (*domain.UserInfo, error) {
	if err := validateLogin(login); err != nil {
		return nil, err
	}
	return s.users.GetInfo(ctx, login)
}

// RevokeSessions отзывает все токены пользователя.
func (s *adminService) RevokeSessions(ctx context.Context, login string) error {
	if err := validateLogin(login); err != nil {
		return err
	}
	return s.tokens.RevokeAllByLogin(ctx, login)
}

// DeleteUser удаляет пользователя по логину.
func (s *adminService) DeleteUser(ctx context.Context, login string) error {
	if err := validateLogin(login); err != nil {
		return err
	}
	return s.users.DeleteByLogin(ctx, login)
}
