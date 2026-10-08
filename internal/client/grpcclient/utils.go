package grpcclient

import (
	"github.com/F3dosik/GophKeeper/internal/domain"
	pb "github.com/F3dosik/GophKeeper/proto/gen"
)

// tokenMetadataKey используется как ключ заголовка авторизации в gRPC metadata.
const tokenMetadataKey = "authorization"

func toPBCredentials(c domain.Credentials) *pb.Credentials {
	return pb.Credentials_builder{
		Login:   &c.Login,
		AuthKey: c.AuthKey,
	}.Build()
}

func fromPBSecrets(items []*pb.SecretItem) []*domain.Secret {
	secrets := make([]*domain.Secret, 0, len(items))
	for _, item := range items {
		secrets = append(secrets, &domain.Secret{
			BlindIndex: item.GetBlindIndex(),
			Data:       item.GetData(),
			CreatedAt:  item.GetCreatedAt().AsTime(),
			UpdatedAt:  item.GetUpdatedAt().AsTime(),
		})
	}
	return secrets
}

// toPBKDF переводит параметры Argon2id в protobuf.
func toPBKDF(p domain.KDFParams) *pb.KDFParams {
	threads := uint32(p.Threads)
	return pb.KDFParams_builder{Time: &p.Time, MemoryKib: &p.MemoryKiB, Threads: &threads}.Build()
}

// fromPBKDF переводит параметры Argon2id из protobuf. Отсутствующие параметры дают
// нулевое значение, которое отклонит KDFParams.Validate; слишком большое число потоков
// насыщается до 255 по той же причине.
func fromPBKDF(p *pb.KDFParams) domain.KDFParams {
	threads := p.GetThreads()
	if threads > 255 {
		threads = 255
	}
	return domain.KDFParams{Time: p.GetTime(), MemoryKiB: p.GetMemoryKib(), Threads: uint8(threads)}
}
