package service

import (
	"context"
	"testing"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/F3dosik/GophKeeper/internal/server/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func users(logins ...string) []*domain.UserInfo {
	result := make([]*domain.UserInfo, len(logins))
	for i, l := range logins {
		result[i] = &domain.UserInfo{Login: l}
	}
	return result
}

func TestAdminService_ListUsers(t *testing.T) {
	ctx := context.Background()

	t.Run("last page", func(t *testing.T) {
		repo := mocks.NewUserRepository(t)
		repo.On("ListInfo", mock.Anything, "", MaxUsersPageSize+1).Return(users("a", "b"), nil)

		page, err := NewAdminService(repo, mocks.NewTokenRepository(t)).ListUsers(ctx, "", 0)
		require.NoError(t, err)
		assert.Len(t, page.Users, 2)
		assert.Empty(t, page.NextPageToken)
	})

	t.Run("more pages", func(t *testing.T) {
		repo := mocks.NewUserRepository(t)
		repo.On("ListInfo", mock.Anything, "a", 3).Return(users("b", "c", "d"), nil)

		page, err := NewAdminService(repo, mocks.NewTokenRepository(t)).ListUsers(ctx, "a", 2)
		require.NoError(t, err)
		assert.Len(t, page.Users, 2)
		assert.Equal(t, "c", page.NextPageToken)
	})

	t.Run("invalid arguments", func(t *testing.T) {
		svc := NewAdminService(mocks.NewUserRepository(t), mocks.NewTokenRepository(t))
		_, err := svc.ListUsers(ctx, "", MaxUsersPageSize+1)
		assert.ErrorIs(t, err, domain.ErrInvalidArgument)
		_, err = svc.ListUsers(ctx, "bad\ntoken", 0)
		assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	})
}

func TestAdminService_UserOperations(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewUserRepository(t)
	tokens := mocks.NewTokenRepository(t)
	repo.On("GetInfo", mock.Anything, "alice").Return(&domain.UserInfo{Login: "alice", SecretCount: 2}, nil)
	repo.On("DeleteByLogin", mock.Anything, "ghost").Return(domain.ErrUserNotFound)
	tokens.On("RevokeAllByLogin", mock.Anything, "alice").Return(nil)
	svc := NewAdminService(repo, tokens)

	info, err := svc.GetUser(ctx, "alice")
	require.NoError(t, err)
	assert.Equal(t, 2, info.SecretCount)

	assert.NoError(t, svc.RevokeSessions(ctx, "alice"))
	assert.ErrorIs(t, svc.DeleteUser(ctx, "ghost"), domain.ErrUserNotFound)

	// Неверный логин отклоняется до обращения к БД.
	assert.ErrorIs(t, svc.DeleteUser(ctx, ""), domain.ErrInvalidArgument)
	_, err = svc.GetUser(ctx, "")
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	assert.ErrorIs(t, svc.RevokeSessions(ctx, ""), domain.ErrInvalidArgument)
}
