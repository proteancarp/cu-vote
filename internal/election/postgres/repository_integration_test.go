package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/proteancarp/cu-vote/internal/election"
	"github.com/proteancarp/cu-vote/internal/election/application"
	"github.com/proteancarp/cu-vote/internal/election/postgres"
	"github.com/proteancarp/cu-vote/internal/platform/database"
)

type fixedID string

func (id fixedID) NewID() (string, error) { return string(id), nil }

func TestRepositoryIntegration(t *testing.T) {
	pool := integrationPool(t)
	repository, err := postgres.New(pool)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("create through application and reject duplicate", func(t *testing.T) {
		uc, err := application.NewCreateElection(fixedID(" election-1 "), repository)
		if err != nil {
			t.Fatal(err)
		}
		opensAt := time.Date(2030, time.January, 2, 9, 0, 0, 123456000, time.FixedZone("offset", 3600))
		command := application.CreateElectionCommand{
			Title: " Proposal Election ", OpensAt: opensAt, ClosesAt: opensAt.Add(time.Hour),
		}
		result, err := uc.Execute(t.Context(), command)
		if err != nil {
			t.Fatal(err)
		}
		if result.ID != " election-1 " {
			t.Fatalf("result ID = %q", result.ID)
		}
		assertPersistedElection(t, pool, result.ID, command)
		get, err := application.NewGetElection(repository)
		if err != nil {
			t.Fatal(err)
		}
		retrieved, err := get.Execute(t.Context(), result.ID)
		if err != nil {
			t.Fatal(err)
		}
		if retrieved.ID != result.ID || retrieved.Title != command.Title || retrieved.State != election.Draft ||
			!retrieved.OpensAt.Equal(command.OpensAt) || !retrieved.ClosesAt.Equal(command.ClosesAt) {
			t.Fatalf("retrieved definition does not match stored election: %+v", retrieved)
		}
		// Change every non-ID field, verifying a duplicate never overwrites them.
		changed := application.CreateElectionCommand{
			Title: "Replacement", OpensAt: opensAt.Add(24 * time.Hour), ClosesAt: opensAt.Add(25 * time.Hour),
		}
		_, err = uc.Execute(t.Context(), changed)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
			t.Fatalf("duplicate error = %v, want PostgreSQL unique violation", err)
		}
		assertPersistedElection(t, pool, result.ID, command)
	})
	t.Run("all lifecycle states as metadata", func(t *testing.T) {
		for _, state := range []election.State{election.Draft, election.Frozen, election.Open, election.Closed, election.Finalized} {
			id := "metadata-" + string(state)
			_, err := pool.Exec(t.Context(), `INSERT INTO elections (id, title, opens_at, closes_at, state)
				VALUES ($1, $2, $3, $4, $5)`, id, " Metadata ", time.Unix(0, 0), time.Unix(3600, 0), string(state))
			if err != nil {
				t.Fatal(err)
			}
			metadata, err := repository.Get(t.Context(), id)
			if err != nil || metadata.ID != id || metadata.Title != " Metadata " || metadata.State != state {
				t.Fatalf("Get = %+v, %v; want metadata for %s", metadata, err, state)
			}
		}
	})
	t.Run("canceled caller context", func(t *testing.T) {
		e, err := election.New("canceled", "Election", time.Unix(0, 0), time.Unix(3600, 0))
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		if err := repository.Create(ctx, *e); !errors.Is(err, context.Canceled) {
			t.Fatalf("Create error = %v, want context.Canceled", err)
		}
		if _, err := repository.Get(ctx, " election-1 "); !errors.Is(err, context.Canceled) {
			t.Fatalf("Get error = %v, want context.Canceled", err)
		}
		var count int
		if err := pool.QueryRow(t.Context(), "SELECT count(*) FROM elections WHERE id = $1", e.ID()).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatal("canceled creation persisted an election")
		}
	})
	t.Run("missing election", func(t *testing.T) {
		get, err := application.NewGetElection(repository)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := get.Execute(t.Context(), "missing"); !errors.Is(err, application.ErrElectionNotFound) || errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("Get error = %v, want ErrElectionNotFound without pgx.ErrNoRows", err)
		}
	})
	t.Run("invalid persisted title", func(t *testing.T) {
		// The existing SQL schema permits blank titles; metadata validation must reject them.
		_, err := pool.Exec(t.Context(), `INSERT INTO elections (id, title, opens_at, closes_at, state)
			VALUES ($1, $2, $3, $4, $5)`, "corrupt", " \t\u2003", time.Unix(0, 0), time.Unix(3600, 0), "DRAFT")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := repository.Get(t.Context(), "corrupt"); !errors.Is(err, election.ErrInvalidTitle) {
			t.Fatalf("Get error = %v, want ErrInvalidTitle", err)
		}
	})
	t.Run("migration rollback and reapply", func(t *testing.T) {
		applyMigration(t, pool, "000001_create_elections.down.sql")
		var table *string
		if err := pool.QueryRow(t.Context(), "SELECT to_regclass('elections')::text").Scan(&table); err != nil {
			t.Fatal(err)
		}
		if table != nil {
			t.Fatal("down migration left elections table present")
		}
		metadata, err := repository.Get(t.Context(), "missing-table")
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "42P01" || metadata != (application.GetElectionResult{}) {
			t.Fatalf("Get = %+v, %v; want empty metadata and undefined-table error", metadata, err)
		}
		applyMigration(t, pool, "000001_create_elections.up.sql")
	})
}

func assertPersistedElection(t *testing.T, pool *pgxpool.Pool, id string, command application.CreateElectionCommand) {
	t.Helper()
	var storedID, title, state string
	var opensAt, closesAt time.Time
	err := pool.QueryRow(t.Context(), `SELECT id, title, opens_at, closes_at, state FROM elections WHERE id = $1`, id).
		Scan(&storedID, &title, &opensAt, &closesAt, &state)
	if err != nil {
		t.Fatal(err)
	}
	if storedID != id || title != command.Title || state != string(election.Draft) {
		t.Fatalf("stored ID/title/state = %q/%q/%q", storedID, title, state)
	}
	if !opensAt.Equal(command.OpensAt) || !closesAt.Equal(command.ClosesAt) {
		t.Fatalf("stored schedule = %v to %v, want %v to %v", opensAt, closesAt, command.OpensAt, command.ClosesAt)
	}
}

func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	t.Cleanup(cancel)
	admin, err := database.Open(ctx, url)
	if err != nil {
		t.Fatalf("open test PostgreSQL: %v", err)
	}
	t.Cleanup(admin.Close)
	schema := fmt.Sprintf("cuvote_test_%d", time.Now().UnixNano())
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, stop := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer stop()
		if _, err := admin.Exec(cleanupCtx, "DROP SCHEMA "+quotedSchema+" CASCADE"); err != nil {
			t.Errorf("remove test schema: %v", err)
		}
	})
	poolConfig, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	applyMigration(t, pool, "000001_create_elections.up.sql")
	return pool
}

func applyMigration(t *testing.T, pool *pgxpool.Pool, name string) {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", name))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), string(contents)); err != nil {
		t.Fatalf("apply migration %s: %v", name, err)
	}
}
