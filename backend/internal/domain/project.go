package domain

import (
	"time"

	"github.com/google/uuid"
)

// Project maps to the projects table.
type Project struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description *string   `db:"description" json:"description,omitempty"`
	OwnerID     uuid.UUID `db:"owner_id" json:"owner_id"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// ProjectWithTasks is a project aggregate including its tasks (e.g. detail views).
type ProjectWithTasks struct {
	Project
	TaskCount int64  `json:"task_count"`
	Tasks     []Task `json:"tasks"`
}

// ProjectTaskStats is returned by GET /projects/:id/stats.
type ProjectTaskStats struct {
	ByStatus   map[string]int64 `json:"by_status"`
	ByAssignee map[string]int64 `json:"by_assignee"`
}

// CreateProjectRequest is the body for creating a project.
type CreateProjectRequest struct {
	Name        string  `json:"name" validate:"required,min=1,max=255"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=10000"`
}

// UpdateProjectRequest is the body for partial project updates; only non-nil fields apply.
type UpdateProjectRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=10000"`
}
