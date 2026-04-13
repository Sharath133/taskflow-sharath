package service

import (
	"context"

	"github.com/taskflow/backend/internal/domain"
)

// UserReader loads users (expand with concrete repository interfaces).
type UserReader interface {
	FindByID(ctx context.Context, id string) (*domain.User, error)
}

// UserService coordinates user-related use cases.
type UserService struct {
	users UserReader
}

// NewUserService constructs a UserService with injected dependencies.
func NewUserService(users UserReader) *UserService {
	return &UserService{users: users}
}
