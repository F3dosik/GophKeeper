package grpcclient

import (
	"github.com/F3dosik/GophKeeper/internal/domain"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
)

// tokenMetadataKey используется как ключ заголовка авторизации в gRPC metadata.
const tokenMetadataKey = "authorization"

func toPBCredentials(c domain.Credentials) *pb.Credentials {
	return pb.Credentials_builder{
		Login:     &c.Login,
		MasterKey: c.MasterKey,
	}.Build()
}

func fromPBSecrets(items []*pb.SecretItem) []*domain.Secret {
	secrets := make([]*domain.Secret, 0, len(items))
	for _, item := range items {
		secrets = append(secrets, &domain.Secret{
			BlindIndex: item.GetBlindIndex(),
			Data:       item.GetData(),
			CreatedAt:  item.GetCreatedAt().AsTime(),
			UpdatedAt:  item.GetCreatedAt().AsTime(),
		})
	}
	return secrets
}
