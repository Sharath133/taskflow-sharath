package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// RecoveryMiddleware recovers from panics, logs the value and full stack trace, and returns 500 JSON.
// Client-facing text is generic; details stay in logs only.
//
// Example:
//
//	r := gin.New()
//	r.Use(middleware.RecoveryMiddleware())
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := debug.Stack()
				reqID, _ := c.Get(ContextRequestID)
				slog.Error("panic_recovered",
					"request_id", reqID,
					"error", rec,
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"stack", string(stack),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":   "internal_error",
					"message": "an unexpected error occurred",
				})
			}
		}()
		c.Next()
	}
}
