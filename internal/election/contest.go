package election

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

var (
	ErrConfigurationLocked = errors.New("election configuration is locked")
	ErrNotReadyToFreeze    = errors.New("election is not ready to freeze")
	ErrInvalidContestID    = errors.New("contest ID must not be blank")
	ErrInvalidContestTitle = errors.New("contest title must not be blank")
	ErrDuplicateContestID  = errors.New("duplicate contest ID")
	ErrContestNotFound     = errors.New("contest not found")
	ErrInvalidChoiceID     = errors.New("choice ID must not be blank")
	ErrInvalidChoiceLabel  = errors.New("choice label must not be blank")
	ErrDuplicateChoiceID   = errors.New("duplicate choice ID")
)

// Contest is a question or position with available choices. Only Election
// methods can change its definition.
type Contest struct {
	id      string
	title   string
	choices []Choice
}

func (c Contest) ID() string    { return c.id }
func (c Contest) Title() string { return c.title }

// Choices returns a snapshot in insertion order.
func (c Contest) Choices() []Choice { return slices.Clone(c.choices) }

// Choice is an election option, which need not represent a candidate.
type Choice struct {
	id    string
	label string
}

func (c Choice) ID() string    { return c.id }
func (c Choice) Label() string { return c.label }

// Contests returns a snapshot, including choices, in insertion order.
func (e *Election) Contests() []Contest {
	contests := slices.Clone(e.contests)
	for i := range contests {
		contests[i].choices = slices.Clone(contests[i].choices)
	}
	return contests
}

// AddContest adds an empty contest to a draft election, preserving its inputs.
// IDs are compared exactly and must be unique within this election.
func (e *Election) AddContest(id, title string) error {
	if !e.CanConfigure() {
		return fmt.Errorf("%w: state %q", ErrConfigurationLocked, e.state)
	}
	if strings.TrimSpace(id) == "" {
		return ErrInvalidContestID
	}
	if strings.TrimSpace(title) == "" {
		return ErrInvalidContestTitle
	}
	for _, contest := range e.contests {
		if contest.id == id {
			return fmt.Errorf("%w: %q", ErrDuplicateContestID, id)
		}
	}
	// Copy before writing so separately copied Election values remain independent.
	e.contests = append(slices.Clone(e.contests), Contest{id: id, title: title})
	return nil
}

// AddChoice adds an option to an existing contest in a draft election.
// IDs must be unique within the target contest. Accepted inputs are preserved.
func (e *Election) AddChoice(contestID, id, label string) error {
	if !e.CanConfigure() {
		return fmt.Errorf("%w: state %q", ErrConfigurationLocked, e.state)
	}
	if strings.TrimSpace(id) == "" {
		return ErrInvalidChoiceID
	}
	if strings.TrimSpace(label) == "" {
		return ErrInvalidChoiceLabel
	}
	for i, contest := range e.contests {
		if contest.id != contestID {
			continue
		}
		for _, choice := range contest.choices {
			if choice.id == id {
				return fmt.Errorf("%w: contest %q, choice %q", ErrDuplicateChoiceID, contestID, id)
			}
		}
		contests := slices.Clone(e.contests)
		contests[i].choices = append(slices.Clone(contest.choices), Choice{id: id, label: label})
		e.contests = contests
		return nil
	}
	return fmt.Errorf("%w: %q", ErrContestNotFound, contestID)
}
