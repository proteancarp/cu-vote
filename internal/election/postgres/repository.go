// Package postgres implements PostgreSQL persistence for the election module.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/proteancarp/cu-vote/internal/election"
	"github.com/proteancarp/cu-vote/internal/election/application"
)

// ErrUnsupportedElection prevents partial persistence of configuration that
// this create-only repository does not yet support.
var ErrUnsupportedElection = errors.New("only unconfigured draft elections can be created")

type Repository struct {
	pool *pgxpool.Pool
}

var _ application.ElectionRepository = (*Repository)(nil)

// New uses a caller-owned pool. The repository does not close it.
func New(pool *pgxpool.Pool) (*Repository, error) {
	if pool == nil {
		return nil, errors.New("election repository requires a PostgreSQL pool")
	}
	return &Repository{pool: pool}, nil
}

// Create inserts the initial election definition without overwriting existing
// data. Database errors, including duplicate IDs, remain available via errors.As.
// A single INSERT provides the necessary atomicity without an explicit transaction.
func (r *Repository) Create(ctx context.Context, e election.Election) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("create election %q: %w", e.ID(), err)
	}
	if e.State() != election.Draft || len(e.Contests()) != 0 {
		return fmt.Errorf("create election %q: %w", e.ID(), ErrUnsupportedElection)
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO elections (id, title, opens_at, closes_at, state)
		VALUES ($1, $2, $3, $4, $5)`,
		e.ID(), e.Title(), e.OpensAt(), e.ClosesAt(), string(e.State()))
	if err != nil {
		return fmt.Errorf("insert election %q: %w", e.ID(), err)
	}
	return nil
}
