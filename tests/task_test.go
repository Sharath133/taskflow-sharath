//go:build integration

package tests

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCreateTask_Success(t *testing.T) {
	cleanDatabase(t)

	owner := createTestUser(t, "Owner", fmt.Sprintf("owner_%s@test.local", uuid.NewString()), "password123")
	pid := createProjectID(t, owner.Token, "Task Project")

	body := map[string]any{
		"title":    "Write integration tests",
		"priority": "high",
		"status":   "todo",
	}
	w := makeRequest(t, httpMethodPost, "/projects/"+pid+"/tasks", toJSONBody(t, body), owner.Token)
	payload := assertJSON(t, w, httpStatusCreated)
	data := dataMap(t, payload)

	require.Equal(t, "Write integration tests", data["title"])
	require.Equal(t, "todo", data["status"])
	require.Equal(t, "high", data["priority"])
	require.Equal(t, pid, data["project_id"])
}

func TestCreateTask_InvalidStatus(t *testing.T) {
	cleanDatabase(t)

	owner := createTestUser(t, "Owner", fmt.Sprintf("owner_%s@test.local", uuid.NewString()), "password123")
	pid := createProjectID(t, owner.Token, "Invalid Status Project")

	body := map[string]any{
		"title":    "Second task",
		"description": "Optional",
		"status":   "h",
		"priority": "medium",
	}
	w := makeRequest(t, httpMethodPost, "/projects/"+pid+"/tasks", toJSONBody(t, body), owner.Token)
	payload := assertErrorPayload(t, w, httpStatusBadRequest)
	require.Equal(t, "validation failed", payload["error"])
	fields, ok := payload["fields"].(map[string]any)
	require.True(t, ok)
	st, ok := fields["status"].(string)
	require.True(t, ok)
	require.Contains(t, st, "todo")
	require.Contains(t, st, "in_progress")
	require.Contains(t, st, "done")
}

func TestUpdateTask_Status(t *testing.T) {
	cleanDatabase(t)

	owner := createTestUser(t, "Owner", fmt.Sprintf("owner_%s@test.local", uuid.NewString()), "password123")
	pid := createProjectID(t, owner.Token, "Update Task Project")

	createBody := map[string]any{"title": "Mutable task", "priority": "low"}
	wc := makeRequest(t, httpMethodPost, "/projects/"+pid+"/tasks", toJSONBody(t, createBody), owner.Token)
	created := assertJSON(t, wc, httpStatusCreated)
	tid := asString(t, dataMap(t, created)["id"])

	patch := map[string]any{"status": "done"}
	w := makeRequest(t, httpMethodPatch, "/tasks/"+tid, toJSONBody(t, patch), owner.Token)
	payload := assertJSON(t, w, httpStatusOK)
	data := dataMap(t, payload)
	require.Equal(t, "done", data["status"])
}

func TestFilterTasks_ByStatus(t *testing.T) {
	cleanDatabase(t)

	owner := createTestUser(t, "Owner", fmt.Sprintf("owner_%s@test.local", uuid.NewString()), "password123")
	pid := createProjectID(t, owner.Token, "Filter Status Project")

	mustCreateTask(t, owner.Token, pid, "A", "todo")
	mustCreateTask(t, owner.Token, pid, "B", "in_progress")
	mustCreateTask(t, owner.Token, pid, "C", "done")

	q := url.Values{}
	q.Set("status", "todo")
	w := makeRequest(t, httpMethodGet, "/projects/"+pid+"/tasks?"+q.Encode(), nil, owner.Token)
	payload := assertJSON(t, w, httpStatusOK)
	items := dataSlice(t, payload)
	require.Len(t, items, 1)
	m := items[0].(map[string]any)
	require.Equal(t, "A", m["title"])
	require.Equal(t, "todo", m["status"])
}

