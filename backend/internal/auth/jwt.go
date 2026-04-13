package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/taskflow/backend/internal/domain"
)

// Sentinel errors for token validation (use errors.Is); do not expose strings to clients.
var (
	ErrTokenMalformed = errors.New("token malformed")
	ErrTokenExpired   = errors.New("token expired")
	ErrTokenInvalid   = errors.New("token invalid")
)

// GenerateToken builds a signed HS256 JWT with exp, iat, user_id, and email.
//
// Example:
//
//	token, err := auth.GenerateToken(uid, "user@example.com", cfg.JWTSecret, 24*time.Hour)
//	if err != nil { ... }
func GenerateToken(userID uuid.UUID, email string, secret string, expiry time.Duration) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("%w: empty secret", ErrTokenInvalid)
	}
	if expiry <= 0 {
		return "", fmt.Errorf("%w: non-positive expiry", ErrTokenInvalid)
	}

	now := time.Now().UTC()
	claims := domain.Claims{
		UserID: userID.String(),
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)
	return t.SignedString([]byte(secret))
}

// ValidateToken parses and verifies an HS256 token, enforcing expiry and registered claims.
//
// Example:
//
//	claims, err := auth.ValidateToken(tokenString, cfg.JWTSecret)
//	if err != nil {
//	    if errors.Is(err, auth.ErrTokenExpired) { ... }
//	}
func ValidateToken(tokenString string, secret string) (*domain.Claims, error) {
	if tokenString == "" || secret == "" {
		return nil, ErrTokenMalformed
	}

	claims := &domain.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w", ErrTokenMalformed)
		}
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("%w", ErrTokenMalformed)
		}
		return []byte(secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("%w: %v", ErrTokenExpired, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}
	if token == nil || !token.Valid {
		return nil, ErrTokenInvalid
	}

	if _, err := claims.ParsedUserID(); err != nil {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}
