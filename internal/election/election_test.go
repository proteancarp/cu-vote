package election_test

import (
	"errors"
	"testing"
	"time"

	"github.com/proteancarp/cu-vote/internal/election"
)

var openingTime = time.Date(2030, time.January, 2, 9, 0, 0, 0, time.UTC)

func TestNew(t *testing.T) {
	// Surrounding whitespace is preserved, rather than silently normalized.
	id, title := " election-1 ", " Committee Election "
	closingTime := openingTime.Add(time.Hour)
	e, err := election.New(id, title, openingTime, closingTime)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	assertDefinition(t, e, id, title, openingTime, closingTime)
	if e.State() != election.Draft {
		t.Fatalf("state = %q, want Draft", e.State())
	}
	if !e.CanConfigure() {
		t.Fatal("draft election must permit configuration")
	}
}

func TestNewRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		title    string
		closesAt time.Time
		wantErr  error
	}{
		{"empty ID", "", "Election", openingTime.Add(time.Hour), election.ErrInvalidID},
		{"blank ID", " \t\n\u2003", "Election", openingTime.Add(time.Hour), election.ErrInvalidID},
		{"empty title", "election-1", "", openingTime.Add(time.Hour), election.ErrInvalidTitle},
		{"blank title", "election-1", " \t\n\u2003", openingTime.Add(time.Hour), election.ErrInvalidTitle},
		{"equal times", "election-1", "Election", openingTime, election.ErrInvalidTimeRange},
		{"closing before opening", "election-1", "Election", openingTime.Add(-time.Hour), election.ErrInvalidTimeRange},
		{"equal instants in different zones", "election-1", "Election", openingTime.In(time.FixedZone("offset", 3600)), election.ErrInvalidTimeRange},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := election.New(tt.id, tt.title, openingTime, tt.closesAt)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("New error = %v, want %v", err, tt.wantErr)
			}
			if e != nil {
				t.Fatal("invalid input returned an election")
			}
		})
	}
}

// Exercise every exposed action from every reachable state, including repeats,
// skipped stages, attempts to move backwards, and every action after Finalized.
func TestLifecycle(t *testing.T) {
	states := []election.State{election.Draft, election.Frozen, election.Open, election.Closed, election.Finalized}
	actions := []struct {
		name string
		from election.State
		to   election.State
		run  func(*election.Election) error
	}{
		{"freeze", election.Draft, election.Frozen, (*election.Election).Freeze},
		{"open", election.Frozen, election.Open, (*election.Election).Open},
		{"close", election.Open, election.Closed, (*election.Election).Close},
		{"finalize", election.Closed, election.Finalized, (*election.Election).Finalize},
	}
	for i, state := range states {
		for _, action := range actions {
			t.Run(string(state)+"/"+action.name, func(t *testing.T) {
				e, err := election.New("election-1", "Committee Election", openingTime, openingTime.Add(time.Hour))
				if err != nil {
					t.Fatal(err)
				}
				for _, setup := range actions[:i] {
					if err := setup.run(e); err != nil {
						t.Fatalf("setup %s: %v", setup.name, err)
					}
				}
				if e.State() != state || e.CanConfigure() != (state == election.Draft) {
					t.Fatalf("unexpected state or configuration permission before %s", action.name)
				}
				wantState := state
				err = action.run(e)
				if state == action.from {
					if err != nil {
						t.Fatalf("%s: %v", action.name, err)
					}
					wantState = action.to
				} else if !errors.Is(err, election.ErrInvalidTransition) {
					t.Fatalf("error = %v, want ErrInvalidTransition", err)
				}
				if e.State() != wantState {
					t.Fatalf("state = %q, want %q", e.State(), wantState)
				}
				if e.CanConfigure() != (wantState == election.Draft) {
					t.Fatalf("unexpected configuration permission in %s", wantState)
				}
				assertDefinition(t, e, "election-1", "Committee Election", openingTime, openingTime.Add(time.Hour))
			})
		}
	}
}

func TestZeroValueCannotConfigureOrTransition(t *testing.T) {
	var e election.Election
	if e.CanConfigure() {
		t.Fatal("uninitialized election must not permit configuration")
	}
	initialState := e.State()
	for _, action := range []func() error{e.Freeze, e.Open, e.Close, e.Finalize} {
		if err := action(); !errors.Is(err, election.ErrInvalidTransition) {
			t.Fatalf("error = %v, want ErrInvalidTransition", err)
		}
		if e.State() != initialState {
			t.Fatal("failed transition changed state")
		}
	}
}

func assertDefinition(t *testing.T, e *election.Election, id, title string, opensAt, closesAt time.Time) {
	t.Helper()
	if e.ID() != id || e.Title() != title {
		t.Fatalf("ID/title = %q/%q, want %q/%q", e.ID(), e.Title(), id, title)
	}
	if e.OpensAt() != opensAt || e.ClosesAt() != closesAt {
		t.Fatalf("schedule = %v to %v, want %v to %v", e.OpensAt(), e.ClosesAt(), opensAt, closesAt)
	}
}
