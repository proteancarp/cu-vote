package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/proteancarp/cu-vote/internal/election"
	"github.com/proteancarp/cu-vote/internal/election/postgres"
)

func TestNewRequiresPool(t *testing.T) {
	if repository, err := postgres.New(nil); repository != nil || err == nil {
		t.Fatalf("New(nil) = %v, %v; want nil and error", repository, err)
	}
}

func TestCreateRejectsUnsupportedElection(t *testing.T) {
	// A lazy pool requires no database; rejected inputs must never reach I/O.
	pool, err := pgxpool.New(t.Context(), "postgres://localhost/cuvote?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repository, err := postgres.New(pool)
	if err != nil {
		t.Fatal(err)
	}
	e, err := election.New("configured", "Election", time.Unix(0, 0), time.Unix(3600, 0))
	if err != nil {
		t.Fatal(err)
	}
	if err := e.AddContest("proposal", "Proposal"); err != nil {
		t.Fatal(err)
	}
	configured := *e
	if err := e.AddChoice("proposal", "yes", "Yes"); err != nil {
		t.Fatal(err)
	}
	if err := e.Freeze(); err != nil {
		t.Fatal(err)
	}
	for _, candidate := range []election.Election{{}, configured, *e} {
		if err := repository.Create(t.Context(), candidate); !errors.Is(err, postgres.ErrUnsupportedElection) {
			t.Fatalf("Create error = %v, want ErrUnsupportedElection", err)
		}
	}
}

func TestCreatePreservesCanceledContextWithoutDatabase(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgres://localhost/cuvote?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repository, err := postgres.New(pool)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := repository.Create(ctx, election.Election{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create error = %v, want context.Canceled", err)
	}
}
