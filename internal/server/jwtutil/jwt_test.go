package jwtutil_test

import (
	"testing"
	"time"

	"github.com/F3dosik/GophKeeper/internal/server/jwtutil"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret"

func TestGenerateToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		userID := uuid.New()

		token, err := jwtutil.GenerateToken(userID, 0, testSecret, time.Hour)

		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("different userIDs produce different tokens", func(t *testing.T) {
		t1, err := jwtutil.GenerateToken(uuid.New(), 0, testSecret, time.Hour)
		require.NoError(t, err)
		t2, err := jwtutil.GenerateToken(uuid.New(), 0, testSecret, time.Hour)
		require.NoError(t, err)

		assert.NotEqual(t, t1, t2)
	})
}

func TestParseToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		userID := uuid.New()
		token, err := jwtutil.GenerateToken(userID, 0, testSecret, time.Hour)
		require.NoError(t, err)

		claims, err := jwtutil.ParseToken(token, testSecret)

		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.NotEqual(t, uuid.Nil, claims.TokenID)
	})

	t.Run("carries token version", func(t *testing.T) {
		token, err := jwtutil.GenerateToken(uuid.New(), 7, testSecret, time.Hour)
		require.NoError(t, err)

		claims, err := jwtutil.ParseToken(token, testSecret)
		require.NoError(t, err)
		assert.Equal(t, 7, claims.TokenVersion)
	})

	t.Run("each token has unique jti", func(t *testing.T) {
		userID := uuid.New()
		t1, err := jwtutil.GenerateToken(userID, 0, testSecret, time.Hour)
		require.NoError(t, err)
		t2, err := jwtutil.GenerateToken(userID, 0, testSecret, time.Hour)
		require.NoError(t, err)

		c1, err := jwtutil.ParseToken(t1, testSecret)
		require.NoError(t, err)
		c2, err := jwtutil.ParseToken(t2, testSecret)
		require.NoError(t, err)
		assert.NotEqual(t, c1.TokenID, c2.TokenID)
	})

	t.Run("token without expiration is rejected", func(t *testing.T) {
		claims := jwtutil.Claims{
			UserID:           uuid.New(),
			RegisteredClaims: jwt.RegisteredClaims{ID: uuid.NewString()},
		}
		signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
		require.NoError(t, err)

		_, err = jwtutil.ParseToken(signed, testSecret)
		require.Error(t, err)
	})

	t.Run("token without jti is rejected", func(t *testing.T) {
		claims := jwtutil.Claims{
			UserID: uuid.New(),
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		}
		signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
		require.NoError(t, err)

		_, err = jwtutil.ParseToken(signed, testSecret)
		require.Error(t, err)
	})

	t.Run("wrong secret", func(t *testing.T) {
		token, err := jwtutil.GenerateToken(uuid.New(), 0, testSecret, time.Hour)
		require.NoError(t, err)

		_, err = jwtutil.ParseToken(token, "wrong-secret")

		require.Error(t, err)
	})

	t.Run("malformed token", func(t *testing.T) {
		_, err := jwtutil.ParseToken("not-a-jwt", testSecret)

		require.Error(t, err)
	})

	t.Run("expired token", func(t *testing.T) {
		claims := jwtutil.Claims{
			UserID: uuid.New(),
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			},
		}
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := tok.SignedString([]byte(testSecret))
		require.NoError(t, err)

		_, err = jwtutil.ParseToken(signed, testSecret)

		require.Error(t, err)
	})

	t.Run("unexpected signing method", func(t *testing.T) {
		claims := jwtutil.Claims{
			UserID: uuid.New(),
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		}
		tok := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
		signed, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)

		_, err = jwtutil.ParseToken(signed, testSecret)

		require.Error(t, err)
	})
}
