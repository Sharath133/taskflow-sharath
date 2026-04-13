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

// ProjectHandler exposes project CRUD routes (JWT-protected).
type ProjectHandler struct {
	projects service.ProjectService
	tasks    service.TaskService
}

// NewProjectHandler constructs a ProjectHandler.
func NewProjectHandler(projects service.ProjectService, tasks service.TaskService) *ProjectHandler {
	return &ProjectHandler{projects: projects, tasks: tasks}
}

// List handles GET /projects. Optional ?page=&limit= (defaults page=1, limit=20, max 100).
func (h *ProjectHandler) List(c *gin.Context) {
	userID := middleware.GetUserIDFromContext(c)
	if userID == uuid.Nil {
		errorResponse(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	page, limit, bad := ParseOptionalPagination(c)
	if bad {
		return
	}

	out, err := h.projects.ListAccessible(c.Request.Context(), userID, page, limit)
	if err != nil {
		if HandleServiceError(c, err) {
			return
		}
		slog.Error("list projects failed", "error", err)
		errorResponse(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	if out.Paginated {
		successResponse(c, http.StatusOK, gin.H{
			"items": out.Items,
			"total": out.Total,
			"page":  out.Page,
			"limit": out.Limit,
		})
		return
	}
	successResponse(c, http.StatusOK, out.Items)
}

// Create handles POST /projects.
func (h *ProjectHandler) Create(c *gin.Context) {
	userID := middleware.GetUserIDFromContext(c)
	if userID == uuid.Nil {
		errorResponse(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var req domain.CreateProjectRequest
	if !BindJSONAndValidate(c, &req) {
		return
	}

	p, err := h.projects.Create(c.Request.Context(), userID, req)
	if err != nil {
		slog.Error("create project failed", "error", err)
		errorResponse(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	successResponse(c, http.StatusCreated, p)
}

// Get handles GET /projects/:id (project with tasks).
func (h *ProjectHandler) Get(c *gin.Context) {
	userID := middleware.GetUserIDFromContext(c)
	if userID == uuid.Nil {
		errorResponse(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	projectID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	detail, err := h.projects.GetByID(c.Request.Context(), userID, projectID)
	if err != nil {
		if HandleServiceError(c, err) {
			return
		}
		slog.Error("get project failed", "error", err)
		errorResponse(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	taskOut, err := h.tasks.ListByProject(c.Request.Context(), userID, projectID, domain.TaskFilters{})
	if err != nil {
		if HandleServiceError(c, err) {
			return
		}
		slog.Error("list project tasks failed", "error", err)
		errorResponse(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	successResponse(c, http.StatusOK, domain.ProjectWithTasks{
		Project:   detail.Project,
		TaskCount: detail.TaskCount,
		Tasks:     taskOut.Items,
	})
}

// Stats handles GET /projects/:id/stats (task counts by status and assignee).
func (h *ProjectHandler) Stats(c *gin.Context) {
	userID := middleware.GetUserIDFromContext(c)
	if userID == uuid.Nil {
		errorResponse(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	projectID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	stats, err := h.projects.GetStats(c.Request.Context(), userID, projectID)
	if err != nil {
		if HandleServiceError(c, err) {
			return
		}
		slog.Error("project stats failed", "error", err)
		errorResponse(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	successResponse(c, http.StatusOK, stats)
}

// Update handles PATCH /projects/:id (owner only).
func (h *ProjectHandler) Update(c *gin.Context) {
	userID := middleware.GetUserIDFromContext(c)
	if userID == uuid.Nil {
		errorResponse(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	projectID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var req domain.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)
		return
	}
	if !ValidateStruct(c, &req) {
		return
	}

	p, err := h.projects.Update(c.Request.Context(), userID, projectID, req)
	if err != nil {
		if HandleServiceError(c, err) {
			return
		}
		slog.Error("update project failed", "error", err)
		errorResponse(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	successResponse(c, http.StatusOK, p)
}

// Delete handles DELETE /projects/:id (owner only).
func (h *ProjectHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserIDFromContext(c)
	if userID == uuid.Nil {
		errorResponse(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	projectID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.projects.Delete(c.Request.Context(), userID, projectID); err != nil {
		if HandleServiceError(c, err) {
			return
		}
		slog.Error("delete project failed", "error", err)
		errorResponse(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	c.Status(http.StatusNoContent)
}
