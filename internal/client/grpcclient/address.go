package grpcclient

import (
	"errors"
	"net"
	"strconv"
	"strings"
)

// DefaultPort — порт сервера GophKeeper по умолчанию.
const DefaultPort = "50051"

// ErrBadAddress возвращается для пустого или некорректного адреса сервера.
var ErrBadAddress = errors.New("invalid server address")

// NormalizeAddress приводит адрес сервера к виду host:port. Порт необязателен:
// без него подставляется DefaultPort. Поддерживаются имена, IPv4 и IPv6
// (с портом — в квадратных скобках: [fe80::1]:50051).
func NormalizeAddress(address string) (string, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return "", ErrBadAddress
	}

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		// Порт не указан: «host», «[::1]» или IPv6 без скобок («fe80::1»).
		host = strings.TrimSuffix(strings.TrimPrefix(address, "["), "]")
		if strings.Contains(host, ":") && net.ParseIP(host) == nil {
			return "", ErrBadAddress
		}
		port = DefaultPort
	}
	if host == "" || strings.ContainsAny(host, " /") {
		return "", ErrBadAddress
	}
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
		return "", ErrBadAddress
	}
	return net.JoinHostPort(host, port), nil
}
