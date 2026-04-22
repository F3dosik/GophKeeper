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
		return domain.ErrInvalidArgument
	default:
		return fmt.Errorf("internal: %s", st.Message())
	}
}
