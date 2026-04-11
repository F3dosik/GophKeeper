// Package jwtutil предоставляет утилиты для генерации и валидации JWT токенов.
package jwtutil

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims представляет данные хранящиеся внутри JWT токена
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateToken генерирует подписанный JWT токен для пользователя.
// Токен содержит userID и истекает через 24 часа.
// Возвращает подписанную строку токена или ошибку если подпись не удалась.
func GenerateToken(userID uuid.UUID, secretKey string) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secretKey))
}

// ParseToken валидирует JWT токен и извлекает Claims.
// Возвращает ошибку если токен невалиден, истёк или подписан другим методом.
func ParseToken(tokenString, secretKey string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secretKey), nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	return claims, nil
}
