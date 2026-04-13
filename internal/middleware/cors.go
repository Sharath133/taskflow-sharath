package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	corsAllowMethods = "GET, POST, PATCH, DELETE, OPTIONS"
	corsAllowHeaders = "Authorization, Content-Type"
)

// CORS returns middleware for browser clients. When allowedOrigins is non-empty, requests are checked
// against that list, Access-Control-Allow-Origin echoes the request Origin, and credentials are enabled.
// When allowedOrigins is empty, Allow-Origin is * and credentials are not set (browser credentialed
// requests require an explicit origin match).
func CORS(allowedOrigins []string) gin.HandlerFunc {
	trimmed := make([]string, 0, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if s := strings.TrimSpace(o); s != "" {
			trimmed = append(trimmed, s)
		}
	}

	allow := make(map[string]struct{}, len(trimmed))
	for _, o := range trimmed {
		allow[o] = struct{}{}
	}

	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Methods", corsAllowMethods)
		c.Header("Access-Control-Allow-Headers", corsAllowHeaders)

		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if len(trimmed) > 0 {
			if origin != "" {
				if _, ok := allow[origin]; ok {
					c.Header("Access-Control-Allow-Origin", origin)
					c.Header("Access-Control-Allow-Credentials", "true")
				}
			}
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
