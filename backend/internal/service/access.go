package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/taskflow/backend/internal/domain"
	"github.com/taskflow/backend/internal/repository"
)

func loadProject(ctx context.Context, projects repository.ProjectRepository, projectID uuid.UUID) (*domain.Project, error) {
	p, err := projects.FindByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("load project: %w", err)
	}
	return p, nil
}

// assertProjectAccessible allows owners and users who are assignee or creator on at least one task in the project.
func assertProjectAccessible(
	ctx context.Context,
	projects repository.ProjectRepository,
	tasks repository.TaskRepository,
	userID, projectID uuid.UUID,
) (*domain.Project, error) {
	p, err := loadProject(ctx, projects, projectID)
	if err != nil {
		return nil, err
	}
	if p.OwnerID == userID {
		return p, nil
	}
	ok, err := tasks.InvolvedInProject(ctx, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("check project access: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("%w: you do not have access to this project", domain.ErrUnauthorized)
	}
	return p, nil
}

func assertProjectOwner(p *domain.Project, userID uuid.UUID) error {
	if p == nil {
		return domain.ErrNotFound
	}
	if p.OwnerID != userID {
		return fmt.Errorf("%w: only the project owner can perform this action", domain.ErrUnauthorized)
	}
	return nil
}
