package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/taskflow/backend/internal/auth"
	"github.com/taskflow/backend/internal/domain"
)

const bearerPrefix = "Bearer "

// ContextJWTClaims is the gin context key for *domain.Claims after successful AuthMiddleware.
const ContextJWTClaims = "jwt_claims"

// AuthMiddleware enforces Authorization: Bearer <JWT>, validates with HS256, and stores claims.
//
// Security notes:
//   - All authentication failures return the same JSON body and status to avoid leaking whether
//     a token was malformed, expired, or signature-invalid.
//   - Rate limiting is not implemented here; place a reverse proxy (e.g. nginx, Envoy) or a
//     dedicated rate-limiter middleware in front of login and token-protected routes to mitigate
//     brute-force and token stuffing.
//
// Example:
//
//	authorized := r.Group("/api/v1")
//	authorized.Use(middleware.AuthMiddleware(cfg.JWTSecret))
//	authorized.GET("/me", handlers.Me)
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := strings.TrimSpace(c.GetHeader("Authorization"))
		if h == "" || !strings.HasPrefix(h, bearerPrefix) {
			unauthorizedJSON(c)
			return
		}
		raw := strings.TrimSpace(strings.TrimPrefix(h, bearerPrefix))
		if raw == "" {
			unauthorizedJSON(c)
			return
		}

		claims, err := auth.ValidateToken(raw, jwtSecret)
		if err != nil {
			if errors.Is(err, auth.ErrTokenExpired) {
				slog.Debug("jwt rejected", "reason", "expired", "path", c.Request.URL.Path)
			} else {
				slog.Debug("jwt rejected", "reason", "invalid", "path", c.Request.URL.Path)
			}
			unauthorizedJSON(c)
			return
		}

		c.Set(ContextJWTClaims, claims)
		c.Next()
	}
}

// GetUserIDFromContext returns the authenticated user id from claims set by AuthMiddleware.
// Returns uuid.Nil if the route was not protected or claims are missing/invalid.
//
// Example:
//
//	uid := middleware.GetUserIDFromContext(c)
//	if uid == uuid.Nil { /* should not happen after AuthMiddleware */ }
func GetUserIDFromContext(c *gin.Context) uuid.UUID {
	v, ok := c.Get(ContextJWTClaims)
	if !ok {
		return uuid.Nil
	}
	claims, ok := v.(*domain.Claims)
	if !ok {
		return uuid.Nil
	}
	id, err := claims.ParsedUserID()
	if err != nil {
		return uuid.Nil
	}
	return id
}

func unauthorizedJSON(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": "unauthorized",
	})
}
