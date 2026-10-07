package election_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/proteancarp/cu-vote/internal/election"
)

func newElection(t *testing.T) *election.Election {
	t.Helper()
	e, err := election.New("election-1", "Committee Election", openingTime, openingTime.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func configureElection(t *testing.T, e *election.Election) {
	t.Helper()
	if err := e.AddContest("proposal", "Approve proposal?"); err != nil {
		t.Fatal(err)
	}
	if err := e.AddChoice("proposal", "yes", "Yes"); err != nil {
		t.Fatal(err)
	}
}

func TestAddContestsAndChoices(t *testing.T) {
	e := newElection(t)
	for _, id := range []string{" proposal ", "other"} {
		if err := e.AddContest(id, " Approve proposal? "); err != nil {
			t.Fatal(err)
		}
		// Choice IDs may be reused in another contest; labels need not be unique.
		for _, choiceID := range []string{" yes ", "no", "alternate"} {
			if err := e.AddChoice(id, choiceID, " Option "); err != nil {
				t.Fatal(err)
			}
		}
	}
	contests := e.Contests()
	if len(contests) != 2 {
		t.Fatalf("got %d contests, want 2", len(contests))
	}
	for i, id := range []string{" proposal ", "other"} {
		if contests[i].ID() != id || contests[i].Title() != " Approve proposal? " {
			t.Fatalf("contest %d did not preserve its ID/title", i)
		}
		choices := contests[i].Choices()
		if len(choices) != 3 {
			t.Fatalf("contest %q has %d choices, want 3", id, len(choices))
		}
		for j, choiceID := range []string{" yes ", "no", "alternate"} {
			if choices[j].ID() != choiceID || choices[j].Label() != " Option " {
				t.Fatalf("choice %d did not preserve its ID/label", j)
			}
		}
	}
	if e.State() != election.Draft || !e.CanConfigure() {
		t.Fatal("configuration changed the election lifecycle")
	}
	// Contest uniqueness is election-scoped.
	other := newElection(t)
	if err := other.AddContest(" proposal ", "Another proposal"); err != nil {
		t.Fatalf("contest ID reused across elections: %v", err)
	}
}

func TestConfigurationValidation(t *testing.T) {
	tests := []struct {
		name string
		run  func(*election.Election) error
		want error
	}{
		{"empty contest ID", func(e *election.Election) error { return e.AddContest("", "Title") }, election.ErrInvalidContestID},
		{"blank contest ID", func(e *election.Election) error { return e.AddContest(" \t\u2003", "Title") }, election.ErrInvalidContestID},
		{"empty contest title", func(e *election.Election) error { return e.AddContest("new", "") }, election.ErrInvalidContestTitle},
		{"blank contest title", func(e *election.Election) error { return e.AddContest("new", " \n\u2003") }, election.ErrInvalidContestTitle},
		{"duplicate contest", func(e *election.Election) error { return e.AddContest("proposal", "Changed title") }, election.ErrDuplicateContestID},
		{"empty choice ID", func(e *election.Election) error { return e.AddChoice("proposal", "", "Label") }, election.ErrInvalidChoiceID},
		{"blank choice ID", func(e *election.Election) error { return e.AddChoice("proposal", " \t\u2003", "Label") }, election.ErrInvalidChoiceID},
		{"empty choice label", func(e *election.Election) error { return e.AddChoice("proposal", "new", "") }, election.ErrInvalidChoiceLabel},
		{"blank choice label", func(e *election.Election) error { return e.AddChoice("proposal", "new", " \n\u2003") }, election.ErrInvalidChoiceLabel},
		{"duplicate choice", func(e *election.Election) error { return e.AddChoice("proposal", "yes", "Changed label") }, election.ErrDuplicateChoiceID},
		{"unknown contest", func(e *election.Election) error { return e.AddChoice("missing", "new", "Label") }, election.ErrContestNotFound},
		{"blank target contest", func(e *election.Election) error { return e.AddChoice(" ", "new", "Label") }, election.ErrContestNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newElection(t)
			configureElection(t, e)
			before := e.Contests()
			if err := tt.run(e); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
			if !reflect.DeepEqual(before, e.Contests()) || e.State() != election.Draft {
				t.Fatal("rejected operation changed election configuration or state")
			}
		})
	}
}

func TestConfigurationLocked(t *testing.T) {
	transitions := []func(*election.Election) error{
		(*election.Election).Freeze, (*election.Election).Open,
		(*election.Election).Close, (*election.Election).Finalize,
	}
	for i, state := range []election.State{election.Frozen, election.Open, election.Closed, election.Finalized} {
		t.Run(string(state), func(t *testing.T) {
			e := newElection(t)
			configureElection(t, e)
			for _, transition := range transitions[:i+1] {
				if err := transition(e); err != nil {
					t.Fatal(err)
				}
			}
			before := e.Contests()
			for _, operation := range []func() error{
				func() error { return e.AddContest("new", "New contest") },
				func() error { return e.AddChoice("proposal", "no", "No") },
				// Locked state takes precedence over input validation.
				func() error { return e.AddContest("", "") },
				func() error { return e.AddChoice("missing", "", "") },
			} {
				if err := operation(); !errors.Is(err, election.ErrConfigurationLocked) {
					t.Fatalf("error = %v, want ErrConfigurationLocked", err)
				}
			}
			if e.State() != state || e.CanConfigure() || !reflect.DeepEqual(before, e.Contests()) {
				t.Fatal("locked configuration or lifecycle changed")
			}
		})
	}
}

func TestFreezeReadiness(t *testing.T) {
	tests := []struct {
		name    string
		missing int // -1 means no contests; otherwise the index of an empty contest.
	}{
		{"no contests", -1}, {"first contest empty", 0}, {"later contest empty", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newElection(t)
			ids := []string{"first", "second"}
			if tt.missing >= 0 {
				for i, id := range ids {
					if err := e.AddContest(id, "Proposal"); err != nil {
						t.Fatal(err)
					}
					if i != tt.missing {
						if err := e.AddChoice(id, "yes", "Yes"); err != nil {
							t.Fatal(err)
						}
					}
				}
			}
			before := e.Contests()
			err := e.Freeze()
			if !errors.Is(err, election.ErrNotReadyToFreeze) {
				t.Fatalf("Freeze error = %v, want ErrNotReadyToFreeze", err)
			}
			if tt.missing >= 0 && !strings.Contains(err.Error(), ids[tt.missing]) {
				t.Fatalf("readiness error lacks empty contest ID: %v", err)
			}
			if e.State() != election.Draft || !e.CanConfigure() || !reflect.DeepEqual(before, e.Contests()) {
				t.Fatal("failed freeze changed configuration or lifecycle")
			}
			// Complete the definition and retry. A single choice is sufficient.
			if tt.missing == -1 {
				configureElection(t, e)
			} else if err := e.AddChoice(ids[tt.missing], "yes", "Yes"); err != nil {
				t.Fatal(err)
			}
			if err := e.Freeze(); err != nil {
				t.Fatalf("Freeze after completing configuration: %v", err)
			}
			if e.State() != election.Frozen || e.CanConfigure() {
				t.Fatal("successful freeze did not lock configuration")
			}
		})
	}
}

