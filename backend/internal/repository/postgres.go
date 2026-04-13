package repository

import "github.com/jmoiron/sqlx"

// Postgres holds a shared database handle for repository methods.
type Postgres struct {
	DB *sqlx.DB
}

// NewPostgres returns a repository backed by sqlx.
func NewPostgres(db *sqlx.DB) *Postgres {
	return &Postgres{DB: db}
}
