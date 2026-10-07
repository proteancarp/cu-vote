// Package database manages PostgreSQL connection pools.
package database

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrMissingURL prevents an absent configured URL from selecting pgx defaults.
var ErrMissingURL = errors.New("database URL is required")

// Open establishes and verifies a pool using the supplied connection URL and
// context. Callers own the returned pool and must call Close when finished.
// Pass config.Load().DatabaseURL at the application's configuration boundary.
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("open PostgreSQL pool: %w", err)
	}
	if strings.TrimSpace(url) == "" {
		return nil, ErrMissingURL
	}
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("configure PostgreSQL pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	return pool, nil
}
