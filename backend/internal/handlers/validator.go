package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/taskflow/backend/internal/domain"
)

var (
	validateOnce sync.Once
	validateInst *validator.Validate
)

func getValidator() *validator.Validate {
	validateOnce.Do(func() {
		v := validator.New()
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "" || name == "-" {
				return fld.Name
			}
			return name
		})
		validateInst = v
	})
	return validateInst
}

// BindJSONAndValidate binds JSON into dst and runs struct validation tags.
// On failure it writes a 400 response and returns false.
func BindJSONAndValidate(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		writeBindError(c, err)
		return false
	}
	return ValidateStruct(c, dst)
}

// ValidateStruct runs struct validation tags on dst. On failure it writes 400 and returns false.
func ValidateStruct(c *gin.Context, dst any) bool {
	if verr := structToValidationError(getValidator().Struct(dst)); verr != nil {
		validationErrorResponse(c, verr)
		return false
	}
	return true
}

func writeBindError(c *gin.Context, err error) {
	var verr *domain.ValidationError
	if errors.As(err, &verr) {
		validationErrorResponse(c, verr)
		return
	}
	errorResponse(c, http.StatusBadRequest, domain.ErrValidation.Error(), map[string]string{
		"body": "must be valid JSON",
	})
}

// structToValidationError converts go-playground validator errors to domain.ValidationError (camelCase field names via json tags).
func structToValidationError(err error) *domain.ValidationError {
	if err == nil {
		return nil
	}
	var vErrs validator.ValidationErrors
	if !errors.As(err, &vErrs) {
		return domain.NewValidationError(domain.FieldMessage("_body", err.Error()))
	}
	fields := make([]domain.FieldError, 0, len(vErrs))
	for _, fe := range vErrs {
		fields = append(fields, domain.FieldMessage(fe.Field(), validationMessage(fe)))
	}
	return domain.NewValidationError(fields...)
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email"
	case "min":
		if fe.Param() != "" {
			return fmt.Sprintf("must be at least %s", fe.Param())
		}
		return "is too short"
	case "max":
		if fe.Param() != "" {
			return fmt.Sprintf("must be at most %s", fe.Param())
		}
		return "is too long"
	case "oneof":
		return fmt.Sprintf("must be one of: %s", strings.ReplaceAll(fe.Param(), " ", ", "))
	case "omitempty":
		return "is invalid"
	default:
		return fmt.Sprintf("failed validation on %q", fe.Tag())
	}
}
