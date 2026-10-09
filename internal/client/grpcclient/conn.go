package grpcclient

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

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

// ErrBadCACert возвращается, если в переданном CA-сертификате нет ни одного PEM-сертификата.
var ErrBadCACert = errors.New("no PEM certificates found in CA certificate")

// DialOptions — параметры соединения с сервером.
type DialOptions struct {
	// CACertPEM — корневой сертификат (PEM), которым проверяется сертификат сервера.
	// Пустой — используются системные корневые сертификаты.
	CACertPEM []byte
	// Insecure разрешает соединение без TLS — допустимо лишь для локальной разработки.
	Insecure bool
	// Tokens — токен, прикрепляемый к запросам; может быть nil.
	Tokens *TokenStore
}

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
	opts := DialOptions{Insecure: allowInsecure, Tokens: tokens}
	if tlsCertPath != "" {
		pem, err := os.ReadFile(tlsCertPath)
		if err != nil {
			return nil, fmt.Errorf("dial: read CA certificate: %w", err)
		}
		opts.CACertPEM = pem
	}
	return DialWithOptions(serverAddr, opts)
}

// DialWithOptions — как Dial, но CA-сертификат передаётся содержимым, а не путём к файлу.
// Нужен клиентам, у которых сертификат встроен в приложение или хранится в настройках.
func DialWithOptions(serverAddr string, opts DialOptions) (*grpc.ClientConn, error) {
	creds, err := transportCredentials(opts.CACertPEM, opts.Insecure)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	return grpc.NewClient(serverAddr,
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxRecvMsgSize)),
		grpc.WithUnaryInterceptor(authInterceptor(opts.Tokens)),
		grpc.WithStreamInterceptor(authStreamInterceptor(opts.Tokens)),
	)
}

// transportCredentials выбирает транспорт по настройкам клиента.
func transportCredentials(caCertPEM []byte, allowInsecure bool) (credentials.TransportCredentials, error) {
	switch {
	case allowInsecure && len(caCertPEM) > 0:
		return nil, ErrInsecureWithCert
	case allowInsecure:
		return insecure.NewCredentials(), nil
	case len(caCertPEM) > 0:
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCertPEM) {
			return nil, ErrBadCACert
		}
		return credentials.NewTLS(&tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}), nil
	default:
		// RootCAs == nil — используются системные корневые сертификаты.
		return credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12}), nil
	}
}
