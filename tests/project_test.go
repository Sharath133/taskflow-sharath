//go:build integration

package tests

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCreateProject_Success(t *testing.T) {
	cleanDatabase(t)

	owner := createTestUser(t, "Owner", fmt.Sprintf("owner_%s@test.local", uuid.NewString()), "password123")

	reqBody := map[string]any{
		"name":        "Integration Project",
		"description": "created by test",
	}
	w := makeRequest(t, httpMethodPost, "/projects", toJSONBody(t, reqBody), owner.Token)
	payload := assertJSON(t, w, httpStatusCreated)
	data := dataMap(t, payload)

	require.Equal(t, "Integration Project", data["name"])
	require.Equal(t, owner.ID.String(), data["owner_id"])
	_, err := uuid.Parse(asString(t, data["id"]))
	require.NoError(t, err)
}

func TestListProjects_AsOwner(t *testing.T) {
	cleanDatabase(t)

	owner := createTestUser(t, "Owner", fmt.Sprintf("owner_%s@test.local", uuid.NewString()), "password123")

	createReq := map[string]any{"name": "Listed Project"}
	wc := makeRequest(t, httpMethodPost, "/projects", toJSONBody(t, createReq), owner.Token)
	created := assertJSON(t, wc, httpStatusCreated)
	pid := asString(t, dataMap(t, created)["id"])

	w := makeRequest(t, httpMethodGet, "/projects", nil, owner.Token)
	payload := assertJSON(t, w, httpStatusOK)
	items := dataSlice(t, payload)
	require.GreaterOrEqual(t, len(items), 1)

	found := false
	for _, it := range items {
		m, ok := it.(map[string]any)
		require.True(t, ok)
		if m["id"] == pid {
			found = true
			require.Equal(t, "Listed Project", m["name"])
		}
	}
	require.True(t, found, "created project should appear in owner's list")
}

func TestUpdateProject_AsOwner(t *testing.T) {
	cleanDatabase(t)

	owner := createTestUser(t, "Owner", fmt.Sprintf("owner_%s@test.local", uuid.NewString()), "password123")

	createReq := map[string]any{"name": "Old Name"}
	wc := makeRequest(t, httpMethodPost, "/projects", toJSONBody(t, createReq), owner.Token)
	created := assertJSON(t, wc, httpStatusCreated)
	pid := asString(t, dataMap(t, created)["id"])

	patch := map[string]any{"name": "New Name"}
	w := makeRequest(t, httpMethodPatch, "/projects/"+pid, toJSONBody(t, patch), owner.Token)
	payload := assertJSON(t, w, httpStatusOK)
	data := dataMap(t, payload)
	require.Equal(t, "New Name", data["name"])
}

func TestUpdateProject_AsNonOwner(t *testing.T) {
	cleanDatabase(t)

	owner := createTestUser(t, "Owner", fmt.Sprintf("owner_%s@test.local", uuid.NewString()), "password123")
	member := createTestUser(t, "Member", fmt.Sprintf("member_%s@test.local", uuid.NewString()), "password123")

	createReq := map[string]any{"name": "Shared Project"}
	wc := makeRequest(t, httpMethodPost, "/projects", toJSONBody(t, createReq), owner.Token)
	created := assertJSON(t, wc, httpStatusCreated)
	pid := asString(t, dataMap(t, created)["id"])

	taskBody := map[string]any{
		"title":        "Member task",
		"assignee_id":  member.ID.String(),
		"priority":     "low",
		"status":       "todo",
	}
	wt := makeRequest(t, httpMethodPost, "/projects/"+pid+"/tasks", toJSONBody(t, taskBody), owner.Token)
	assertJSON(t, wt, httpStatusCreated)

	patch := map[string]any{"name": "Hijack Attempt"}
	w := makeRequest(t, httpMethodPatch, "/projects/"+pid, toJSONBody(t, patch), member.Token)
	payload := assertErrorPayload(t, w, httpStatusForbidden)
	require.Equal(t, "forbidden", payload["error"])
}

func TestDeleteProject_CascadesTasks(t *testing.T) {
	cleanDatabase(t)

	owner := createTestUser(t, "Owner", fmt.Sprintf("owner_%s@test.local", uuid.NewString()), "password123")

	createReq := map[string]any{"name": "Cascade Project"}
	wc := makeRequest(t, httpMethodPost, "/projects", toJSONBody(t, createReq), owner.Token)
	created := assertJSON(t, wc, httpStatusCreated)
	pid := asString(t, dataMap(t, created)["id"])
	projectUUID, err := uuid.Parse(pid)
	require.NoError(t, err)

	for i := 0; i < 2; i++ {
		taskBody := map[string]any{
			"title":    fmt.Sprintf("Task %d", i),
			"priority": "medium",
		}
		wt := makeRequest(t, httpMethodPost, "/projects/"+pid+"/tasks", toJSONBody(t, taskBody), owner.Token)
		assertJSON(t, wt, httpStatusCreated)
	}

	var taskCount int
	err = testDB.Get(&taskCount, `SELECT COUNT(*) FROM tasks WHERE project_id = $1`, projectUUID)
	require.NoError(t, err)
	require.Equal(t, 2, taskCount)

	wd := makeRequest(t, httpMethodDelete, "/projects/"+pid, nil, owner.Token)
	require.Equal(t, httpStatusNoContent, wd.Code)

	err = testDB.Get(&taskCount, `SELECT COUNT(*) FROM tasks WHERE project_id = $1`, projectUUID)
	require.NoError(t, err)
	require.Equal(t, 0, taskCount)

	var projectCount int
	err = testDB.Get(&projectCount, `SELECT COUNT(*) FROM projects WHERE id = $1`, projectUUID)
	require.NoError(t, err)
	require.Equal(t, 0, projectCount)
}

func dataMap(t *testing.T, payload map[string]any) map[string]any {
	t.Helper()
	raw, ok := payload["data"]
	require.True(t, ok, "missing data key: %#v", payload)
	m, ok := raw.(map[string]any)
	require.True(t, ok, "data should be object, got %T", raw)
	return m
}

func dataSlice(t *testing.T, payload map[string]any) []any {
	t.Helper()
	raw, ok := payload["data"]
	require.True(t, ok, "missing data key: %#v", payload)
	s, ok := raw.([]any)
	require.True(t, ok, "data should be array, got %T", raw)
	return s
}

func asString(t *testing.T, v any) string {
	t.Helper()
	s, ok := v.(string)
	require.True(t, ok, "expected string, got %T", v)
	return s
}
