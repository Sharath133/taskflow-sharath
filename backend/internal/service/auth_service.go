package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	"github.com/taskflow/backend/internal/auth"
	"github.com/taskflow/backend/internal/domain"
	"github.com/taskflow/backend/internal/repository"
)

// AuthService issues credentials and JWT access tokens.
type AuthService interface {
	Register(ctx context.Context, req domain.RegisterRequest) (*domain.AuthResponse, error)
	Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error)
}

type authService struct {
	users      repository.UserRepository
	jwtSecret  string
	jwtExpiry  time.Duration
	bcryptCost int
}

// NewAuthService wires registration and login with injected configuration.
func NewAuthService(users repository.UserRepository, jwtSecret string, jwtExpiry time.Duration, bcryptCost int) AuthService {
	return &authService{
		users:      users,
		jwtSecret:  jwtSecret,
		jwtExpiry:  jwtExpiry,
		bcryptCost: bcryptCost,
	}
}

// Register validates input, ensures email uniqueness, hashes the password, persists the user, and returns a JWT.
func (s *authService) Register(ctx context.Context, req domain.RegisterRequest) (*domain.AuthResponse, error) {
	if err := validateStruct(&req); err != nil {
		return nil, err
	}

	email := normalizeEmail(req.Email)
	if err := s.ensureEmailAvailable(ctx, email); err != nil {
		return nil, err
	}

	hash, err := auth.HashPassword(req.Password, s.bcryptCost)
	if err != nil {
		if errors.Is(err, auth.ErrPasswordTooWeak) {
			return nil, domain.NewValidationError(domain.FieldMessage("password", "must be at least 8 characters"))
		}
		return nil, fmt.Errorf("hash password: %w", err)
	}

	u := &domain.User{
		Name:     strings.TrimSpace(req.Name),
		Email:    email,
		Password: hash,
	}
	if err := s.users.Create(ctx, u); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, fmt.Errorf("%w: email already registered", domain.ErrConflict)
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return s.buildAuthResponse(u)
}

// Login validates credentials and returns a JWT for the matching user.
func (s *authService) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error) {
	if err := validateStruct(&req); err != nil {
		return nil, err
	}

	email := normalizeEmail(req.Email)
	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("%w: invalid email or password", domain.ErrUnauthorized)
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	if err := auth.ComparePassword(u.Password, req.Password); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, fmt.Errorf("%w: invalid email or password", domain.ErrUnauthorized)
		}
		return nil, fmt.Errorf("compare password: %w", err)
	}

	return s.buildAuthResponse(u)
}

func (s *authService) ensureEmailAvailable(ctx context.Context, email string) error {
	_, err := s.users.FindByEmail(ctx, email)
	if err == nil {
		return fmt.Errorf("%w: email already registered", domain.ErrConflict)
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("check email uniqueness: %w", err)
	}
	return nil
}

func (s *authService) buildAuthResponse(u *domain.User) (*domain.AuthResponse, error) {
	token, err := auth.GenerateToken(u.ID, u.Email, s.jwtSecret, s.jwtExpiry)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	return &domain.AuthResponse{
		Token: token,
		User:  u.ToResponse(),
	}, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
