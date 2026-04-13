package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/taskflow/backend/internal/domain"
	"github.com/taskflow/backend/internal/middleware"
	"github.com/taskflow/backend/internal/service"
)

// TaskHandler exposes task routes (JWT-protected).
type TaskHandler struct {
	svc service.TaskService
}

// NewTaskHandler constructs a TaskHandler.
func NewTaskHandler(svc service.TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}

// ListByProject handles GET /projects/:id/tasks with optional ?status=, ?assignee=, ?page=, ?limit=.
func (h *TaskHandler) ListByProject(c *gin.Context) {
	userID := middleware.GetUserIDFromContext(c)
	if userID == uuid.Nil {
		errorResponse(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	projectID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var q struct {
		Status   string `form:"status"`
		Assignee string `form:"assignee"`
	}
	if err := c.ShouldBindQuery(&q); err != nil {
		errorResponse(c, http.StatusBadRequest, domain.ErrValidation.Error(), map[string]string{
			"query": "invalid query parameters",
		})
		return
	}

	page, limit, bad := ParseOptionalPagination(c)
	if bad {
		return
	}

	filters := domain.TaskFilters{Page: page, Limit: limit}
	if q.Status != "" {
		st, err := domain.ParseTaskStatus(q.Status)
		if err != nil {
			validationErrorResponse(c, domain.NewValidationError(domain.FieldMessage("status", domain.AllowedTaskStatusValuesMessage())))
			return
		}
		filters.Status = &st
	}
	if q.Assignee != "" {
		aid, err := uuid.Parse(q.Assignee)
		if err != nil {
			errorResponse(c, http.StatusBadRequest, domain.ErrValidation.Error(), map[string]string{
				"assignee": "must be a valid UUID",
			})
			return
		}
		filters.AssigneeID = &aid
	}

	out, err := h.svc.ListByProject(c.Request.Context(), userID, projectID, filters)
	if err != nil {
		if HandleServiceError(c, err) {
			return
		}
		slog.Error("list tasks failed", "error", err)
		errorResponse(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	if out.Paginated {
		writeJSON(c, http.StatusOK, gin.H{
			"tasks": out.Items,
			"total": out.Total,
			"page":  out.Page,
			"limit": out.Limit,
		})
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"tasks": out.Items})
}

// Create handles POST /projects/:id/tasks.
func (h *TaskHandler) Create(c *gin.Context) {
	userID := middleware.GetUserIDFromContext(c)
	if userID == uuid.Nil {
		errorResponse(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	projectID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var req domain.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}
	req.ProjectID = projectID
	if !ValidateStruct(c, &req) {
		return
	}

	t, err := h.svc.Create(c.Request.Context(), userID, req)
	if err != nil {
		if HandleServiceError(c, err) {
			return
		}
		slog.Error("create task failed", "error", err)
		errorResponse(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	writeJSON(c, http.StatusCreated, t)
}

// Update handles PATCH /tasks/:id.
func (h *TaskHandler) Update(c *gin.Context) {
	userID := middleware.GetUserIDFromContext(c)
	if userID == uuid.Nil {
		errorResponse(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	taskID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var req domain.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}
	if !ValidateStruct(c, &req) {
		return
	}

	t, err := h.svc.Update(c.Request.Context(), userID, taskID, req)
	if err != nil {
		if HandleServiceError(c, err) {
			return
		}
		slog.Error("update task failed", "error", err)
		errorResponse(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	writeJSON(c, http.StatusOK, t)
}

// Delete handles DELETE /tasks/:id (204 No Content on success).
func (h *TaskHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserIDFromContext(c)
	if userID == uuid.Nil {
		errorResponse(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	taskID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), userID, taskID); err != nil {
		if HandleServiceError(c, err) {
			return
		}
		slog.Error("delete task failed", "error", err)
		errorResponse(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	c.Status(http.StatusNoContent)
}
