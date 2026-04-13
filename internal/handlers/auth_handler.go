package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/taskflow/backend/internal/domain"
	"github.com/taskflow/backend/internal/service"
)

// AuthHandler exposes registration and login endpoints.
type AuthHandler struct {
	svc service.AuthService
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Register handles POST /auth/register.
func (h *AuthHandler) Register(c *gin.Context) {
	var req domain.RegisterRequest
	if !BindJSONAndValidate(c, &req) {
		return
	}

	resp, err := h.svc.Register(c.Request.Context(), req)
	if err != nil {
		if handleRegisterError(c, err) {
			return
		}
		slog.Error("auth register failed", "error", err)
		errorResponse(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	successResponse(c, http.StatusCreated, resp)
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if !BindJSONAndValidate(c, &req) {
		return
	}

	resp, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		var verr *domain.ValidationError
		if errors.As(err, &verr) {
			validationErrorResponse(c, verr)
			return
		}
		if errors.Is(err, domain.ErrUnauthorized) {
			errorResponse(c, http.StatusUnauthorized, "invalid credentials", nil)
			return
		}
		slog.Error("auth login failed", "error", err)
		errorResponse(c, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	successResponse(c, http.StatusOK, resp)
}

func handleRegisterError(c *gin.Context, err error) bool {
	var verr *domain.ValidationError
	if errors.As(err, &verr) {
		validationErrorResponse(c, verr)
		return true
	}
	if errors.Is(err, domain.ErrConflict) {
		errorResponse(c, http.StatusConflict, "email already registered", nil)
		return true
	}
	return false
}
