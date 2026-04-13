package service

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/taskflow/backend/internal/domain"
)

var structValidator = validator.New()

func validateStruct(s any) error {
	if err := structValidator.Struct(s); err != nil {
		return toValidationError(err)
	}
	return nil
}

func toValidationError(err error) error {
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return err
	}
	fields := make([]domain.FieldError, 0, len(verrs))
	for _, fe := range verrs {
		fields = append(fields, domain.FieldError{
			Field:   strings.ToLower(fe.Field()),
			Message: fieldErrorMessage(fe),
		})
	}
	return domain.NewValidationError(fields...)
}

func fieldErrorMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return "is too short"
	case "max":
		return "is too long"
	case "oneof":
		return "must be one of the allowed values"
	case "omitempty":
		return "is invalid"
	default:
		return "is invalid"
	}
}
