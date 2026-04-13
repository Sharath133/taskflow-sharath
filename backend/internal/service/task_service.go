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

// TaskService owns task lifecycle within projects.
type TaskService interface {
	Create(ctx context.Context, userID uuid.UUID, req domain.CreateTaskRequest) (*domain.Task, error)
	Update(ctx context.Context, userID, taskID uuid.UUID, req domain.UpdateTaskRequest) (*domain.Task, error)
	Delete(ctx context.Context, userID, taskID uuid.UUID) error
	ListByProject(ctx context.Context, userID, projectID uuid.UUID, filters domain.TaskFilters) (*TaskListResult, error)
}

type taskService struct {
	db       *sqlx.DB
	tasks    repository.TaskRepository
	projects repository.ProjectRepository
	users    repository.UserRepository
}

// NewTaskService constructs a TaskService with repository dependencies.
func NewTaskService(db *sqlx.DB, tasks repository.TaskRepository, projects repository.ProjectRepository, users repository.UserRepository) TaskService {
	return &taskService{db: db, tasks: tasks, projects: projects, users: users}
}

// Create validates input, ensures the project exists and is accessible, and stores the task.
func (s *taskService) Create(ctx context.Context, userID uuid.UUID, req domain.CreateTaskRequest) (*domain.Task, error) {
	if userID == uuid.Nil {
		return nil, domain.NewValidationError(domain.FieldMessage("user_id", "is required"))
	}
	if err := validateStruct(&req); err != nil {
		return nil, err
	}
	if req.ProjectID == uuid.Nil {
		return nil, domain.NewValidationError(domain.FieldMessage("project_id", "is required"))
	}

	if _, err := assertProjectAccessible(ctx, s.projects, s.tasks, userID, req.ProjectID); err != nil {
		return nil, err
	}

	if err := s.validateAssignee(ctx, req.AssigneeID); err != nil {
		return nil, err
	}

	if err := s.validateTaskEnumsCreate(req); err != nil {
		return nil, err
	}

	t := &domain.Task{
		Title:       strings.TrimSpace(req.Title),
		Description: req.Description,
		Status:      req.Status,
		Priority:    req.Priority,
		ProjectID:   req.ProjectID,
		AssigneeID:  req.AssigneeID,
		DueDate:     req.DueDate,
		CreatedBy:   &userID,
	}
	if t.Status == "" {
		t.Status = domain.TaskStatusTodo
	}
	if t.Priority == "" {
		t.Priority = domain.TaskPriorityMedium
	}

	if err := s.tasks.Create(ctx, t); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	return t, nil
}

// Update applies partial task changes when the user may access the project.
func (s *taskService) Update(ctx context.Context, userID, taskID uuid.UUID, req domain.UpdateTaskRequest) (*domain.Task, error) {
	if taskID == uuid.Nil {
		return nil, domain.NewValidationError(domain.FieldMessage("task_id", "is required"))
	}
	if err := validateStruct(&req); err != nil {
		return nil, err
	}
	if !taskUpdateHasField(req) {
		return nil, domain.NewValidationError(domain.FieldMessage("body", "at least one field must be provided"))
	}

	existing, err := s.tasks.FindByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("load task: %w", err)
	}

	if _, err := assertProjectAccessible(ctx, s.projects, s.tasks, userID, existing.ProjectID); err != nil {
		return nil, err
	}

	if err := s.validateAssignee(ctx, req.AssigneeID); err != nil {
		return nil, err
	}
	if err := s.validateTaskEnumsUpdate(req); err != nil {
		return nil, err
	}

	var updated *domain.Task
	err = repository.WithTx(ctx, s.db, func(rx *repository.Repositories) error {
		if err := rx.Tasks.Update(ctx, taskID, req); err != nil {
			return err
		}
		t, err := rx.Tasks.FindByID(ctx, taskID)
		if err != nil {
			return err
		}
		updated = t
		return nil
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		if strings.Contains(err.Error(), "no fields to update") {
			return nil, domain.NewValidationError(domain.FieldMessage("body", "at least one field must be provided"))
		}
		return nil, fmt.Errorf("update task: %w", err)
	}
	return updated, nil
}

// Delete removes a task when the user is the project owner or the task creator.
func (s *taskService) Delete(ctx context.Context, userID, taskID uuid.UUID) error {
	if taskID == uuid.Nil {
		return domain.NewValidationError(domain.FieldMessage("task_id", "is required"))
	}

	t, err := s.tasks.FindByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("load task: %w", err)
	}

	p, err := loadProject(ctx, s.projects, t.ProjectID)
	if err != nil {
		return err
	}

	if err := assertTaskDeletable(p, t, userID); err != nil {
		return err
	}

	if err := s.tasks.Delete(ctx, taskID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("delete task: %w", err)
	}
	return nil
}

