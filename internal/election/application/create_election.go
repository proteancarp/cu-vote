// Package application orchestrates election use cases using the election domain.
package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/proteancarp/cu-vote/internal/election"
)

// ErrMissingDependency indicates an absent required use-case dependency.
var ErrMissingDependency = errors.New("missing create election dependency")

// IDGenerator supplies identifiers without coupling the domain to an ID scheme.
type IDGenerator interface {
	NewID() (string, error)
}

// ElectionRepository persists newly created elections. Create must reject an
// existing election ID rather than overwrite the stored election.
type ElectionRepository interface {
	Create(context.Context, election.Election) error
}

// CreateElectionCommand supplies only the initial election definition.
type CreateElectionCommand struct {
	Title    string
	OpensAt  time.Time
	ClosesAt time.Time
}

// CreateElectionResult identifies the election successfully persisted by Execute.
type CreateElectionResult struct {
	ID string
}

// CreateElection coordinates ID generation, domain construction, and persistence.
// Construct it with NewCreateElection and non-nil dependency implementations.
type CreateElection struct {
	ids        IDGenerator
	repository ElectionRepository
}

func NewCreateElection(ids IDGenerator, repository ElectionRepository) (*CreateElection, error) {
	if ids == nil {
		return nil, fmt.Errorf("%w: ID generator", ErrMissingDependency)
	}
	if repository == nil {
		return nil, fmt.Errorf("%w: election repository", ErrMissingDependency)
	}
	return &CreateElection{ids: ids, repository: repository}, nil
}

// Execute returns a result only after persistence succeeds. Domain validation is
// delegated to election.New; no persistence is attempted if an earlier step fails.
func (uc *CreateElection) Execute(ctx context.Context, command CreateElectionCommand) (CreateElectionResult, error) {
	id, err := uc.ids.NewID()
	if err != nil {
		return CreateElectionResult{}, fmt.Errorf("generate election ID: %w", err)
	}
	e, err := election.New(id, command.Title, command.OpensAt, command.ClosesAt)
	if err != nil {
		return CreateElectionResult{}, fmt.Errorf("construct election: %w", err)
	}
	if err := uc.repository.Create(ctx, *e); err != nil {
		return CreateElectionResult{}, fmt.Errorf("persist new election: %w", err)
	}
	return CreateElectionResult{ID: e.ID()}, nil
}
