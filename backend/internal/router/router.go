package router

import (
	"log/slog"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/taskflow/backend/internal/handlers"
	"github.com/taskflow/backend/internal/middleware"
	"github.com/taskflow/backend/internal/service"
)

// SetupRouter builds the Gin engine with middleware and routes.
// Gin mode (GIN_MODE in env, validated in config.Load) should be applied by the caller before SetupRouter.
// Trusted proxies: TRUSTED_PROXIES (comma-separated); empty defaults to loopback only.
// CORS origins: CORS_ALLOWED_ORIGINS (comma-separated); empty uses * without credentials.
func SetupRouter(services *service.Services, jwtSecret string) *gin.Engine {
	r := gin.New()

	if err := configureTrustedProxies(r); err != nil {
		slog.Warn("trusted_proxies invalid, using loopback only", "error", err)
		_ = r.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	}

	r.Use(middleware.RecoveryMiddleware())
	r.Use(middleware.LoggingMiddleware())
	r.Use(middleware.CORS(parseCSVEnv("CORS_ALLOWED_ORIGINS")))

	r.GET("/health", handlers.Health)

	authHandler := handlers.NewAuthHandler(services.Auth)
	r.POST("/auth/register", authHandler.Register)
	r.POST("/auth/login", authHandler.Login)

	protected := r.Group("")
	protected.Use(middleware.AuthMiddleware(jwtSecret))

	projectHandler := handlers.NewProjectHandler(services.Project, services.Task)
	protected.GET("/projects", projectHandler.List)
	protected.POST("/projects", projectHandler.Create)
	protected.GET("/projects/:id/stats", projectHandler.Stats)
	protected.GET("/projects/:id", projectHandler.Get)
	protected.PATCH("/projects/:id", projectHandler.Update)
	protected.DELETE("/projects/:id", projectHandler.Delete)

	taskHandler := handlers.NewTaskHandler(services.Task)
	protected.GET("/projects/:id/tasks", taskHandler.ListByProject)
	protected.POST("/projects/:id/tasks", taskHandler.Create)
	protected.PATCH("/tasks/:id", taskHandler.Update)
	protected.DELETE("/tasks/:id", taskHandler.Delete)

	return r
}

func configureTrustedProxies(r *gin.Engine) error {
	raw := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES"))
	if raw == "" {
		return r.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	}
	return r.SetTrustedProxies(parseCSVEnv("TRUSTED_PROXIES"))
}

func parseCSVEnv(key string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}
