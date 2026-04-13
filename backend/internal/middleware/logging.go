package middleware

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/taskflow/backend/internal/observability"
)

const (
	// HeaderRequestID is the standard header for correlating logs and traces.
	HeaderRequestID = "X-Request-ID"
	// ContextRequestID is the gin context key for the request ID string.
	ContextRequestID = "request_id"
)

// LoggingMiddleware records each request using slog: method, path, status, duration, optional error,
// and a stable request ID (from X-Request-ID or generated).
//
// Example:
//
//	r := gin.New()
//	r.Use(middleware.LoggingMiddleware())
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(HeaderRequestID)
		if reqID == "" {
			reqID = uuid.NewString()
		}
		c.Writer.Header().Set(HeaderRequestID, reqID)
		c.Set(ContextRequestID, reqID)
		c.Request = c.Request.WithContext(observability.ContextWithRequestID(c.Request.Context(), reqID))

		start := time.Now()
		c.Next()
		duration := time.Since(start)

		clientIP := c.ClientIP()
		ua := strings.TrimSpace(c.Request.UserAgent())
		if len(ua) > 256 {
			ua = ua[:256] + "..."
		}

		attrs := []any{
			"request_id", reqID,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", duration.Milliseconds(),
			"client_ip", clientIP,
			"user_agent", ua,
		}

		if len(c.Errors) > 0 {
			attrs = append(attrs, "error", c.Errors.String())
		}

		slog.Info("http_request", attrs...)
	}
}
