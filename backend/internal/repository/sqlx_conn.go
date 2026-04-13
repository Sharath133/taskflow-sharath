package repository

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// sqlxConn is implemented by *sqlx.DB and *sqlx.Tx so repositories can run inside a transaction.
// QueryRowContext is listed explicitly because sqlx.ExtContext alone does not expose it on a named
// interface embedding (driver sub-interfaces differ from database/sql.QueryerContext).
type sqlxConn interface {
	sqlx.ExtContext
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error)
}
