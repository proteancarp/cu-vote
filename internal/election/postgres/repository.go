// Package postgres implements PostgreSQL persistence for the election module.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/proteancarp/cu-vote/internal/election"
	"github.com/proteancarp/cu-vote/internal/election/application"
)

// ErrUnsupportedElection prevents partial persistence of configuration that
// this repository does not yet support.
var ErrUnsupportedElection = errors.New("only unconfigured draft elections can be created")

type Repository struct {
	pool *pgxpool.Pool
}

var (
	_ application.ElectionCreator = (*Repository)(nil)
	_ application.ElectionReader  = (*Repository)(nil)
)

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

// Get retrieves and validates persisted metadata. Missing rows use the
// application repository contract; other database and domain errors are wrapped.
func (r *Repository) Get(ctx context.Context, id string) (application.GetElectionResult, error) {
	if err := ctx.Err(); err != nil {
		return application.GetElectionResult{}, fmt.Errorf("get election %q: %w", id, err)
	}
	row := r.pool.QueryRow(ctx, `
		SELECT id, title, opens_at, closes_at, state
		FROM elections WHERE id = $1`, id)
	return readElectionMetadata(id, row)
}

// readElectionMetadata decodes and validates a metadata projection without
// reconstructing an aggregate whose ballot definition has not been loaded.
func readElectionMetadata(id string, row pgx.Row) (application.GetElectionResult, error) {
	var metadata application.GetElectionResult
	var state string
	err := row.Scan(&metadata.ID, &metadata.Title, &metadata.OpensAt, &metadata.ClosesAt, &state)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.GetElectionResult{}, fmt.Errorf("get election %q: %w", id, application.ErrElectionNotFound)
	}
	if err != nil {
		return application.GetElectionResult{}, fmt.Errorf("query election %q: %w", id, err)
	}
	if err := election.ValidateMetadata(metadata.ID, metadata.Title, metadata.OpensAt, metadata.ClosesAt); err != nil {
		return application.GetElectionResult{}, fmt.Errorf("invalid persisted election %q: %w", id, err)
	}
	metadata.State, err = election.ParseState(state)
	if err != nil {
		return application.GetElectionResult{}, fmt.Errorf("invalid persisted election %q: %w", id, err)
	}
	return metadata, nil
}
