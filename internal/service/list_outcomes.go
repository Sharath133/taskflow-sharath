package service

import "github.com/taskflow/backend/internal/domain"

// ProjectListResult is the outcome of listing projects (optionally paginated).
type ProjectListResult struct {
	Items     []domain.Project
	Total     int64
	Page      int
	Limit     int
	Paginated bool
}

// TaskListResult is the outcome of listing tasks (optionally paginated).
type TaskListResult struct {
	Items     []domain.Task
	Total     int64
	Page      int
	Limit     int
	Paginated bool
}
