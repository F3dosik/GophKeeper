package grpcclient

import (
	"crypto/tls"
	"errors"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// ErrInsecureWithCert возвращается, если одновременно запрошены режим без TLS и CA-сертификат.
var ErrInsecureWithCert = errors.New("insecure mode and TLS certificate are mutually exclusive")

// Dial устанавливает gRPC-соединение с сервером по адресу serverAddr.
//
// По умолчанию используется TLS: если tlsCertPath не пуст, сертификат сервера проверяется
// корневым сертификатом из указанного файла, иначе — системным хранилищем сертификатов.
// Соединение без шифрования устанавливается только при allowInsecure == true —
// допустимо лишь для локальной разработки.
//
// token, если не пуст, прикрепляется к каждому исходящему RPC-вызову через
// authInterceptor в заголовке Authorization. Закрытие соединения — ответственность вызывающего.
func Dial(serverAddr, tlsCertPath string, allowInsecure bool, token string) (*grpc.ClientConn, error) {
	creds, err := transportCredentials(tlsCertPath, allowInsecure)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	return grpc.NewClient(serverAddr,
		grpc.WithTransportCredentials(creds),
		grpc.WithUnaryInterceptor(authInterceptor(token)),
	)
}

// transportCredentials выбирает транспорт по настройкам клиента.
func transportCredentials(tlsCertPath string, allowInsecure bool) (credentials.TransportCredentials, error) {
	switch {
	case allowInsecure && tlsCertPath != "":
		return nil, ErrInsecureWithCert
	case allowInsecure:
		return insecure.NewCredentials(), nil
	case tlsCertPath != "":
		return credentials.NewClientTLSFromFile(tlsCertPath, "")
	default:
		// RootCAs == nil — используются системные корневые сертификаты.
		return credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12}), nil
	}
}
