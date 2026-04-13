package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TaskStatus is persisted as VARCHAR; must match DB check constraint.
type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
)

var allowedTaskStatuses = map[TaskStatus]struct{}{
	TaskStatusTodo:       {},
	TaskStatusInProgress: {},
	TaskStatusDone:       {},
}

// AllowedTaskStatusValuesMessage is returned in API validation errors for invalid status values.
func AllowedTaskStatusValuesMessage() string {
	return "must be one of: todo, in_progress, done"
}

// ParseTaskStatus normalizes and validates s.
func ParseTaskStatus(s string) (TaskStatus, error) {
	st := TaskStatus(strings.TrimSpace(strings.ToLower(s)))
	if _, ok := allowedTaskStatuses[st]; !ok {
		return "", fmt.Errorf("invalid task status: %q", s)
	}
	return st, nil
}

// MarshalJSON emits a JSON string for TaskStatus.
func (s TaskStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

// UnmarshalJSON parses a JSON string into TaskStatus.
func (s *TaskStatus) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if strings.TrimSpace(raw) == "" {
		*s = ""
		return nil
	}
	parsed, err := ParseTaskStatus(raw)
	if err != nil {
		return NewValidationError(FieldMessage("status", AllowedTaskStatusValuesMessage()))
	}
	*s = parsed
	return nil
}

// Scan implements sql.Scanner for PostgreSQL text/varchar.
func (s *TaskStatus) Scan(src interface{}) error {
	if src == nil {
		*s = ""
		return nil
	}
	var str string
	switch v := src.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return fmt.Errorf("cannot scan %T into TaskStatus", src)
	}
	parsed, err := ParseTaskStatus(str)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

// Value implements driver.Valuer.
func (s TaskStatus) Value() (driver.Value, error) {
	if s == "" {
		return nil, nil
	}
	if _, ok := allowedTaskStatuses[s]; !ok {
		return nil, fmt.Errorf("invalid task status: %q", s)
	}
	return string(s), nil
}

// UnmarshalParam binds query/form parameters (e.g. Gin) into TaskStatus.
func (s *TaskStatus) UnmarshalParam(param string) error {
	if param == "" {
		*s = ""
		return nil
	}
	parsed, err := ParseTaskStatus(param)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

// TaskPriority is persisted as VARCHAR; must match DB check constraint.
type TaskPriority string

const (
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityMedium TaskPriority = "medium"
	TaskPriorityHigh   TaskPriority = "high"
)

var allowedTaskPriorities = map[TaskPriority]struct{}{
	TaskPriorityLow:    {},
	TaskPriorityMedium: {},
	TaskPriorityHigh:   {},
}

// AllowedTaskPriorityValuesMessage is returned in API validation errors for invalid priority values.
func AllowedTaskPriorityValuesMessage() string {
	return "must be one of: low, medium, high"
}

// ParseTaskPriority normalizes and validates s.
func ParseTaskPriority(s string) (TaskPriority, error) {
	p := TaskPriority(strings.TrimSpace(strings.ToLower(s)))
	if _, ok := allowedTaskPriorities[p]; !ok {
		return "", fmt.Errorf("invalid task priority: %q", s)
	}
	return p, nil
}

// MarshalJSON emits a JSON string for TaskPriority.
func (p TaskPriority) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(p))
}

// UnmarshalJSON parses a JSON string into TaskPriority.
func (p *TaskPriority) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if strings.TrimSpace(raw) == "" {
		*p = ""
		return nil
	}
	parsed, err := ParseTaskPriority(raw)
	if err != nil {
		return NewValidationError(FieldMessage("priority", AllowedTaskPriorityValuesMessage()))
	}
	*p = parsed
	return nil
}

// Scan implements sql.Scanner.
func (p *TaskPriority) Scan(src interface{}) error {
	if src == nil {
		*p = ""
		return nil
	}
	var str string
	switch v := src.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return fmt.Errorf("cannot scan %T into TaskPriority", src)
	}
	parsed, err := ParseTaskPriority(str)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}

// Value implements driver.Valuer.
func (p TaskPriority) Value() (driver.Value, error) {
	if p == "" {
		return nil, nil
	}
	if _, ok := allowedTaskPriorities[p]; !ok {
		return nil, fmt.Errorf("invalid task priority: %q", p)
	}
	return string(p), nil
}

// UnmarshalParam binds query/form parameters into TaskPriority.
func (p *TaskPriority) UnmarshalParam(param string) error {
	if param == "" {
		*p = ""
		return nil
	}
	parsed, err := ParseTaskPriority(param)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}

// Task maps to the tasks table.
type Task struct {
	ID          uuid.UUID    `db:"id" json:"id"`
	Title       string       `db:"title" json:"title"`
	Description *string      `db:"description" json:"description,omitempty"`
	Status      TaskStatus   `db:"status" json:"status"`
	Priority    TaskPriority `db:"priority" json:"priority"`
	ProjectID   uuid.UUID    `db:"project_id" json:"project_id"`
	AssigneeID  *uuid.UUID   `db:"assignee_id" json:"assignee_id,omitempty"`
	CreatedBy   *uuid.UUID   `db:"created_by" json:"created_by,omitempty"`
	DueDate     *time.Time   `db:"due_date" json:"due_date,omitempty"`
	CreatedAt   time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time    `db:"updated_at" json:"updated_at"`
}

// CreateTaskRequest is the body for creating a task.
// If Status is omitted (empty), default to TaskStatusTodo in the application layer (DB column is NOT NULL).
type CreateTaskRequest struct {
	Title       string       `json:"title" validate:"required,min=1,max=255"`
	Description *string      `json:"description,omitempty" validate:"omitempty,max=10000"`
	Status      TaskStatus   `json:"status,omitempty" validate:"omitempty,oneof=todo in_progress done"`
	Priority    TaskPriority `json:"priority,omitempty" validate:"omitempty,oneof=low medium high"`
	ProjectID   uuid.UUID    `json:"project_id" validate:"required"`
	AssigneeID  *uuid.UUID   `json:"assignee_id,omitempty"`
	DueDate     *time.Time   `json:"due_date,omitempty"`
}

// UpdateTaskRequest is for partial task updates; only non-nil fields apply.
type UpdateTaskRequest struct {
	Title       *string       `json:"title,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string       `json:"description,omitempty" validate:"omitempty,max=10000"`
	Status      *TaskStatus   `json:"status,omitempty" validate:"omitempty,oneof=todo in_progress done"`
	Priority    *TaskPriority `json:"priority,omitempty" validate:"omitempty,oneof=low medium high"`
	AssigneeID  *uuid.UUID    `json:"assignee_id,omitempty"`
	DueDate     *time.Time    `json:"due_date,omitempty"`
}

// TaskFilters captures optional list/query parameters (e.g. Gin query binding).
// TaskStatus and TaskPriority support UnmarshalParam for form/query decoding.
type TaskFilters struct {
	ProjectID  *uuid.UUID    `form:"project_id"`
	Status     *TaskStatus   `form:"status"`
	Priority   *TaskPriority `form:"priority"`
	AssigneeID *uuid.UUID    `form:"assignee_id"`
	Page       *int          `form:"page"`
	Limit      *int          `form:"limit"`
}
