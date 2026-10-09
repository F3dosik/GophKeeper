package grpcclient_test

import (
	"context"
	"testing"
	"time"

	"github.com/F3dosik/GophKeeper/internal/client/grpcclient"
	"github.com/F3dosik/GophKeeper/internal/client/mocks"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAdminClient_ListUsers_FollowsPages(t *testing.T) {
	expires := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	user := func(login string, temporary bool) *pb.UserInfo {
		b := pb.UserInfo_builder{Login: &login}
		if temporary {
			b.TemporaryPasswordExpiresAt = timestamppb.New(expires)
		}
		return b.Build()
	}
	page := func(next string, users ...*pb.UserInfo) *pb.ListUsersResponse {
		return pb.ListUsersResponse_builder{Users: users, NextPageToken: &next}.Build()
	}
	withToken := func(token string) any {
		return mock.MatchedBy(func(r *pb.ListUsersRequest) bool { return r.GetPageToken() == token })
	}

	m := mocks.NewPBAdminClient(t)
	m.On("ListUsers", mock.Anything, withToken(""), mock.Anything).Return(page("alice", user("alice", false)), nil)
	m.On("ListUsers", mock.Anything, withToken("alice"), mock.Anything).Return(page("", user("bob", true)), nil)

	users, err := grpcclient.NewAdminClient(m).ListUsers(context.Background())
	require.NoError(t, err)
	require.Len(t, users, 2)
	assert.Nil(t, users[0].PasswordExpiresAt)
	require.NotNil(t, users[1].PasswordExpiresAt)
	assert.Equal(t, expires, *users[1].PasswordExpiresAt)
}
