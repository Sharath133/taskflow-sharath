package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/taskflow/backend/internal/domain"
	"github.com/taskflow/backend/internal/repository"
)

// ProjectDetail is a project plus an aggregate task count.
type ProjectDetail struct {
	domain.Project
	TaskCount int64 `json:"task_count"`
}

// ProjectService owns project lifecycle and access rules.
type ProjectService interface {
	Create(ctx context.Context, userID uuid.UUID, req domain.CreateProjectRequest) (*domain.Project, error)
	GetByID(ctx context.Context, userID, projectID uuid.UUID) (*ProjectDetail, error)
	ListAccessible(ctx context.Context, userID uuid.UUID, page, limit *int) (*ProjectListResult, error)
	GetStats(ctx context.Context, userID, projectID uuid.UUID) (*domain.ProjectTaskStats, error)
	Update(ctx context.Context, userID, projectID uuid.UUID, req domain.UpdateProjectRequest) (*domain.Project, error)
	Delete(ctx context.Context, userID, projectID uuid.UUID) error
}

type projectService struct {
	db       *sqlx.DB
	projects repository.ProjectRepository
	tasks    repository.TaskRepository
}

// NewProjectService constructs a ProjectService with repository dependencies.
func NewProjectService(db *sqlx.DB, projects repository.ProjectRepository, tasks repository.TaskRepository) ProjectService {
	return &projectService{db: db, projects: projects, tasks: tasks}
}

// Create validates input and creates a project owned by userID.
func (s *projectService) Create(ctx context.Context, userID uuid.UUID, req domain.CreateProjectRequest) (*domain.Project, error) {
	if userID == uuid.Nil {
		return nil, domain.NewValidationError(domain.FieldMessage("user_id", "is required"))
	}
	if err := validateStruct(&req); err != nil {
		return nil, err
	}

	p := &domain.Project{
		Name:        strings.TrimSpace(req.Name),
		Description: req.Description,
		OwnerID:     userID,
	}
	if err := s.projects.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	return p, nil
}

// GetByID returns a project if the user may access it, including task count.
func (s *projectService) GetByID(ctx context.Context, userID, projectID uuid.UUID) (*ProjectDetail, error) {
	if projectID == uuid.Nil {
		return nil, domain.NewValidationError(domain.FieldMessage("project_id", "is required"))
	}

	p, err := assertProjectAccessible(ctx, s.projects, s.tasks, userID, projectID)
	if err != nil {
		return nil, err
	}

	n, err := s.tasks.CountByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("task count: %w", err)
	}

	return &ProjectDetail{Project: *p, TaskCount: n}, nil
}

// ListAccessible returns projects the user owns or has at least one task in as assignee or creator.
// When both page and limit are nil, returns all projects (no pagination metadata).
// When either is set, applies pagination with defaults page=1, limit=20 (max 100).
func (s *projectService) ListAccessible(ctx context.Context, userID uuid.UUID, page, limit *int) (*ProjectListResult, error) {
	if userID == uuid.Nil {
		return nil, domain.NewValidationError(domain.FieldMessage("user_id", "is required"))
	}
	if page == nil && limit == nil {
		list, err := s.projects.FindAccessibleByUserID(ctx, userID, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("list projects: %w", err)
		}
		return &ProjectListResult{Items: list, Paginated: false}, nil
	}
	p, l := normalizePagination(page, limit)
	off := (p - 1) * l
	total, err := s.projects.CountAccessibleByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("count projects: %w", err)
	}
	list, err := s.projects.FindAccessibleByUserID(ctx, userID, &l, &off)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return &ProjectListResult{Items: list, Total: total, Page: p, Limit: l, Paginated: true}, nil
}

// GetStats returns task counts by status and assignee for a project the user can access.
func (s *projectService) GetStats(ctx context.Context, userID, projectID uuid.UUID) (*domain.ProjectTaskStats, error) {
	if projectID == uuid.Nil {
		return nil, domain.NewValidationError(domain.FieldMessage("project_id", "is required"))
	}
	if _, err := assertProjectAccessible(ctx, s.projects, s.tasks, userID, projectID); err != nil {
		return nil, err
	}
	stats, err := s.tasks.StatsForProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project stats: %w", err)
	}
	return stats, nil
}

// Update applies partial changes when the user owns the project.
func (s *projectService) Update(ctx context.Context, userID, projectID uuid.UUID, req domain.UpdateProjectRequest) (*domain.Project, error) {
	if projectID == uuid.Nil {
		return nil, domain.NewValidationError(domain.FieldMessage("project_id", "is required"))
	}
	if err := validateStruct(&req); err != nil {
		return nil, err
	}
	if req.Name == nil && req.Description == nil {
		return nil, domain.NewValidationError(domain.FieldMessage("body", "at least one field must be provided"))
	}

	p, err := loadProject(ctx, s.projects, projectID)
	if err != nil {
		return nil, err
	}
	if err := assertProjectOwner(p, userID); err != nil {
		return nil, err
	}

	var updated *domain.Project
	err = repository.WithTx(ctx, s.db, func(rx *repository.Repositories) error {
		if err := rx.Projects.Update(ctx, projectID, req); err != nil {
			return err
		}
		p, err := rx.Projects.FindByID(ctx, projectID)
		if err != nil {
			return err
		}
		updated = p
		return nil
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		if strings.Contains(err.Error(), "no fields to update") {
			return nil, domain.NewValidationError(domain.FieldMessage("body", "at least one field must be provided"))
		}
		return nil, fmt.Errorf("update project: %w", err)
	}
	return updated, nil
}

// Delete removes a project when the user owns it.
func (s *projectService) Delete(ctx context.Context, userID, projectID uuid.UUID) error {
	if projectID == uuid.Nil {
		return domain.NewValidationError(domain.FieldMessage("project_id", "is required"))
	}

	p, err := loadProject(ctx, s.projects, projectID)
	if err != nil {
		return err
	}
	if err := assertProjectOwner(p, userID); err != nil {
		return err
	}

	if err := s.projects.Delete(ctx, projectID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("delete project: %w", err)
	}
	return nil
}
