//go:build integration

package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/taskflow/backend/internal/auth"
)

func migrationsDirPath(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	dir, err := filepath.Abs(filepath.Join(wd, "..", "migrations"))
	require.NoError(t, err)
	return dir
}

// execSQLStatements runs a script split on semicolons (seed files only; not for functions with $$).
func execSQLStatements(db *sqlx.DB, script string) error {
	var sb strings.Builder
	for _, line := range strings.Split(script, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "--") {
			continue
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	for _, part := range strings.Split(sb.String(), ";") {
		s := strings.TrimSpace(part)
		if s == "" {
			continue
		}
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("exec: %w\n%s", err, s)
		}
	}
	return nil
}

// cleanDatabase truncates all application tables and reapplies seed data from 000002.
func cleanDatabase(t *testing.T) {
	t.Helper()
	_, err := testDB.Exec(`TRUNCATE TABLE tasks, projects, users RESTART IDENTITY CASCADE`)
	require.NoError(t, err)

	seedPath := filepath.Join(migrationsDirPath(t), "000002_seed_data.up.sql")
	b, err := os.ReadFile(seedPath)
	require.NoError(t, err)
	require.NoError(t, execSQLStatements(testDB, string(b)))
}

type testUser struct {
	ID       uuid.UUID
	Email    string
	Password string
	Token    string
	Name     string
}

// createTestUser registers a new user via the API and returns credentials plus JWT from the response.
func createTestUser(t *testing.T, name, email, password string) testUser {
	t.Helper()
	body := map[string]string{
		"name":     name,
		"email":    email,
		"password": password,
	}
	w := makeRequest(t, httpMethodPost, "/auth/register", toJSONBody(t, body), "")

	payload := assertJSON(t, w, httpStatusCreated)
	token, _ := payload["token"].(string)
	require.NotEmpty(t, token)

	userObj, ok := payload["user"].(map[string]any)
	require.True(t, ok, "expected user object")
	idStr, _ := userObj["id"].(string)
	uid, err := uuid.Parse(idStr)
	require.NoError(t, err)

	return testUser{
		ID:       uid,
		Email:    email,
		Password: password,
		Token:    token,
		Name:     name,
	}
}

// getAuthToken logs in and returns the JWT access token.
func getAuthToken(t *testing.T, email, password string) string {
	t.Helper()
	body := map[string]string{
		"email":    email,
		"password": password,
	}
	w := makeRequest(t, httpMethodPost, "/auth/login", toJSONBody(t, body), "")
	payload := assertJSON(t, w, httpStatusOK)
	token, _ := payload["token"].(string)
	require.NotEmpty(t, token)
	return token
}

// projectsSlice returns the `projects` array from a GET /projects response.
func projectsSlice(t *testing.T, payload map[string]any) []any {
	t.Helper()
	raw, ok := payload["projects"]
	require.True(t, ok, "missing projects key: %#v", payload)
	s, ok := raw.([]any)
	require.True(t, ok, "projects should be array, got %T", raw)
	return s
}

// tasksSlice returns the `tasks` array from a GET /projects/:id/tasks response.
func tasksSlice(t *testing.T, payload map[string]any) []any {
	t.Helper()
	raw, ok := payload["tasks"]
	require.True(t, ok, "missing tasks key: %#v", payload)
	s, ok := raw.([]any)
	require.True(t, ok, "tasks should be array, got %T", raw)
	return s
}

const (
	httpMethodGet    = "GET"
	httpMethodPost   = "POST"
	httpMethodPatch  = "PATCH"
	httpMethodDelete = "DELETE"

	httpStatusOK           = 200
	httpStatusCreated      = 201
	httpStatusNoContent    = 204
	httpStatusBadRequest   = 400
	httpStatusUnauthorized = 401
	httpStatusForbidden    = 403
	httpStatusNotFound     = 404
	httpStatusConflict     = 409
)

// makeRequest performs a request against the Gin test engine with optional Bearer JWT.
func makeRequest(t *testing.T, method, path string, body io.Reader, bearerToken string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	testEngine.ServeHTTP(w, req)
	return w
}

func toJSONBody(t *testing.T, v any) io.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewReader(b)
}

// assertJSON unmarshals the response body as a generic JSON object and checks status.
func assertJSON(t *testing.T, w *httptest.ResponseRecorder, wantStatus int) map[string]any {
	t.Helper()
	require.Equal(t, wantStatus, w.Code, "body: %s", w.Body.String())
	var m map[string]any
	if w.Body.Len() == 0 {
		return m
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &m))
	return m
}

// assertErrorPayload checks error responses: {"error": "..."} optionally with fields.
func assertErrorPayload(t *testing.T, w *httptest.ResponseRecorder, wantStatus int) map[string]any {
	t.Helper()
	return assertJSON(t, w, wantStatus)
}

func validateAccessToken(t *testing.T, token string, wantUserID uuid.UUID, wantEmail string) {
	t.Helper()
	claims, err := auth.ValidateToken(token, testJWTSecret)
	require.NoError(t, err)
	id, err := claims.ParsedUserID()
	require.NoError(t, err)
	require.Equal(t, wantUserID, id)
	require.Equal(t, strings.ToLower(strings.TrimSpace(wantEmail)), claims.Email)
}
