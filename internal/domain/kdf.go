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

// WeakerThan сообщает, дешевле ли перебор пароля с параметрами p, чем с other:
// меньше проходов или меньше памяти. Число потоков на стоимость перебора почти не влияет.
func (p KDFParams) WeakerThan(other KDFParams) bool {
	return p.Time < other.Time || p.MemoryKiB < other.MemoryKiB
}

// String — параметры в виде «t=3, 64 МиБ».
func (p KDFParams) String() string {
	return fmt.Sprintf("t=%d, %d МиБ", p.Time, p.MemoryKiB/1024)
}

// KDFDowngradeError — сервер прислал параметры Argon2id слабее тех, что клиент уже видел
// у этой учётки. Ключ аутентификации с такими параметрами дешевле перебирать, поэтому
// подменённый сервер мог бы так добыть его для офлайн-перебора пароля. Клиент не выводит
// ключ, пока пользователь не подтвердит, что сам сменил параметры.
type KDFDowngradeError struct {
	// Known — параметры, запомненные на этом устройстве.
	Known KDFParams
	// Received — параметры, присланные сервером.
	Received KDFParams
}

func (e *KDFDowngradeError) Error() string {
	return fmt.Sprintf("server sent weaker kdf params than before: %s, known %s", e.Received, e.Known)
}

// Unwrap позволяет проверять ошибку через errors.Is(err, ErrKDFDowngrade).
func (e *KDFDowngradeError) Unwrap() error { return ErrKDFDowngrade }
