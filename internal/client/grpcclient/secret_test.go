package grpcclient_test

import (
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

func TestSecretsClient_ListSecrets(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockPB := mocks.NewPBSecretsClient(t)
		blindIndex := "blind-index"
		mockPB.On("ListSecrets", mock.Anything, mock.Anything, mock.Anything).
			Return(pb.ListSecretsResponse_builder{
				Items: []*pb.SecretItem{
					pb.SecretItem_builder{
						BlindIndex: &blindIndex,
						Data:       []byte("encrypted"),
					}.Build(),
				},
			}.Build(), nil)

		client := grpcclient.NewSecretsClient(mockPB)
		secrets, err := client.ListSecrets(context.Background())

		require.NoError(t, err)
		require.Len(t, secrets, 1)
		assert.Equal(t, blindIndex, secrets[0].BlindIndex)
		assert.Equal(t, []byte("encrypted"), secrets[0].Data)
	})

	t.Run("empty list", func(t *testing.T) {
		mockPB := mocks.NewPBSecretsClient(t)
		mockPB.On("ListSecrets", mock.Anything, mock.Anything, mock.Anything).
			Return(pb.ListSecretsResponse_builder{Items: nil}.Build(), nil)

		client := grpcclient.NewSecretsClient(mockPB)
		secrets, err := client.ListSecrets(context.Background())

		require.NoError(t, err)
		assert.Empty(t, secrets)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		mockPB := mocks.NewPBSecretsClient(t)
		mockPB.On("ListSecrets", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.Unauthenticated, "unauthenticated"))

		client := grpcclient.NewSecretsClient(mockPB)
		_, err := client.ListSecrets(context.Background())

		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("internal error", func(t *testing.T) {
		mockPB := mocks.NewPBSecretsClient(t)
		mockPB.On("ListSecrets", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.Internal, "internal error"))

		client := grpcclient.NewSecretsClient(mockPB)
		_, err := client.ListSecrets(context.Background())

		assert.Error(t, err)
	})
}

func TestSecretsClient_CreateSecret(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockPB := mocks.NewPBSecretsClient(t)
		mockPB.On("CreateSecret", mock.Anything, mock.MatchedBy(func(req *pb.CreateSecretRequest) bool {
			return req.GetItem().GetBlindIndex() == "blind-index" &&
				string(req.GetItem().GetData()) == "encrypted"
		}), mock.Anything).Return(&pb.CreateSecretResponse{}, nil)

		client := grpcclient.NewSecretsClient(mockPB)
		err := client.CreateSecret(context.Background(), "blind-index", []byte("encrypted"))

		require.NoError(t, err)
	})

	t.Run("already exists", func(t *testing.T) {
		mockPB := mocks.NewPBSecretsClient(t)
		mockPB.On("CreateSecret", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.AlreadyExists, "secret already exists"))

		client := grpcclient.NewSecretsClient(mockPB)
		err := client.CreateSecret(context.Background(), "blind-index", []byte("encrypted"))

		assert.ErrorIs(t, err, domain.ErrAlreadyExists)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		mockPB := mocks.NewPBSecretsClient(t)
		mockPB.On("CreateSecret", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.Unauthenticated, "unauthenticated"))

		client := grpcclient.NewSecretsClient(mockPB)
		err := client.CreateSecret(context.Background(), "blind-index", []byte("encrypted"))

		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})
}

func TestSecretsClient_UpdateSecret(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockPB := mocks.NewPBSecretsClient(t)
		mockPB.On("UpdateSecret", mock.Anything, mock.MatchedBy(func(req *pb.UpdateSecretRequest) bool {
			return req.GetItem().GetBlindIndex() == "blind-index" &&
				string(req.GetItem().GetData()) == "encrypted"
		}), mock.Anything).Return(&pb.UpdateSecretResponse{}, nil)

		client := grpcclient.NewSecretsClient(mockPB)
		err := client.UpdateSecret(context.Background(), "blind-index", []byte("encrypted"))

		require.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		mockPB := mocks.NewPBSecretsClient(t)
		mockPB.On("UpdateSecret", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.NotFound, "secret not found"))

		client := grpcclient.NewSecretsClient(mockPB)
		err := client.UpdateSecret(context.Background(), "blind-index", []byte("encrypted"))

		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		mockPB := mocks.NewPBSecretsClient(t)
		mockPB.On("UpdateSecret", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.Unauthenticated, "unauthenticated"))

		client := grpcclient.NewSecretsClient(mockPB)
		err := client.UpdateSecret(context.Background(), "blind-index", []byte("encrypted"))

		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})
}

func TestSecretsClient_GetSecret(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockPB := mocks.NewPBSecretsClient(t)
		mockPB.On("GetSecret", mock.Anything, mock.MatchedBy(func(req *pb.GetSecretRequest) bool {
			return req.GetBlindIndex() == "blind-index"
		}), mock.Anything).Return(
			pb.GetSecretResponse_builder{Data: []byte("encrypted")}.Build(), nil,
		)

		client := grpcclient.NewSecretsClient(mockPB)
		secret, err := client.GetSecret(context.Background(), "blind-index")

		require.NoError(t, err)
		assert.Equal(t, []byte("encrypted"), secret.Data)
	})

	t.Run("not found", func(t *testing.T) {
		mockPB := mocks.NewPBSecretsClient(t)
		mockPB.On("GetSecret", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.NotFound, "secret not found"))

		client := grpcclient.NewSecretsClient(mockPB)
		_, err := client.GetSecret(context.Background(), "blind-index")

		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		mockPB := mocks.NewPBSecretsClient(t)
		mockPB.On("GetSecret", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.Unauthenticated, "unauthenticated"))

		client := grpcclient.NewSecretsClient(mockPB)
		_, err := client.GetSecret(context.Background(), "blind-index")

		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})
}

func TestSecretsClient_DeleteSecret(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockPB := mocks.NewPBSecretsClient(t)
		mockPB.On("DeleteSecret", mock.Anything, mock.MatchedBy(func(req *pb.DeleteSecretRequest) bool {
			return req.GetBlindIndex() == "blind-index"
		}), mock.Anything).Return(&pb.DeleteSecretResponse{}, nil)

		client := grpcclient.NewSecretsClient(mockPB)
		err := client.DeleteSecret(context.Background(), "blind-index")

		require.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		mockPB := mocks.NewPBSecretsClient(t)
		mockPB.On("DeleteSecret", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.NotFound, "secret not found"))

		client := grpcclient.NewSecretsClient(mockPB)
		err := client.DeleteSecret(context.Background(), "blind-index")

		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		mockPB := mocks.NewPBSecretsClient(t)
		mockPB.On("DeleteSecret", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.Unauthenticated, "unauthenticated"))

		client := grpcclient.NewSecretsClient(mockPB)
		err := client.DeleteSecret(context.Background(), "blind-index")

		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})
}
