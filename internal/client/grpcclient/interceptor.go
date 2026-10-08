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

// authStreamInterceptor — потоковый аналог authInterceptor.
func authStreamInterceptor(tokens *TokenStore) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		if token := tokens.Token(); token != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, tokenMetadataKey, "Bearer "+token)
		}
		return streamer(ctx, desc, cc, method, opts...)
	}
}
