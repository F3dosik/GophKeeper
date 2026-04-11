// Package repository содержит вспомогательные утилиты для работы с репозиториями,
// в том числе механизм повторных попыток при временных сбоях хранилища.
package repository

import (
	"context"
	"fmt"
	"time"
)

// WithRetry выполняет операцию op с экспоненциальными задержками между попытками.
// Повторная попытка выполняется только если IsRetriable вернула true для возникшей ошибки.
// Всего делается не более 4 попыток с задержками 100 мс, 300 мс и 700 мс между ними.
// Если контекст отменён до или во время ожидания — немедленно возвращается ctx.Err().
func WithRetry(ctx context.Context, IsRetriable func(error) bool, op func() error) error {
	delays := []time.Duration{
		100 * time.Millisecond,
		300 * time.Millisecond,
		700 * time.Millisecond,
	}
	maxAttempts := len(delays) + 1

	var err error
	for i := 0; i < maxAttempts; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err = op()
		if err == nil {
			return nil
		}

		if !IsRetriable(err) || i == len(delays) {
			return fmt.Errorf("operation failed after %d attempt(s): %w", i+1, err)
		}

		select {
		case <-time.After(delays[i]):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("operation failed after %d attempt(s): %w", maxAttempts, err)
}
