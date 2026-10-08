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
	// TokenVersion — версия токенов пользователя на момент выдачи; после «выхода
	// на всех устройствах» версия пользователя растёт и старые токены перестают действовать.
	TokenVersion int `json:"tv"`
	// TokenID — уникальный идентификатор токена (jti), по нему отзывается отдельный токен.
	TokenID uuid.UUID `json:"-"`
	jwt.RegisteredClaims
}

// GenerateToken генерирует подписанный JWT токен для пользователя.
// Токен содержит userID, версию токенов и случайный jti и истекает через ttl.
// Возвращает подписанную строку токена или ошибку если подпись не удалась.
func GenerateToken(userID uuid.UUID, tokenVersion int, secretKey string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:       userID,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

// ParseToken валидирует JWT токен и извлекает Claims.
// Возвращает ошибку если токен невалиден, истёк, не содержит срока действия или jti,
// либо подписан другим методом.
func ParseToken(tokenString, secretKey string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secretKey), nil
	}, jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	jti, err := uuid.Parse(claims.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid token: bad jti: %w", err)
	}
	claims.TokenID = jti
	return claims, nil
}
