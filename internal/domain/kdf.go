package domain

import "fmt"

// KDFParams — параметры Argon2id, с которыми из пароля выводится мастер-ключ.
// Хранятся у каждого пользователя рядом с солью: так стоимость деривации можно
// повышать для новых пользователей и при смене пароля, не ломая существующие ключи.
type KDFParams struct {
	// Time — число проходов по памяти.
	Time uint32
	// MemoryKiB — объём памяти в КиБ.
	MemoryKiB uint32
	// Threads — степень параллелизма.
	Threads uint8
}

var (
	// DefaultKDFParams — параметры для новых пользователей и смены пароля
	// (второй рекомендуемый вариант RFC 9106: t=3, m=64 MiB).
	DefaultKDFParams = KDFParams{Time: 3, MemoryKiB: 64 * 1024, Threads: 4}

	// LegacyKDFParams — параметры, с которыми созданы пользователи до появления
	// хранения параметров. Миграция 000004 проставляет их существующим пользователям.
	LegacyKDFParams = KDFParams{Time: 1, MemoryKiB: 64 * 1024, Threads: 4}
)

// Границы допустимых параметров. Нижние не дают ослабить деривацию (их проверяет и
// клиент: подменённый сервер мог бы прислать слабые параметры, чтобы удешевить перебор),
// верхние — заставить клиента считать часами или исчерпать память.
const (
	minKDFTime      = 1
	maxKDFTime      = 10
	minKDFMemoryKiB = 19 * 1024 // минимум OWASP для Argon2id
	maxKDFMemoryKiB = 1024 * 1024
	minKDFThreads   = 1
	maxKDFThreads   = 16
)

// Validate проверяет, что параметры в допустимых границах.
func (p KDFParams) Validate() error {
	if p.Time < minKDFTime || p.Time > maxKDFTime {
		return fmt.Errorf("%w: kdf time must be %d-%d", ErrInvalidArgument, minKDFTime, maxKDFTime)
	}
	if p.MemoryKiB < minKDFMemoryKiB || p.MemoryKiB > maxKDFMemoryKiB {
		return fmt.Errorf("%w: kdf memory must be %d-%d KiB", ErrInvalidArgument, minKDFMemoryKiB, maxKDFMemoryKiB)
	}
	if p.Threads < minKDFThreads || p.Threads > maxKDFThreads {
		return fmt.Errorf("%w: kdf threads must be %d-%d", ErrInvalidArgument, minKDFThreads, maxKDFThreads)
	}
	return nil
}
