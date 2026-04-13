//go:build integration

package tests

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/taskflow/backend/internal/auth"
)

func TestRegister_Success(t *testing.T) {
	cleanDatabase(t)

	email := fmt.Sprintf("user_%s@test.local", uuid.NewString())
	u := createTestUser(t, "Alice", email, "password123")

	require.NotEqual(t, uuid.Nil, u.ID)
	require.NotEmpty(t, u.Token)
	validateAccessToken(t, u.Token, u.ID, email)

	var hash string
	err := testDB.Get(&hash, `SELECT password FROM users WHERE id = $1`, u.ID)
	require.NoError(t, err)
	assert.True(t,
		len(hash) > 4 && (hash[:4] == "$2a$" || hash[:4] == "$2b$" || hash[:4] == "$2y$"),
		"password should be stored as bcrypt hash, got prefix: %q", safePrefix(hash, 8))
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte("password123")))
}

func TestRegister_DuplicateEmail(t *testing.T) {
	cleanDatabase(t)

	email := fmt.Sprintf("dup_%s@test.local", uuid.NewString())
	_ = createTestUser(t, "First", email, "password123")

	body := map[string]string{
		"name":     "Second",
		"email":    email,
		"password": "password4567",
	}
	w := makeRequest(t, httpMethodPost, "/auth/register", toJSONBody(t, body), "")
	payload := assertErrorPayload(t, w, httpStatusConflict)
	require.Equal(t, "email already registered", payload["error"])
}

func TestLogin_Success(t *testing.T) {
	cleanDatabase(t)

	// Seeded user: test@example.com / password123
	token := getAuthToken(t, "test@example.com", "password123")
	require.NotEmpty(t, token)

	claims, err := auth.ValidateToken(token, testJWTSecret)
	require.NoError(t, err)
	uid, err := claims.ParsedUserID()
	require.NoError(t, err)
	require.Equal(t, "10000000-0000-4000-8000-000000000001", uid.String())
}

func TestLogin_InvalidCredentials(t *testing.T) {
	cleanDatabase(t)

	body := map[string]string{
		"email":    "test@example.com",
		"password": "wrong-password-xyz",
	}
	w := makeRequest(t, httpMethodPost, "/auth/login", toJSONBody(t, body), "")
	payload := assertErrorPayload(t, w, httpStatusUnauthorized)
	require.Equal(t, "invalid credentials", payload["error"])
}

func TestLogin_NonexistentUser(t *testing.T) {
	cleanDatabase(t)

	body := map[string]string{
		"email":    fmt.Sprintf("nobody_%s@test.local", uuid.NewString()),
		"password": "password123",
	}
	w := makeRequest(t, httpMethodPost, "/auth/login", toJSONBody(t, body), "")
	payload := assertErrorPayload(t, w, httpStatusUnauthorized)
	require.Equal(t, "invalid credentials", payload["error"])
}

func safePrefix(s string, n int) string {
	if len(s) < n {
		return s
	}
	return s[:n]
}
