package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/taskflow/backend/internal/domain"
)

// writeJSON writes a JSON response with the given status (Appendix A shape — no `data` envelope).
func writeJSON(c *gin.Context, status int, payload any) {
	c.JSON(status, payload)
}

// errorResponse writes JSON: {"error": message, "fields": fields}.
// fields may be nil; it is omitted when empty if we use map - actually gin.H with nil might serialize "fields": null.
func errorResponse(c *gin.Context, status int, message string, fields map[string]string) {
	body := gin.H{"error": message}
	if len(fields) > 0 {
		body["fields"] = fields
	}
	c.JSON(status, body)
}

// validationErrorResponse writes a 400 response from domain.ValidationError using field keys as map keys.
func validationErrorResponse(c *gin.Context, verr *domain.ValidationError) {
	if verr == nil || len(verr.Fields) == 0 {
		errorResponse(c, http.StatusBadRequest, domain.ErrValidation.Error(), nil)
		return
	}
	fields := make(map[string]string, len(verr.Fields))
	for _, f := range verr.Fields {
		fields[f.Field] = f.Message
	}
	errorResponse(c, http.StatusBadRequest, domain.ErrValidation.Error(), fields)
}

// HandleServiceError maps common domain errors to JSON responses. Returns true if a response was written for err.
func HandleServiceError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	var verr *domain.ValidationError
	if errors.As(err, &verr) {
		validationErrorResponse(c, verr)
		return true
	}
	switch {
	case errors.Is(err, domain.ErrNotFound):
		errorResponse(c, http.StatusNotFound, "not found", nil)
		return true
	case errors.Is(err, domain.ErrForbidden):
		errorResponse(c, http.StatusForbidden, "forbidden", nil)
		return true
	case errors.Is(err, domain.ErrUnauthorized):
		// Used after AuthMiddleware for insufficient access (distinct from 401 unauthenticated on login).
		errorResponse(c, http.StatusForbidden, "forbidden", nil)
		return true
	default:
		return false
	}
}