// ListByProject returns tasks in a project the user can access, applying optional filters.
func (s *taskService) ListByProject(ctx context.Context, userID, projectID uuid.UUID, filters domain.TaskFilters) (*TaskListResult, error) {
	if projectID == uuid.Nil {
		return nil, domain.NewValidationError(domain.FieldMessage("project_id", "is required"))
	}
	if filters.ProjectID != nil && *filters.ProjectID != projectID {
		return nil, domain.NewValidationError(domain.FieldMessage("project_id", "must match the project in the request path"))
	}

	if _, err := assertProjectAccessible(ctx, s.projects, s.tasks, userID, projectID); err != nil {
		return nil, err
	}

	if err := s.validateFilterEnums(filters); err != nil {
		return nil, err
	}

	if filters.Page == nil && filters.Limit == nil {
		list, err := s.tasks.FindByProjectID(ctx, projectID, filters.Status, filters.Priority, filters.AssigneeID, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("list tasks: %w", err)
		}
		return &TaskListResult{Items: list, Paginated: false}, nil
	}

	total, err := s.tasks.CountByProjectIDFiltered(ctx, projectID, filters.Status, filters.Priority, filters.AssigneeID)
	if err != nil {
		return nil, fmt.Errorf("count tasks: %w", err)
	}
	p, l := normalizePagination(filters.Page, filters.Limit)
	off := (p - 1) * l
	list, err := s.tasks.FindByProjectID(ctx, projectID, filters.Status, filters.Priority, filters.AssigneeID, &l, &off)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return &TaskListResult{Items: list, Total: total, Page: p, Limit: l, Paginated: true}, nil
}

func assertTaskDeletable(p *domain.Project, t *domain.Task, userID uuid.UUID) error {
	if p.OwnerID == userID {
		return nil
	}
	if t.CreatedBy != nil && *t.CreatedBy == userID {
		return nil
	}
	return fmt.Errorf("%w: only the project owner or task creator can delete this task", domain.ErrUnauthorized)
}

func (s *taskService) validateAssignee(ctx context.Context, assigneeID *uuid.UUID) error {
	if assigneeID == nil {
		return nil
	}
	if *assigneeID == uuid.Nil {
		return domain.NewValidationError(domain.FieldMessage("assignee_id", "must be a valid user id"))
	}
	_, err := s.users.FindByID(ctx, *assigneeID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewValidationError(domain.FieldMessage("assignee_id", "user not found"))
		}
		return fmt.Errorf("resolve assignee: %w", err)
	}
	return nil
}

func (s *taskService) validateTaskEnumsCreate(req domain.CreateTaskRequest) error {
	if req.Status != "" {
		if _, err := domain.ParseTaskStatus(string(req.Status)); err != nil {
			return domain.NewValidationError(domain.FieldMessage("status", err.Error()))
		}
	}
	if req.Priority != "" {
		if _, err := domain.ParseTaskPriority(string(req.Priority)); err != nil {
			return domain.NewValidationError(domain.FieldMessage("priority", err.Error()))
		}
	}
	return nil
}

func (s *taskService) validateTaskEnumsUpdate(req domain.UpdateTaskRequest) error {
	if req.Status != nil {
		if _, err := domain.ParseTaskStatus(string(*req.Status)); err != nil {
			return domain.NewValidationError(domain.FieldMessage("status", err.Error()))
		}
	}
	if req.Priority != nil {
		if _, err := domain.ParseTaskPriority(string(*req.Priority)); err != nil {
			return domain.NewValidationError(domain.FieldMessage("priority", err.Error()))
		}
	}
	return nil
}

func (s *taskService) validateFilterEnums(f domain.TaskFilters) error {
	if f.Status != nil {
		if _, err := domain.ParseTaskStatus(string(*f.Status)); err != nil {
			return domain.NewValidationError(domain.FieldMessage("status", err.Error()))
		}
	}
	if f.Priority != nil {
		if _, err := domain.ParseTaskPriority(string(*f.Priority)); err != nil {
			return domain.NewValidationError(domain.FieldMessage("priority", err.Error()))
		}
	}
	return nil
}

func taskUpdateHasField(req domain.UpdateTaskRequest) bool {
	return req.Title != nil || req.Description != nil || req.Status != nil ||
		req.Priority != nil || req.AssigneeID != nil || req.DueDate != nil
}
