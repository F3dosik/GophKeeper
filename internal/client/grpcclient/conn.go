package grpcclient

import (
	"crypto/tls"
	"errors"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// maxRecvMsgSize — лимит размера ответа сервера. Страница списка ограничена ~4 MiB,
// но всегда содержит хотя бы один секрет, а секрет может быть до SECRET_MAX_SIZE
// (сервер проверяет, что он не больше 32 MiB).
const maxRecvMsgSize = 64 << 20

// ErrInsecureWithCert возвращается, если одновременно запрошены режим без TLS и CA-сертификат.
var ErrInsecureWithCert = errors.New("insecure mode and TLS certificate are mutually exclusive")

// Dial устанавливает gRPC-соединение с сервером по адресу serverAddr.
//
// По умолчанию используется TLS: если tlsCertPath не пуст, сертификат сервера проверяется
// корневым сертификатом из указанного файла, иначе — системным хранилищем сертификатов.
// Соединение без шифрования устанавливается только при allowInsecure == true —
// допустимо лишь для локальной разработки.
//
// Текущий токен из tokens, если не пуст, прикрепляется к каждому исходящему RPC-вызову
// через authInterceptor в заголовке Authorization. Закрытие соединения — ответственность вызывающего.
func Dial(serverAddr, tlsCertPath string, allowInsecure bool, tokens *TokenStore) (*grpc.ClientConn, error) {
	creds, err := transportCredentials(tlsCertPath, allowInsecure)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	return grpc.NewClient(serverAddr,
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxRecvMsgSize)),
		grpc.WithUnaryInterceptor(authInterceptor(tokens)),
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
