package grpchandler

import (
	"context"
	"testing"
	"time"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/F3dosik/GophKeeper/internal/server/mocks"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAdminHandler_DeleteUser(t *testing.T) {
	svc := mocks.NewAdminService(t)
	svc.On("DeleteUser", mock.Anything, "bob").Return(nil)
	svc.On("DeleteUser", mock.Anything, "ghost").Return(domain.ErrUserNotFound)
	handler := NewAdminHandler(svc)

	login := "bob"
	_, err := handler.DeleteUser(context.Background(), pb.DeleteUserRequest_builder{Login: &login}.Build())
	assert.NoError(t, err)

	login = "ghost"
	_, err = handler.DeleteUser(context.Background(), pb.DeleteUserRequest_builder{Login: &login}.Build())
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestAdminHandler_ListUsers(t *testing.T) {
	created := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	expires := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	svc := mocks.NewAdminService(t)
	svc.On("ListUsers", mock.Anything, "cursor", 10).Return(&domain.UserPage{
		Users: []*domain.UserInfo{
			{Login: "alice", CreatedAt: created, SecretCount: 3, KDF: domain.DefaultKDFParams},
			{Login: "bob", CreatedAt: created, PasswordExpiresAt: &expires, KDF: domain.DefaultKDFParams},
		},
		NextPageToken: "bob",
	}, nil)

	size, token := int32(10), "cursor"
	resp, err := NewAdminHandler(svc).ListUsers(context.Background(),
		pb.ListUsersRequest_builder{PageSize: &size, PageToken: &token}.Build())
	require.NoError(t, err)
	require.Len(t, resp.GetUsers(), 2)
	assert.Equal(t, "bob", resp.GetNextPageToken())

	alice, bob := resp.GetUsers()[0], resp.GetUsers()[1]
	assert.Equal(t, int32(3), alice.GetSecretCount())
	assert.False(t, alice.HasTemporaryPasswordExpiresAt(), "permanent password")
	assert.Equal(t, expires, bob.GetTemporaryPasswordExpiresAt().AsTime())
	assert.Equal(t, domain.DefaultKDFParams.Time, bob.GetKdfTime())
}

func TestAdminHandler_GetUserAndRevokeSessions(t *testing.T) {
	svc := mocks.NewAdminService(t)
	svc.On("GetUser", mock.Anything, "ghost").Return(nil, domain.ErrUserNotFound)
	svc.On("RevokeSessions", mock.Anything, "alice").Return(nil)
	handler := NewAdminHandler(svc)

	login := "ghost"
	_, err := handler.GetUser(context.Background(), pb.GetUserRequest_builder{Login: &login}.Build())
	assert.Equal(t, codes.NotFound, status.Code(err))

	login = "alice"
	_, err = handler.RevokeSessions(context.Background(), pb.RevokeSessionsRequest_builder{Login: &login}.Build())
	assert.NoError(t, err)
}
