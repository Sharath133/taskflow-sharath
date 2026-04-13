package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/taskflow/backend/internal/domain"
)

// TaskRepository persists and loads tasks.
type TaskRepository interface {
	Create(ctx context.Context, t *domain.Task) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Task, error)
	FindByProjectID(ctx context.Context, projectID uuid.UUID, status *domain.TaskStatus, priority *domain.TaskPriority, assigneeID *uuid.UUID, limit, offset *int) ([]domain.Task, error)
	CountByProjectID(ctx context.Context, projectID uuid.UUID) (int64, error)
	CountByProjectIDFiltered(ctx context.Context, projectID uuid.UUID, status *domain.TaskStatus, priority *domain.TaskPriority, assigneeID *uuid.UUID) (int64, error)
	StatsForProject(ctx context.Context, projectID uuid.UUID) (*domain.ProjectTaskStats, error)
	InvolvedInProject(ctx context.Context, userID, projectID uuid.UUID) (bool, error)
	Update(ctx context.Context, id uuid.UUID, req domain.UpdateTaskRequest) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByAssigneeID(ctx context.Context, assigneeID uuid.UUID) ([]domain.Task, error)
}

type taskRepository struct {
	db sqlxConn
}

// NewTaskRepository returns a PostgreSQL TaskRepository.
func NewTaskRepository(db sqlxConn) TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) Create(ctx context.Context, t *domain.Task) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	if t.Status == "" {
		t.Status = domain.TaskStatusTodo
	}
	if t.Priority == "" {
		t.Priority = domain.TaskPriorityMedium
	}
	const q = `
		INSERT INTO tasks (id, title, description, status, priority, project_id, assignee_id, created_by, due_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at`
	if err := r.db.QueryRowContext(ctx, q,
		t.ID, t.Title, t.Description, t.Status, t.Priority, t.ProjectID, t.AssigneeID, t.CreatedBy, t.DueDate,
	).Scan(&t.CreatedAt, &t.UpdatedAt); err != nil {
		return fmt.Errorf("task create: %w", err)
	}
	return nil
}

func (r *taskRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	const q = `
		SELECT id, title, description, status, priority, project_id, assignee_id, created_by, due_date, created_at, updated_at
		FROM tasks
		WHERE id = $1`
	var t domain.Task
	if err := sqlx.GetContext(ctx, r.db, &t, q, id); err != nil {
		return nil, mapSingleRowErr(err, "task find by id")
	}
	return &t, nil
}

func (r *taskRepository) FindByProjectID(ctx context.Context, projectID uuid.UUID, status *domain.TaskStatus, priority *domain.TaskPriority, assigneeID *uuid.UUID, limit, offset *int) ([]domain.Task, error) {
	q := `
		SELECT id, title, description, status, priority, project_id, assignee_id, created_by, due_date, created_at, updated_at
		FROM tasks
		WHERE project_id = $1`
	args := []any{projectID}
	n := 2

	if status != nil {
		q += fmt.Sprintf(" AND status = $%d", n)
		args = append(args, string(*status))
		n++
	}
	if priority != nil {
		q += fmt.Sprintf(" AND priority = $%d", n)
		args = append(args, string(*priority))
		n++
	}
	if assigneeID != nil {
		q += fmt.Sprintf(" AND assignee_id = $%d", n)
		args = append(args, *assigneeID)
		n++
	}
	q += `
		ORDER BY created_at ASC`
	if limit != nil {
		off := 0
		if offset != nil {
			off = *offset
		}
		q += fmt.Sprintf(" LIMIT $%d OFFSET $%d", n, n+1)
		args = append(args, *limit, off)
	}

	var tasks []domain.Task
	if err := sqlx.SelectContext(ctx, r.db, &tasks, q, args...); err != nil {
		return nil, fmt.Errorf("task find by project: %w", err)
	}
	if tasks == nil {
		tasks = []domain.Task{}
	}
	return tasks, nil
}

func (r *taskRepository) CountByProjectID(ctx context.Context, projectID uuid.UUID) (int64, error) {
	const q = `SELECT COUNT(*) FROM tasks WHERE project_id = $1`
	var n int64
	if err := sqlx.GetContext(ctx, r.db, &n, q, projectID); err != nil {
		return 0, fmt.Errorf("task count by project: %w", err)
	}
	return n, nil
}

func (r *taskRepository) CountByProjectIDFiltered(ctx context.Context, projectID uuid.UUID, status *domain.TaskStatus, priority *domain.TaskPriority, assigneeID *uuid.UUID) (int64, error) {
	q := `SELECT COUNT(*) FROM tasks WHERE project_id = $1`
	args := []any{projectID}
	n := 2
	if status != nil {
		q += fmt.Sprintf(" AND status = $%d", n)
		args = append(args, string(*status))
		n++
	}
	if priority != nil {
		q += fmt.Sprintf(" AND priority = $%d", n)
		args = append(args, string(*priority))
		n++
	}
	if assigneeID != nil {
		q += fmt.Sprintf(" AND assignee_id = $%d", n)
		args = append(args, *assigneeID)
		n++
	}
	var count int64
	if err := sqlx.GetContext(ctx, r.db, &count, q, args...); err != nil {
		return 0, fmt.Errorf("task count by project filtered: %w", err)
	}
	return count, nil
}

