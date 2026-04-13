package service

import (
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/taskflow/backend/internal/repository"
)

// Services groups application services for dependency injection.
type Services struct {
	Auth    AuthService
	Project ProjectService
	Task    TaskService
}

// NewServices constructs all services with shared repositories and auth settings.
// db is used for transactional workflows where multiple repository calls must commit atomically.
func NewServices(db *sqlx.DB, repos *repository.Repositories, jwtSecret string, jwtExpiry time.Duration, bcryptCost int) *Services {
	return &Services{
		Auth:    NewAuthService(repos.Users, jwtSecret, jwtExpiry, bcryptCost),
		Project: NewProjectService(db, repos.Projects, repos.Tasks),
		Task:    NewTaskService(db, repos.Tasks, repos.Projects, repos.Users),
	}
}
