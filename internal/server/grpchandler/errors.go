package grpchandler

import (
	"errors"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// toGRPCError конвертирует ошибки domain в gRPC.
func toGRPCError(err error) error {
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrUserAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, domain.ErrSecretNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrSecretAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
