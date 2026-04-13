package domain

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors for cross-layer error handling (map to HTTP status in handlers).
var (
	ErrNotFound     = errors.New("resource not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("resource conflict")
	ErrValidation   = errors.New("validation failed")
)

// FieldError describes a single invalid input field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError carries field-level validation failures.
type ValidationError struct {
	Fields []FieldError `json:"fields"`
}

// Error implements error.
func (e *ValidationError) Error() string {
	if e == nil || len(e.Fields) == 0 {
		return ErrValidation.Error()
	}
	var b strings.Builder
	b.WriteString(ErrValidation.Error())
	b.WriteString(": ")
	for i, f := range e.Fields {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(f.Field)
		b.WriteString(" ")
		b.WriteString(f.Message)
	}
	return b.String()
}

// Unwrap supports errors.Is(e, ErrValidation).
func (e *ValidationError) Unwrap() error {
	return ErrValidation
}

// NewValidationError builds a ValidationError from a list of field problems.
func NewValidationError(fields ...FieldError) *ValidationError {
	return &ValidationError{Fields: append([]FieldError(nil), fields...)}
}

// FieldMessage is a small helper for a single-field validation error.
func FieldMessage(field, message string) FieldError {
	return FieldError{Field: field, Message: message}
}

// Errorf builds a FieldError with a formatted message.
func Errorf(field, format string, args ...any) FieldError {
	return FieldError{Field: field, Message: fmt.Sprintf(format, args...)}
}
