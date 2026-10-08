package grpcclient_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/F3dosik/GophKeeper/internal/client/grpcclient"
	"github.com/F3dosik/GophKeeper/internal/client/mocks"
	"github.com/F3dosik/GophKeeper/internal/domain"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	hash     = []byte("hash")
	testSalt = []byte("salt")
	jwtToken = "jwtToken"
)

func TestAuthClient_CreateUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockPB := mocks.NewPBAuthClient(t)
		mockPB.On("CreateUser", mock.Anything, mock.MatchedBy(func(req *pb.CreateUserRequest) bool {
			return req.GetCredentials().GetLogin() == "user" &&
				bytes.Equal(req.GetCredentials().GetAuthKey(), hash) &&
				bytes.Equal(req.GetSalt(), testSalt) &&
				req.GetKdf().GetTime() == domain.DefaultKDFParams.Time
		}), mock.Anything).Return(&pb.CreateUserResponse{}, nil)

		client := grpcclient.NewAuthClient(mockPB)
		err := client.CreateUser(context.Background(), domain.Credentials{
			Login:   "user",
			AuthKey: hash,
		}, testSalt, domain.DefaultKDFParams)

		require.NoError(t, err)
	})

	t.Run("user already exists", func(t *testing.T) {
		mockPB := mocks.NewPBAuthClient(t)
		mockPB.On("CreateUser", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.AlreadyExists, "user already exists"))

		client := grpcclient.NewAuthClient(mockPB)
		err := client.CreateUser(context.Background(), domain.Credentials{}, testSalt, domain.DefaultKDFParams)

		assert.ErrorIs(t, err, domain.ErrAlreadyExists)
	})

	t.Run("invalid argument", func(t *testing.T) {
		mockPB := mocks.NewPBAuthClient(t)
		mockPB.On("CreateUser", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.InvalidArgument, "invalid argument"))

		client := grpcclient.NewAuthClient(mockPB)
		err := client.CreateUser(context.Background(), domain.Credentials{}, []byte{}, domain.DefaultKDFParams)

		assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	})

	t.Run("internal error", func(t *testing.T) {
		mockPB := mocks.NewPBAuthClient(t)
		mockPB.On("CreateUser", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.Internal, "internal error"))

		client := grpcclient.NewAuthClient(mockPB)
		err := client.CreateUser(context.Background(), domain.Credentials{}, testSalt, domain.DefaultKDFParams)

		assert.Error(t, err)
	})
}

func TestAuthClient_GetSalt(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockPB := mocks.NewPBAuthClient(t)
		mockPB.On("GetSalt", mock.Anything, mock.MatchedBy(func(req *pb.GetSaltRequest) bool {
			return req.GetLogin() == "user"
		}), mock.Anything).Return(
			pb.GetSaltResponse_builder{Salt: testSalt, Kdf: pbKDF(domain.DefaultKDFParams)}.Build(), nil,
		)

		client := grpcclient.NewAuthClient(mockPB)
		salt, kdf, err := client.GetSalt(context.Background(), "user")

		require.NoError(t, err)
		assert.Equal(t, testSalt, salt)
		assert.Equal(t, domain.DefaultKDFParams, kdf)
	})

	t.Run("user not found", func(t *testing.T) {
		mockPB := mocks.NewPBAuthClient(t)
		mockPB.On("GetSalt", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.NotFound, "user not found"))

		client := grpcclient.NewAuthClient(mockPB)
		_, _, err := client.GetSalt(context.Background(), "unknown")

		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("internal error", func(t *testing.T) {
		mockPB := mocks.NewPBAuthClient(t)
		mockPB.On("GetSalt", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.Internal, "internal error"))

		client := grpcclient.NewAuthClient(mockPB)
		_, _, err := client.GetSalt(context.Background(), "user")

		assert.Error(t, err)
	})
}

func TestAuthClient_Login(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockPB := mocks.NewPBAuthClient(t)
		mockPB.On("Login", mock.Anything, mock.MatchedBy(func(req *pb.LoginRequest) bool {
			return req.GetCredentials().GetLogin() == "user" &&
				bytes.Equal(req.GetCredentials().GetAuthKey(), hash)
		}), mock.Anything).Return(
			pb.LoginResponse_builder{Token: &jwtToken}.Build(), nil,
		)

		client := grpcclient.NewAuthClient(mockPB)
		token, err := client.Login(context.Background(), domain.Credentials{
			Login:   "user",
			AuthKey: hash,
		})

		require.NoError(t, err)
		assert.Equal(t, jwtToken, token)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		mockPB := mocks.NewPBAuthClient(t)
		mockPB.On("Login", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.Unauthenticated, "invalid credentials"))

		client := grpcclient.NewAuthClient(mockPB)
		_, err := client.Login(context.Background(), domain.Credentials{})

		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("user not found", func(t *testing.T) {
		mockPB := mocks.NewPBAuthClient(t)
		mockPB.On("Login", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.NotFound, "user not found"))

		client := grpcclient.NewAuthClient(mockPB)
		_, err := client.Login(context.Background(), domain.Credentials{})

		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("internal error", func(t *testing.T) {
		mockPB := mocks.NewPBAuthClient(t)
		mockPB.On("Login", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.Internal, "internal error"))

		client := grpcclient.NewAuthClient(mockPB)
		_, err := client.Login(context.Background(), domain.Credentials{})

		assert.Error(t, err)
	})
}

// pbKDF собирает protobuf-параметры Argon2id для ответов мока.
func pbKDF(p domain.KDFParams) *pb.KDFParams {
	threads := uint32(p.Threads)
	return pb.KDFParams_builder{Time: &p.Time, MemoryKib: &p.MemoryKiB, Threads: &threads}.Build()
}
