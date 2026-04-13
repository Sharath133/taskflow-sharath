package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/taskflow/backend/internal/domain"
)

// parseUUIDParam parses c.Param(name) as a UUID. On failure writes 400 with fields keyed by param name.
func parseUUIDParam(c *gin.Context, name string) (uuid.UUID, bool) {
	raw := c.Param(name)
	id, err := uuid.Parse(raw)
	if err != nil {
		errorResponse(c, http.StatusBadRequest, domain.ErrValidation.Error(), map[string]string{
			name: "must be a valid UUID",
		})
		return uuid.Nil, false
	}
	return id, true
}