func TestConfigurationSnapshots(t *testing.T) {
	for _, freeze := range []bool{false, true} {
		name := "draft"
		if freeze {
			name = "frozen"
		}
		t.Run(name, func(t *testing.T) {
			e := newElection(t)
			configureElection(t, e)
			before := e.Contests()
			retained := e.Contests()[0]
			if freeze {
				if err := e.Freeze(); err != nil {
					t.Fatal(err)
				}
			}
			contests := e.Contests()
			choices := contests[0].Choices()
			choices[0] = election.Choice{}
			choices = append(choices, election.Choice{})
			contests[0] = election.Contest{}
			contests = append(contests, election.Contest{})
			retainedChoices := retained.Choices()
			retainedChoices[0] = election.Choice{}
			if !reflect.DeepEqual(before, e.Contests()) {
				t.Fatal("modifying returned data changed election configuration")
			}
			if retained.Choices()[0].ID() != "yes" {
				t.Fatal("modifying returned choices changed the contest snapshot")
			}
			if !freeze {
				if err := e.AddChoice("proposal", "no", "No"); err != nil {
					t.Fatal(err)
				}
				if len(retained.Choices()) != 1 {
					t.Fatal("later configuration changed an earlier snapshot")
				}
			}
		})
	}
}

func TestElectionCopyCannotChangeFrozenConfiguration(t *testing.T) {
	e := newElection(t)
	configureElection(t, e)
	draftCopy := *e
	if err := e.Freeze(); err != nil {
		t.Fatal(err)
	}
	before := e.Contests()
	if err := draftCopy.AddChoice("proposal", "no", "No"); err != nil {
		t.Fatal(err)
	}
	if err := draftCopy.AddContest("other", "Other proposal"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, e.Contests()) || e.State() != election.Frozen {
		t.Fatal("configuring a draft copy changed the frozen election")
	}
}

func TestZeroValueConfiguration(t *testing.T) {
	var e election.Election
	if err := e.AddContest("proposal", "Proposal"); !errors.Is(err, election.ErrConfigurationLocked) {
		t.Fatalf("AddContest error = %v, want ErrConfigurationLocked", err)
	}
	if err := e.AddChoice("proposal", "yes", "Yes"); !errors.Is(err, election.ErrConfigurationLocked) {
		t.Fatalf("AddChoice error = %v, want ErrConfigurationLocked", err)
	}
	if len(e.Contests()) != 0 || e.CanConfigure() {
		t.Fatal("zero-value election became configurable")
	}
	var contest election.Contest
	var choice election.Choice
	if contest.ID() != "" || contest.Title() != "" || len(contest.Choices()) != 0 || choice.ID() != "" || choice.Label() != "" {
		t.Fatal("unexpected zero-value contest or choice data")
	}
}
