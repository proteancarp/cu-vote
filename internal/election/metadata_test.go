package election_test

import (
	"errors"
	"testing"
	"time"

	"github.com/proteancarp/cu-vote/internal/election"
)

func TestValidateMetadata(t *testing.T) {
	tests := []struct {
		name, id, title string
		closesAt        time.Time
		want            error
	}{
		{"valid preserved whitespace", " election-1 ", " Election ", openingTime.Add(time.Hour), nil},
		{"empty ID", "", "Election", openingTime.Add(time.Hour), election.ErrInvalidID},
		{"blank ID", " \t\u2003", "Election", openingTime.Add(time.Hour), election.ErrInvalidID},
		{"empty title", "e-1", "", openingTime.Add(time.Hour), election.ErrInvalidTitle},
		{"blank title", "e-1", " \n\u2003", openingTime.Add(time.Hour), election.ErrInvalidTitle},
		{"equal times", "e-1", "Election", openingTime, election.ErrInvalidTimeRange},
		{"reversed times", "e-1", "Election", openingTime.Add(-time.Hour), election.ErrInvalidTimeRange},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := election.ValidateMetadata(tt.id, tt.title, openingTime, tt.closesAt); !errors.Is(err, tt.want) {
				t.Fatalf("ValidateMetadata = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestParseState(t *testing.T) {
	for _, want := range []election.State{election.Draft, election.Frozen, election.Open, election.Closed, election.Finalized} {
		t.Run(string(want), func(t *testing.T) {
			got, err := election.ParseState(string(want))
			if err != nil || got != want {
				t.Fatalf("ParseState = %q, %v; want %q, nil", got, err, want)
			}
		})
	}
	for _, value := range []string{"", "UNKNOWN", "draft", " DRAFT ", " \t\u2003"} {
		t.Run("invalid/"+value, func(t *testing.T) {
			got, err := election.ParseState(value)
			if got != "" || !errors.Is(err, election.ErrInvalidState) {
				t.Fatalf("ParseState = %q, %v; want empty state, ErrInvalidState", got, err)
			}
		})
	}
}
