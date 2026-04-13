package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/taskflow/backend/internal/domain"
	"github.com/taskflow/backend/internal/service"
)

type noopAuthService struct{}

func (noopAuthService) Register(context.Context, domain.RegisterRequest) (*domain.AuthResponse, error) {
	return nil, errors.New("unexpected register call")
}

func (noopAuthService) Login(context.Context, domain.LoginRequest) (*domain.AuthResponse, error) {
	return nil, errors.New("unexpected login call")
}

type stubAuthService struct {
	registerFn func(context.Context, domain.RegisterRequest) (*domain.AuthResponse, error)
	loginFn    func(context.Context, domain.LoginRequest) (*domain.AuthResponse, error)
}

func (s *stubAuthService) Register(ctx context.Context, req domain.RegisterRequest) (*domain.AuthResponse, error) {
	if s.registerFn != nil {
		return s.registerFn(ctx, req)
	}
	return nil, errors.New("register not stubbed")
}

func (s *stubAuthService) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error) {
	if s.loginFn != nil {
		return s.loginFn(ctx, req)
	}
	return nil, errors.New("login not stubbed")
}

// TestRegister_ValidationError_InvalidEmail ensures validation runs before the service layer (integration-style handler test).
func TestRegister_ValidationError_InvalidEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAuthHandler(noopAuthService{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"Test","email":"not-an-email","password":"password123"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte(body)))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Register(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "validation failed" {
		t.Fatalf("expected validation failed error, got %#v", resp["error"])
	}
}

// TestRegister_Success_JSONShape verifies successful registration response wraps token and user under data.
func TestRegister_Success_JSONShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uid := uuid.MustParse("10000000-0000-4000-8000-000000000099")
	fixedTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	svc := &stubAuthService{
		registerFn: func(ctx context.Context, req domain.RegisterRequest) (*domain.AuthResponse, error) {
			return &domain.AuthResponse{
				Token: "test.jwt.token",
				User: domain.UserResponse{
					ID:        uid,
					Name:      req.Name,
					Email:     req.Email,
					CreatedAt: fixedTime,
				},
			}, nil
		},
	}
	h := NewAuthHandler(svc)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"Alice","email":"alice@example.com","password":"password123"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte(body)))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Register(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}
	var outer struct {
		Data struct {
			Token string               `json:"token"`
			User  domain.UserResponse `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &outer); err != nil {
		t.Fatal(err)
	}
	if outer.Data.Token != "test.jwt.token" {
		t.Fatalf("unexpected token: %q", outer.Data.Token)
	}
	if outer.Data.User.Email != "alice@example.com" {
		t.Fatalf("unexpected user email: %q", outer.Data.User.Email)
	}
}

// TestLogin_InvalidCredentials_Returns401 maps domain.ErrUnauthorized to 401 for failed login.
func TestLogin_InvalidCredentials_Returns401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubAuthService{
		loginFn: func(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error) {
			return nil, fmt.Errorf("%w: invalid email or password", domain.ErrUnauthorized)
		},
	}
	h := NewAuthHandler(svc)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"email":"nobody@example.com","password":"wrong"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte(body)))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Login(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
}

// Compile-time check: stubs implement service interfaces.
var (
	_ service.AuthService = noopAuthService{}
	_ service.AuthService = (*stubAuthService)(nil)
)
