package grpcclient

import (
	"fmt"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func fromGRPCError(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return fmt.Errorf("grpcclient: %w", err)
	}
	switch st.Code() {
	case codes.NotFound:
		return domain.ErrNotFound
	case codes.AlreadyExists:
		return domain.ErrAlreadyExists
	case codes.Unauthenticated:
		return domain.ErrInvalidCredentials
	case codes.InvalidArgument:
		return fmt.Errorf("%w: %s", domain.ErrInvalidArgument, st.Message())
	case codes.Aborted:
		return domain.ErrSecretsChanged
	case codes.ResourceExhausted:
		return fmt.Errorf("%w: %s", domain.ErrResourceExhausted, st.Message())
	default:
		return fmt.Errorf("internal: %s", st.Message())
	}
}
