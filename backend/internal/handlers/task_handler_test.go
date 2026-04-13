package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/taskflow/backend/internal/domain"
	"github.com/taskflow/backend/internal/middleware"
	"github.com/taskflow/backend/internal/service"
)

type stubTaskService struct {
	listFn   func(context.Context, uuid.UUID, uuid.UUID, domain.TaskFilters) (*service.TaskListResult, error)
	deleteFn func(context.Context, uuid.UUID, uuid.UUID) error
}

func (s *stubTaskService) Create(context.Context, uuid.UUID, domain.CreateTaskRequest) (*domain.Task, error) {
	return nil, errors.New("unimplemented")
}

func (s *stubTaskService) Update(context.Context, uuid.UUID, uuid.UUID, domain.UpdateTaskRequest) (*domain.Task, error) {
	return nil, errors.New("unimplemented")
}

func (s *stubTaskService) Delete(ctx context.Context, userID, taskID uuid.UUID) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, userID, taskID)
	}
	return errors.New("unimplemented")
}

func (s *stubTaskService) ListByProject(ctx context.Context, userID, projectID uuid.UUID, filters domain.TaskFilters) (*service.TaskListResult, error) {
	if s.listFn != nil {
		return s.listFn(ctx, userID, projectID, filters)
	}
	return nil, errors.New("list not stubbed")
}

// TestListByProject_PaginatedResponse verifies ?page=&limit= yields tasks/total/page/limit at the top level.
func TestListByProject_PaginatedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pid := uuid.MustParse("20000000-0000-4000-8000-000000000001")
	uid := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	taskID := uuid.MustParse("30000000-0000-4000-8000-000000000001")

	svc := &stubTaskService{
		listFn: func(ctx context.Context, userID, projectID uuid.UUID, filters domain.TaskFilters) (*service.TaskListResult, error) {
			if userID != uid {
				t.Fatalf("userID mismatch")
			}
			if projectID != pid {
				t.Fatalf("projectID mismatch")
			}
			if filters.Page == nil || *filters.Page != 1 || filters.Limit == nil || *filters.Limit != 10 {
				t.Fatalf("expected page=1 limit=10, got page=%v limit=%v", filters.Page, filters.Limit)
			}
			return &service.TaskListResult{
				Items: []domain.Task{
					{ID: taskID, Title: "T1", Status: domain.TaskStatusTodo, Priority: domain.TaskPriorityLow, ProjectID: pid},
				},
				Total:     42,
				Page:      1,
				Limit:     10,
				Paginated: true,
			}, nil
		},
	}

	h := NewTaskHandler(svc)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: pid.String()}}
	c.Request = httptest.NewRequest(http.MethodGet, "/projects/"+pid.String()+"/tasks?page=1&limit=10", nil)
	claims := &domain.Claims{UserID: uid.String(), Email: "u@example.com"}
	c.Set(middleware.ContextJWTClaims, claims)

	h.ListByProject(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Tasks []domain.Task `json:"tasks"`
		Total int64         `json:"total"`
		Page  int           `json:"page"`
		Limit int           `json:"limit"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Total != 42 || body.Page != 1 || body.Limit != 10 {
		t.Fatalf("unexpected pagination meta: %+v", body)
	}
	if len(body.Tasks) != 1 || body.Tasks[0].Title != "T1" {
		t.Fatalf("unexpected tasks: %+v", body.Tasks)
	}
}

func TestCreate_InvalidStatusReturnsFieldError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uid := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	pid := uuid.MustParse("20000000-0000-4000-8000-000000000002")

	h := NewTaskHandler(&stubTaskService{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: pid.String()}}
	c.Request = httptest.NewRequest(http.MethodPost, "/projects/"+pid.String()+"/tasks", strings.NewReader(
		`{"title":"Second task","description":"Optional","status":"h","priority":"medium"}`,
	))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middleware.ContextJWTClaims, &domain.Claims{UserID: uid.String(), Email: "u@example.com"})

	h.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Error  string            `json:"error"`
		Fields map[string]string `json:"fields"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error != domain.ErrValidation.Error() {
		t.Fatalf("error: %q", resp.Error)
	}
	msg, ok := resp.Fields["status"]
	if !ok || !strings.Contains(msg, "todo") || !strings.Contains(msg, "in_progress") {
		t.Fatalf("fields.status: %q", msg)
	}
}

func TestDelete_SuccessNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uid := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	tid := uuid.MustParse("30000000-0000-4000-8000-000000000003")

	svc := &stubTaskService{
		deleteFn: func(context.Context, uuid.UUID, uuid.UUID) error { return nil },
	}
	h := NewTaskHandler(svc)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: tid.String()}}
	c.Request = httptest.NewRequest(http.MethodDelete, "/tasks/"+tid.String(), nil)
	c.Set(middleware.ContextJWTClaims, &domain.Claims{UserID: uid.String(), Email: "u@example.com"})

	h.Delete(c)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", w.Code, w.Body.String())
	}
	if w.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", w.Body.String())
	}
}

var _ service.TaskService = (*stubTaskService)(nil)
