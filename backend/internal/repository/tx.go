package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// WithTx runs fn with repository implementations bound to a single SQL transaction.
// If fn returns an error, the transaction is rolled back. Panics propagate after rollback.
func WithTx(ctx context.Context, db *sqlx.DB, fn func(*Repositories) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	repos := &Repositories{
		Users:    NewUserRepository(tx),
		Projects: NewProjectRepository(tx),
		Tasks:    NewTaskRepository(tx),
	}

	if err := fn(repos); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback: %v (after: %w)", rbErr, err)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
