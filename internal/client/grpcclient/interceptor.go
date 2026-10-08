package grpcclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// authInterceptor возвращает unary-интерцептор, прикрепляющий текущий JWT токен из tokens
// к каждому исходящему RPC-вызову в заголовке Authorization в формате "Bearer <token>".
//
// Если токен пуст (пользователь не аутентифицирован), заголовок не добавляется —
// это нужно для методов, не требующих авторизации (CreateUser, GetSalt, Login).
func authInterceptor(tokens *TokenStore) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if token := tokens.Token(); token != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, tokenMetadataKey, "Bearer "+token)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
