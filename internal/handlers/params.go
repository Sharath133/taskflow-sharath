package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// parseUUIDParam parses c.Param(name) as a UUID. On failure writes 400 with fields keyed by param name.
func parseUUIDParam(c *gin.Context, name string) (uuid.UUID, bool) {
	raw := c.Param(name)
	id, err := uuid.Parse(raw)
	if err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid path parameter", map[string]string{
			name: "must be a valid UUID",
		})
		return uuid.Nil, false
	}
	return id, true
}
