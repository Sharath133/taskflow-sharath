package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/taskflow/backend/internal/domain"
)

// ParseOptionalPagination reads ?page= and ?limit=. If both are absent, returns (nil, nil, false).
// If either is present, validates integers (page and limit must be ≥ 1 when provided).
// On validation failure, writes 400 and returns handled=true.
func ParseOptionalPagination(c *gin.Context) (page, limit *int, handled bool) {
	pq := strings.TrimSpace(c.Query("page"))
	lq := strings.TrimSpace(c.Query("limit"))
	if pq == "" && lq == "" {
		return nil, nil, false
	}
	if pq != "" {
		p, err := strconv.Atoi(pq)
		if err != nil || p < 1 {
			errorResponse(c, http.StatusBadRequest, domain.ErrValidation.Error(), map[string]string{
				"page": "must be a positive integer",
			})
			return nil, nil, true
		}
		pp := p
		page = &pp
	}
	if lq != "" {
		l, err := strconv.Atoi(lq)
		if err != nil || l < 1 {
			errorResponse(c, http.StatusBadRequest, domain.ErrValidation.Error(), map[string]string{
				"limit": "must be a positive integer",
			})
			return nil, nil, true
		}
		ll := l
		limit = &ll
	}
	return page, limit, false
}