func TestFilterTasks_ByAssignee(t *testing.T) {
	cleanDatabase(t)

	owner := createTestUser(t, "Owner", fmt.Sprintf("owner_%s@test.local", uuid.NewString()), "password123")
	member := createTestUser(t, "Member", fmt.Sprintf("member_%s@test.local", uuid.NewString()), "password123")
	pid := createProjectID(t, owner.Token, "Filter Assignee Project")

	unassigned := map[string]any{"title": "Unassigned", "priority": "low"}
	wu := makeRequest(t, httpMethodPost, "/projects/"+pid+"/tasks", toJSONBody(t, unassigned), owner.Token)
	assertJSON(t, wu, httpStatusCreated)

	assigned := map[string]any{
		"title":       "For member",
		"priority":    "medium",
		"assignee_id": member.ID.String(),
	}
	wa := makeRequest(t, httpMethodPost, "/projects/"+pid+"/tasks", toJSONBody(t, assigned), owner.Token)
	assertJSON(t, wa, httpStatusCreated)

	q := url.Values{}
	q.Set("assignee", member.ID.String())
	w := makeRequest(t, httpMethodGet, "/projects/"+pid+"/tasks?"+q.Encode(), nil, owner.Token)
	payload := assertJSON(t, w, httpStatusOK)
	items := dataSlice(t, payload)
	require.Len(t, items, 1)
	m := items[0].(map[string]any)
	require.Equal(t, "For member", m["title"])
	require.Equal(t, member.ID.String(), m["assignee_id"])
}

func TestDeleteTask_AsProjectOwner(t *testing.T) {
	cleanDatabase(t)

	owner := createTestUser(t, "Owner", fmt.Sprintf("owner_%s@test.local", uuid.NewString()), "password123")
	pid := createProjectID(t, owner.Token, "Delete Owner Project")

	createBody := map[string]any{"title": "Deletable", "priority": "low"}
	wc := makeRequest(t, httpMethodPost, "/projects/"+pid+"/tasks", toJSONBody(t, createBody), owner.Token)
	created := assertJSON(t, wc, httpStatusCreated)
	tid := asString(t, dataMap(t, created)["id"])

	wd := makeRequest(t, httpMethodDelete, "/tasks/"+tid, nil, owner.Token)
	delPayload := assertJSON(t, wd, httpStatusOK)
	delData, ok := delPayload["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, tid, delData["id"])
	require.Contains(t, asString(t, delData["message"]), tid)
	require.Contains(t, asString(t, delData["message"]), "deleted")

	var cnt int
	err := testDB.Get(&cnt, `SELECT COUNT(*) FROM tasks WHERE id = $1`, tid)
	require.NoError(t, err)
	require.Equal(t, 0, cnt)
}

func TestDeleteTask_AsNonOwner(t *testing.T) {
	cleanDatabase(t)

	owner := createTestUser(t, "Owner", fmt.Sprintf("owner_%s@test.local", uuid.NewString()), "password123")
	member := createTestUser(t, "Member", fmt.Sprintf("member_%s@test.local", uuid.NewString()), "password123")
	pid := createProjectID(t, owner.Token, "Delete NonOwner Project")

	createBody := map[string]any{
		"title":       "Member assigned",
		"priority":    "low",
		"assignee_id": member.ID.String(),
	}
	wc := makeRequest(t, httpMethodPost, "/projects/"+pid+"/tasks", toJSONBody(t, createBody), owner.Token)
	created := assertJSON(t, wc, httpStatusCreated)
	tid := asString(t, dataMap(t, created)["id"])

	wd := makeRequest(t, httpMethodDelete, "/tasks/"+tid, nil, member.Token)
	payload := assertErrorPayload(t, wd, httpStatusForbidden)
	require.Equal(t, "forbidden", payload["error"])

	var cnt int
	err := testDB.Get(&cnt, `SELECT COUNT(*) FROM tasks WHERE id = $1`, tid)
	require.NoError(t, err)
	require.Equal(t, 1, cnt)
}

func createProjectID(t *testing.T, token, name string) string {
	t.Helper()
	reqBody := map[string]any{"name": name}
	w := makeRequest(t, httpMethodPost, "/projects", toJSONBody(t, reqBody), token)
	payload := assertJSON(t, w, httpStatusCreated)
	return asString(t, dataMap(t, payload)["id"])
}

func mustCreateTask(t *testing.T, token, projectID, title, status string) {
	t.Helper()
	body := map[string]any{
		"title":    title,
		"priority": "low",
		"status":   status,
	}
	w := makeRequest(t, httpMethodPost, "/projects/"+projectID+"/tasks", toJSONBody(t, body), token)
	assertJSON(t, w, httpStatusCreated)
}
