// Package postgres предоставляет реализации репозиториев домена на основе PostgreSQL.
// Все операции с базой данных выполняются через pgxpool и поддерживают
// автоматический повтор при временных сбоях соединения.
package postgres

import (
	"errors"
	"net"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// isRetriable проверяет, является ли ошибка повторяемой и стоит ли повторять операцию.
func isRetriable(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		// Проблемы с соединением
		case pgerrcode.ConnectionException,
			pgerrcode.ConnectionDoesNotExist,
			pgerrcode.ConnectionFailure,
			pgerrcode.CannotConnectNow,
			pgerrcode.AdminShutdown,
			pgerrcode.CrashShutdown,
			// Конкурентный доступ
			pgerrcode.SerializationFailure,
			pgerrcode.DeadlockDetected,
			// Временная перегрузка
			pgerrcode.TooManyConnections,
			pgerrcode.LockNotAvailable:
			return true
		}
	}
	// pgxpool — нет свободных соединений
	if errors.Is(err, pgx.ErrNoRows) {
		return false
	}
	// сеть упала
	if errors.Is(err, net.ErrClosed) {
		return true
	}
	return false
}

// isUniqueViolation проверяет, является ли ошибка нарушением уникальности в базе данных.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation
}

// isNoRows проверяет, является ли ошибка отсутствием строк в результате запроса.
func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// isForeignKeyViolation проверяет, является ли ошибка нарушением внешнего ключа.
func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgerrcode.ForeignKeyViolation
}
