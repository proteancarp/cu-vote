// Package election models election definitions and explicit lifecycle transitions.
package election

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidID         = errors.New("election ID must not be blank")
	ErrInvalidTitle      = errors.New("election title must not be blank")
	ErrInvalidTimeRange  = errors.New("election closing time must be after opening time")
	ErrInvalidTransition = errors.New("invalid election lifecycle transition")
)

// State identifies a stage of the election lifecycle.
type State string

const (
	Draft     State = "DRAFT"
	Frozen    State = "FROZEN"
	Open      State = "OPEN"
	Closed    State = "CLOSED"
	Finalized State = "FINALIZED"
)

// Election owns its definition and lifecycle. Use New to construct one;
// the zero value is uninitialized and cannot configure or transition.
// An Election requires exclusive access when its lifecycle is changed.
type Election struct {
	id       string
	title    string
	opensAt  time.Time
	closesAt time.Time
	state    State
}

// New creates a draft election without normalizing its inputs.
// The caller supplies an ID unique across elections; this package does not
// maintain a registry or generate identifiers. Times are schedule data only.
func New(id, title string, opensAt, closesAt time.Time) (*Election, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrInvalidID
	}
	if strings.TrimSpace(title) == "" {
		return nil, ErrInvalidTitle
	}
	if !closesAt.After(opensAt) {
		return nil, ErrInvalidTimeRange
	}
	return &Election{
		id: id, title: title, opensAt: opensAt, closesAt: closesAt, state: Draft,
	}, nil
}

func (e *Election) ID() string          { return e.id }
func (e *Election) Title() string       { return e.title }
func (e *Election) OpensAt() time.Time  { return e.opensAt }
func (e *Election) ClosesAt() time.Time { return e.closesAt }
func (e *Election) State() State        { return e.state }

// CanConfigure reports whether ballot-affecting configuration may change.
// Future configuration operations must enforce this boundary themselves.
func (e *Election) CanConfigure() bool { return e.state == Draft }

// Freeze makes the ballot definition immutable.
func (e *Election) Freeze() error { return e.transition(Draft, Frozen) }

// Open explicitly opens a frozen election, independent of its schedule.
func (e *Election) Open() error { return e.transition(Frozen, Open) }

// Close explicitly closes an open election, independent of its schedule.
func (e *Election) Close() error { return e.transition(Open, Closed) }

// Finalize marks a closed election as finalized. No further transitions exist.
func (e *Election) Finalize() error { return e.transition(Closed, Finalized) }

func (e *Election) transition(from, to State) error {
	if e.state != from {
		return fmt.Errorf("%w: %q -> %q", ErrInvalidTransition, e.state, to)
	}
	e.state = to
	return nil
}
