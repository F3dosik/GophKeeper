package domain

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidCardNumber = errors.New("некорректный номер карты")
	ErrInvalidCardExpiry = errors.New("некорректный срок действия (формат MM/YY)")
	ErrInvalidCardCVV    = errors.New("некорректный CVV")
	ErrInvalidCardHolder = errors.New("имя держателя не может быть пустым")
)

var (
	expiryRegex = regexp.MustCompile(`^(0[1-9]|1[0-2])/\d{2}$`)
	cvvRegex    = regexp.MustCompile(`^\d{3,4}$`)
)

// Validate проверяет корректность данных банковской карты.
func (c *CardSecret) Validate() error {
	num := strings.ReplaceAll(c.Number, " ", "")
	if len(num) < 13 || len(num) > 19 || !luhnValid(num) {
		return ErrInvalidCardNumber
	}
	if !expiryRegex.MatchString(c.Expiry) {
		return ErrInvalidCardExpiry
	}
	if !cvvRegex.MatchString(c.CVV) {
		return ErrInvalidCardCVV
	}
	if strings.TrimSpace(c.Holder) == "" {
		return ErrInvalidCardHolder
	}
	return nil
}

// luhnValid проверяет номер карты алгоритмом Луна.
func luhnValid(number string) bool {
	n := len(number)
	if n == 0 {
		return false
	}
	var sum int
	parity := n % 2
	for i := 0; i < n; i++ {
		if number[i] < '0' || number[i] > '9' {
			return false
		}
		digit := int(number[i] - '0')
		if i%2 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}
	return sum%10 == 0
}
