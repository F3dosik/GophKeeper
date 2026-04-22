package grpcclient

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Dial устанавливает gRPC-соединение с сервером по адресу serverAddr.
//
// Если tlsCertPath не пуст, используется TLS с корневым сертификатом из указанного файла;
// иначе соединение устанавливается без шифрования (insecure) — допустимо только для локальной разработки.
//
// token, если не пуст, прикрепляется к каждому исходящему RPC-вызову через
// authInterceptor в заголовке Authorization. Закрытие соединения — ответственность вызывающего.
func Dial(serverAddr, tlsCertPath, token string) (*grpc.ClientConn, error) {
	var creds credentials.TransportCredentials
	if tlsCertPath != "" {
		c, err := credentials.NewClientTLSFromFile(tlsCertPath, "")
		if err != nil {
			return nil, fmt.Errorf("dial: %w", err)
		}
		creds = c
	} else {
		creds = insecure.NewCredentials()
	}
	return grpc.NewClient(serverAddr,
		grpc.WithTransportCredentials(creds),
		grpc.WithUnaryInterceptor(authInterceptor(token)),
	)
}
