package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/proteancarp/cu-vote/internal/election"
)

// ErrElectionNotFound is the repository contract for a missing election,
// independent of any storage driver.
var ErrElectionNotFound = errors.New("election not found")

// ElectionReader retrieves validated metadata, not a mutable Election aggregate.
// Get returns ErrElectionNotFound for an absent ID.
type ElectionReader interface {
	Get(context.Context, string) (GetElectionResult, error)
}

// GetElectionResult is a metadata snapshot with no lifecycle or configuration
// operations. State records the stored value, not ballot-definition readiness.
type GetElectionResult struct {
	ID       string
	Title    string
	OpensAt  time.Time
	ClosesAt time.Time
	State    election.State
}

// GetElection retrieves election metadata using a caller-supplied repository.
type GetElection struct {
	repository ElectionReader
}

func NewGetElection(repository ElectionReader) (*GetElection, error) {
	if repository == nil {
		return nil, fmt.Errorf("%w: election repository", ErrMissingDependency)
	}
	return &GetElection{repository: repository}, nil
}

func (uc *GetElection) Execute(ctx context.Context, id string) (GetElectionResult, error) {
	metadata, err := uc.repository.Get(ctx, id)
	if err != nil {
		return GetElectionResult{}, fmt.Errorf("get election %q: %w", id, err)
	}
	return metadata, nil
}
