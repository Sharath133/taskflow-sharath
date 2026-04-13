package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/taskflow/backend/internal/domain"
)

// Repositories groups all repository interfaces for dependency injection.
type Repositories struct {
	Users    UserRepository
	Projects ProjectRepository
	Tasks    TaskRepository
}

// NewRepositories wires PostgreSQL-backed implementations.
func NewRepositories(db *sqlx.DB) *Repositories {
	return &Repositories{
		Users:    NewUserRepository(db),
		Projects: NewProjectRepository(db),
		Tasks:    NewTaskRepository(db),
	}
}

func mapSingleRowErr(err error, op string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	return fmt.Errorf("%s: %w", op, err)
}
