package grpcclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// authInterceptor возвращает unary-интерцептор, прикрепляющий JWT токен
// к каждому исходящему RPC-вызову в заголовке Authorization в формате "Bearer <token>".
//
// Если token пуст (пользователь не аутентифицирован), заголовок не добавляется —
// это нужно для методов, не требующих авторизации (CreateUser, GetSalt, Login).
func authInterceptor(token string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if token != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, tokenMetadataKey, "Bearer "+token)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
