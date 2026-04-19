package grpchandler

import (
	"context"

	"github.com/F3dosik/GophKeeper/internal/server/middleware"
	"github.com/F3dosik/GophKeeper/internal/server/service"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// secretHandler реализует интерфейс pb.SecretServer.
// Обрабатывает запросы создания, изменения, удаления и получения секретов.
type secretHandler struct {
	pb.UnimplementedSecretsServer
	secretService service.SecretService
}

// NewSecretHandler создает новый экземпляр secretHandler.
func NewSecretHandler(secretService service.SecretService) *secretHandler {
	return &secretHandler{secretService: secretService}
}

// CreateSecret обрабатывает запрос создания нового секрета.
// Возвращает codes.AlreadyExists если секрет с таким blind index уже существует.
// Возвращает codes.InvalidArgument если blind index или data пустые.
func (h *secretHandler) CreateSecret(ctx context.Context, req *pb.CreateSecretRequest) (*pb.CreateSecretResponse, error) {
	userID, err := middleware.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.secretService.Create(
		ctx, userID, req.GetItem().GetBlindIndex(), req.GetItem().GetData(),
	); err != nil {
		return nil, toGRPCError(err)
	}

	return pb.CreateSecretResponse_builder{}.Build(), nil
}

// UpdateSecret обрабатывает запрос обновления существующего секрета.
// Возвращает codes.NotFound если секрет не найден.
// Возвращает codes.InvalidArgument если blind index или data пустые.
func (h *secretHandler) UpdateSecret(ctx context.Context, req *pb.UpdateSecretRequest) (*pb.UpdateSecretResponse, error) {
	userID, err := middleware.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.secretService.Update(
		ctx, userID, req.GetItem().GetBlindIndex(), req.GetItem().GetData(),
	); err != nil {
		return nil, toGRPCError(err)
	}

	return pb.UpdateSecretResponse_builder{}.Build(), nil
}

// GetSecret обрабатывает запрос получения секрета по blind index.
// Возвращает codes.NotFound если секрет не найден.
func (h *secretHandler) GetSecret(ctx context.Context, req *pb.GetSecretRequest) (*pb.GetSecretResponse, error) {
	userID, err := middleware.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	secret, err := h.secretService.GetByBlindIndex(ctx, userID, req.GetBlindIndex())
	if err != nil {
		return nil, toGRPCError(err)
	}

	return pb.GetSecretResponse_builder{
		Data:      secret.Data,
		CreatedAt: timestamppb.New(secret.CreatedAt),
		UpdatedAt: timestamppb.New(secret.UpdatedAt),
	}.Build(), nil
}

// DeleteSecret обрабатывает запрос удаления секрета по blind index.
// Возвращает codes.NotFound если секрет не найден.
func (h *secretHandler) DeleteSecret(ctx context.Context, req *pb.DeleteSecretRequest) (*pb.DeleteSecretResponse, error) {
	userID, err := middleware.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.secretService.Delete(ctx, userID, req.GetBlindIndex()); err != nil {
		return nil, toGRPCError(err)
	}

	return pb.DeleteSecretResponse_builder{}.Build(), nil
}

// ListSecrets обрабатывает запрос получения всех секретов пользователя.
func (h *secretHandler) ListSecrets(ctx context.Context, req *pb.ListSecretsRequest) (*pb.ListSecretsResponse, error) {
	userID, err := middleware.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	secrets, err := h.secretService.ListByUserID(ctx, userID)
	if err != nil {
		return nil, toGRPCError(err)
	}

	var items []*pb.SecretItem
	for _, secret := range secrets {
		si := &pb.SecretItem{}
		si.SetBlindIndex(secret.BlindIndex)
		si.SetData(secret.Data)
		si.SetCreatedAt(timestamppb.New(secret.CreatedAt))
		si.SetUpdatedAt(timestamppb.New(secret.UpdatedAt))
		items = append(items, si)
	}

	return pb.ListSecretsResponse_builder{Items: items}.Build(), nil
}
