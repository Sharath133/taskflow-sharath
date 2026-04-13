package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/taskflow/backend/internal/domain"
)

// ProjectRepository persists and loads projects.
type ProjectRepository interface {
	Create(ctx context.Context, p *domain.Project) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Project, error)
	FindByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]domain.Project, error)
	FindAccessibleByUserID(ctx context.Context, userID uuid.UUID, limit, offset *int) ([]domain.Project, error)
	CountAccessibleByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
	Update(ctx context.Context, id uuid.UUID, req domain.UpdateProjectRequest) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type projectRepository struct {
	db sqlxConn
}

// NewProjectRepository returns a PostgreSQL ProjectRepository.
func NewProjectRepository(db sqlxConn) ProjectRepository {
	return &projectRepository{db: db}
}

func (r *projectRepository) Create(ctx context.Context, p *domain.Project) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	const q = `
		INSERT INTO projects (id, name, description, owner_id)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at`
	if err := r.db.QueryRowContext(ctx, q, p.ID, p.Name, p.Description, p.OwnerID).Scan(&p.CreatedAt); err != nil {
		return fmt.Errorf("project create: %w", err)
	}
	return nil
}

func (r *projectRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	const q = `
		SELECT id, name, description, owner_id, created_at
		FROM projects
		WHERE id = $1`
	var p domain.Project
	if err := sqlx.GetContext(ctx, r.db, &p, q, id); err != nil {
		return nil, mapSingleRowErr(err, "project find by id")
	}
	return &p, nil
}

func (r *projectRepository) FindByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]domain.Project, error) {
	const q = `
		SELECT id, name, description, owner_id, created_at
		FROM projects
		WHERE owner_id = $1
		ORDER BY created_at DESC`
	var projects []domain.Project
	if err := sqlx.SelectContext(ctx, r.db, &projects, q, ownerID); err != nil {
		return nil, fmt.Errorf("project find by owner: %w", err)
	}
	if projects == nil {
		projects = []domain.Project{}
	}
	return projects, nil
}

func (r *projectRepository) FindAccessibleByUserID(ctx context.Context, userID uuid.UUID, limit, offset *int) ([]domain.Project, error) {
	q := `
		SELECT DISTINCT p.id, p.name, p.description, p.owner_id, p.created_at
		FROM projects p
		WHERE p.owner_id = $1
		   OR EXISTS (
			 SELECT 1 FROM tasks t
			 WHERE t.project_id = p.id AND t.assignee_id = $1
		   )
		   OR EXISTS (
			 SELECT 1 FROM tasks t
			 WHERE t.project_id = p.id AND t.created_by = $1
		   )
		ORDER BY p.created_at DESC`
	args := []any{userID}
	if limit != nil {
		off := 0
		if offset != nil {
			off = *offset
		}
		q += ` LIMIT $2 OFFSET $3`
		args = append(args, *limit, off)
	}
	var projects []domain.Project
	if err := sqlx.SelectContext(ctx, r.db, &projects, q, args...); err != nil {
		return nil, fmt.Errorf("project find accessible: %w", err)
	}
	if projects == nil {
		projects = []domain.Project{}
	}
	return projects, nil
}

func (r *projectRepository) CountAccessibleByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	const q = `
		SELECT COUNT(*) FROM (
			SELECT DISTINCT p.id
			FROM projects p
			WHERE p.owner_id = $1
			   OR EXISTS (
				 SELECT 1 FROM tasks t
				 WHERE t.project_id = p.id AND t.assignee_id = $1
			   )
			   OR EXISTS (
				 SELECT 1 FROM tasks t
				 WHERE t.project_id = p.id AND t.created_by = $1
			   )
		) sub`
	var n int64
	if err := sqlx.GetContext(ctx, r.db, &n, q, userID); err != nil {
		return 0, fmt.Errorf("project count accessible: %w", err)
	}
	return n, nil
}

func (r *projectRepository) Update(ctx context.Context, id uuid.UUID, req domain.UpdateProjectRequest) error {
	var sets []string
	var args []any
	n := 1

	if req.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", n))
		args = append(args, *req.Name)
		n++
	}
	if req.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", n))
		args = append(args, *req.Description)
		n++
	}
	if len(sets) == 0 {
		return fmt.Errorf("project update: no fields to update")
	}

	args = append(args, id)
	q := fmt.Sprintf(`
		UPDATE projects
		SET %s
		WHERE id = $%d`, strings.Join(sets, ", "), n)

	res, err := r.db.ExecContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("project update: %w", err)
	}
	aff, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("project update rows affected: %w", err)
	}
	if aff == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Delete removes a project. Task rows cascade via FK in a single atomic DELETE.
func (r *projectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM projects WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("project delete: %w", err)
	}
	aff, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("project delete rows affected: %w", err)
	}
	if aff == 0 {
		return domain.ErrNotFound
	}
	return nil
}