func (r *taskRepository) StatsForProject(ctx context.Context, projectID uuid.UUID) (*domain.ProjectTaskStats, error) {
	const qStatus = `
		SELECT status, COUNT(*) AS cnt
		FROM tasks
		WHERE project_id = $1
		GROUP BY status`
	rows, err := r.db.QueryxContext(ctx, qStatus, projectID)
	if err != nil {
		return nil, fmt.Errorf("task stats by status: %w", err)
	}
	defer func() { _ = rows.Close() }()

	byStatus := make(map[string]int64)
	for rows.Next() {
		var st string
		var cnt int64
		if err := rows.Scan(&st, &cnt); err != nil {
			return nil, fmt.Errorf("task stats by status scan: %w", err)
		}
		byStatus[st] = cnt
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("task stats by status rows: %w", err)
	}

	const qAssignee = `
		SELECT COALESCE(assignee_id::text, '__unassigned__') AS assignee_key, COUNT(*) AS cnt
		FROM tasks
		WHERE project_id = $1
		GROUP BY assignee_id`
	rows2, err := r.db.QueryxContext(ctx, qAssignee, projectID)
	if err != nil {
		return nil, fmt.Errorf("task stats by assignee: %w", err)
	}
	defer func() { _ = rows2.Close() }()

	byAssignee := make(map[string]int64)
	for rows2.Next() {
		var key string
		var cnt int64
		if err := rows2.Scan(&key, &cnt); err != nil {
			return nil, fmt.Errorf("task stats by assignee scan: %w", err)
		}
		byAssignee[key] = cnt
	}
	if err := rows2.Err(); err != nil {
		return nil, fmt.Errorf("task stats by assignee rows: %w", err)
	}

	return &domain.ProjectTaskStats{ByStatus: byStatus, ByAssignee: byAssignee}, nil
}

func (r *taskRepository) InvolvedInProject(ctx context.Context, userID, projectID uuid.UUID) (bool, error) {
	const q = `
		SELECT EXISTS (
			SELECT 1 FROM tasks
			WHERE project_id = $1 AND (assignee_id = $2 OR created_by = $2)
		)`
	var ok bool
	if err := sqlx.GetContext(ctx, r.db, &ok, q, projectID, userID); err != nil {
		return false, fmt.Errorf("task involved in project: %w", err)
	}
	return ok, nil
}

func (r *taskRepository) Update(ctx context.Context, id uuid.UUID, req domain.UpdateTaskRequest) error {
	var sets []string
	var args []any
	n := 1

	if req.Title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", n))
		args = append(args, *req.Title)
		n++
	}
	if req.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", n))
		args = append(args, *req.Description)
		n++
	}
	if req.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", n))
		args = append(args, string(*req.Status))
		n++
	}
	if req.Priority != nil {
		sets = append(sets, fmt.Sprintf("priority = $%d", n))
		args = append(args, string(*req.Priority))
		n++
	}
	if req.AssigneeID != nil {
		sets = append(sets, fmt.Sprintf("assignee_id = $%d", n))
		args = append(args, *req.AssigneeID)
		n++
	}
	if req.DueDate != nil {
		sets = append(sets, fmt.Sprintf("due_date = $%d", n))
		args = append(args, *req.DueDate)
		n++
	}
	if len(sets) == 0 {
		return fmt.Errorf("task update: no fields to update")
	}

	args = append(args, id)
	q := fmt.Sprintf(`
		UPDATE tasks
		SET %s
		WHERE id = $%d`, strings.Join(sets, ", "), n)

	res, err := r.db.ExecContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("task update: %w", err)
	}
	aff, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("task update rows affected: %w", err)
	}
	if aff == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *taskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM tasks WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("task delete: %w", err)
	}
	aff, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("task delete rows affected: %w", err)
	}
	if aff == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *taskRepository) FindByAssigneeID(ctx context.Context, assigneeID uuid.UUID) ([]domain.Task, error) {
	const q = `
		SELECT id, title, description, status, priority, project_id, assignee_id, created_by, due_date, created_at, updated_at
		FROM tasks
		WHERE assignee_id = $1
		ORDER BY due_date NULLS LAST, created_at ASC`
	var tasks []domain.Task
	if err := sqlx.SelectContext(ctx, r.db, &tasks, q, assigneeID); err != nil {
		return nil, fmt.Errorf("task find by assignee: %w", err)
	}
	if tasks == nil {
		tasks = []domain.Task{}
	}
	return tasks, nil
}
