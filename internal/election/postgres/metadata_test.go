package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/proteancarp/cu-vote/internal/election"
	"github.com/proteancarp/cu-vote/internal/election/application"
)

// metadataRow exercises the same decoding boundary used by Get, including
// corrupt values prohibited by the real schema, without database infrastructure.
type metadataRow struct {
	id, title, state  string
	opensAt, closesAt time.Time
	err               error
}

func (r metadataRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*dest[0].(*string) = r.id
	*dest[1].(*string) = r.title
	*dest[2].(*time.Time) = r.opensAt
	*dest[3].(*time.Time) = r.closesAt
	*dest[4].(*string) = r.state
	return nil
}

func validMetadataRow() metadataRow {
	return metadataRow{
		id: " election-1 ", title: " Election ", state: "DRAFT",
		opensAt: time.Unix(0, 0), closesAt: time.Unix(3600, 0),
	}
}

func TestReadElectionMetadataStates(t *testing.T) {
	for _, state := range []election.State{election.Draft, election.Frozen, election.Open, election.Closed, election.Finalized} {
		t.Run(string(state), func(t *testing.T) {
			row := validMetadataRow()
			row.state = string(state)
			got, err := readElectionMetadata(row.id, row)
			want := application.GetElectionResult{
				ID: row.id, Title: row.title, OpensAt: row.opensAt, ClosesAt: row.closesAt, State: state,
			}
			if err != nil || got != want {
				t.Fatalf("readElectionMetadata = %+v, %v; want %+v, nil", got, err, want)
			}
		})
	}
}

func TestReadElectionMetadataRejectsCorruption(t *testing.T) {
	tests := []struct {
		name   string
		change func(*metadataRow)
		want   error
	}{
		{"empty ID", func(r *metadataRow) { r.id = "" }, election.ErrInvalidID},
		{"blank ID", func(r *metadataRow) { r.id = " \t\u2003" }, election.ErrInvalidID},
		{"empty title", func(r *metadataRow) { r.title = "" }, election.ErrInvalidTitle},
		{"blank title", func(r *metadataRow) { r.title = " \n\u2003" }, election.ErrInvalidTitle},
		{"equal times", func(r *metadataRow) { r.closesAt = r.opensAt }, election.ErrInvalidTimeRange},
		{"reversed times", func(r *metadataRow) { r.closesAt = r.opensAt.Add(-time.Hour) }, election.ErrInvalidTimeRange},
		{"unknown state", func(r *metadataRow) { r.state = "UNKNOWN" }, election.ErrInvalidState},
		{"empty state", func(r *metadataRow) { r.state = "" }, election.ErrInvalidState},
		{"lowercase state", func(r *metadataRow) { r.state = "draft" }, election.ErrInvalidState},
		{"padded state", func(r *metadataRow) { r.state = " DRAFT " }, election.ErrInvalidState},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := validMetadataRow()
			tt.change(&row)
			got, err := readElectionMetadata("requested-id", row)
			if got != (application.GetElectionResult{}) || !errors.Is(err, tt.want) {
				t.Fatalf("readElectionMetadata = %+v, %v; want empty metadata, %v", got, err, tt.want)
			}
			if !strings.Contains(err.Error(), `invalid persisted election "requested-id"`) {
				t.Fatalf("missing validation context: %v", err)
			}
		})
	}
}

func TestReadElectionMetadataErrors(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "42P01", Message: "undefined table"}
	for _, cause := range []error{pgx.ErrNoRows, pgErr, context.Canceled, context.DeadlineExceeded, errors.New("scan failed")} {
		t.Run(cause.Error(), func(t *testing.T) {
			got, err := readElectionMetadata("requested-id", metadataRow{err: fmt.Errorf("driver: %w", cause)})
			want := cause
			if errors.Is(cause, pgx.ErrNoRows) {
				want = application.ErrElectionNotFound
				if errors.Is(err, pgx.ErrNoRows) {
					t.Fatalf("driver missing-row error escaped: %v", err)
				}
			}
			if got != (application.GetElectionResult{}) || !errors.Is(err, want) {
				t.Fatalf("readElectionMetadata = %+v, %v; want empty metadata, %v", got, err, want)
			}
			if !strings.Contains(err.Error(), `election "requested-id"`) {
				t.Fatalf("missing query context: %v", err)
			}
			if cause == pgErr {
				var gotPgErr *pgconn.PgError
				if !errors.As(err, &gotPgErr) || gotPgErr != pgErr {
					t.Fatalf("PostgreSQL error details lost: %v", err)
				}
			}
		})
	}
}
