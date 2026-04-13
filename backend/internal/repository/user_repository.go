package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/taskflow/backend/internal/domain"
)

// UserRepository persists and loads users.
type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	List(ctx context.Context) ([]domain.User, error)
}

type userRepository struct {
	db sqlxConn
}

// NewUserRepository returns a PostgreSQL UserRepository.
func NewUserRepository(db sqlxConn) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, u *domain.User) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	const q = `
		INSERT INTO users (id, name, email, password)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at`
	if err := r.db.QueryRowContext(ctx, q, u.ID, u.Name, u.Email, u.Password).Scan(&u.CreatedAt); err != nil {
		return fmt.Errorf("user create: %w", err)
	}
	return nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	const q = `
		SELECT id, name, email, password, created_at
		FROM users
		WHERE email = $1`
	var u domain.User
	if err := sqlx.GetContext(ctx, r.db, &u, q, email); err != nil {
		return nil, mapSingleRowErr(err, "user find by email")
	}
	return &u, nil
}

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const q = `
		SELECT id, name, email, password, created_at
		FROM users
		WHERE id = $1`
	var u domain.User
	if err := sqlx.GetContext(ctx, r.db, &u, q, id); err != nil {
		return nil, mapSingleRowErr(err, "user find by id")
	}
	return &u, nil
}

func (r *userRepository) List(ctx context.Context) ([]domain.User, error) {
	const q = `
		SELECT id, name, email, password, created_at
		FROM users
		ORDER BY created_at ASC`
	var users []domain.User
	if err := sqlx.SelectContext(ctx, r.db, &users, q); err != nil {
		return nil, fmt.Errorf("user list: %w", err)
	}
	if users == nil {
		users = []domain.User{}
	}
	return users, nil
}
