package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const minPasswordLength = 8

var (
	// ErrPasswordTooWeak is returned when plaintext fails strength rules before hashing.
	ErrPasswordTooWeak = errors.New("password does not meet minimum requirements")
)

// HashPassword validates strength (minimum 8 characters), then bcrypt-hashes the password.
// cost should be bcrypt.DefaultCost (10) or higher in production; avoid costs above ~15 without profiling.
//
// Example:
//
//	hashStr, err := auth.HashPassword(plain, bcrypt.DefaultCost)
//	if errors.Is(err, auth.ErrPasswordTooWeak) { ... }
func HashPassword(password string, cost int) (string, error) {
	if err := validatePasswordStrength(password); err != nil {
		return "", err
	}
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return "", fmt.Errorf("bcrypt cost must be between %d and %d", bcrypt.MinCost, bcrypt.MaxCost)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// ComparePassword checks plaintext against a bcrypt hash using constant-time comparison
// (via bcrypt.CompareHashAndPassword). Hashed password must be the ASCII-encoded bcrypt string.
//
// Example:
//
//	if err := auth.ComparePassword(user.PasswordHash, plain); err != nil {
//	    // invalid password — respond with the same error as "user not found" to avoid user enumeration
//	}
func ComparePassword(hashedPassword, password string) error {
	if hashedPassword == "" {
		return bcrypt.ErrMismatchedHashAndPassword
	}
	// bcrypt.CompareHashAndPassword runs in constant time with respect to the candidate password.
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func validatePasswordStrength(password string) error {
	if len(password) < minPasswordLength {
		return fmt.Errorf("%w: at least %d characters required", ErrPasswordTooWeak, minPasswordLength)
	}
	return nil
}
