package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/proteancarp/cu-vote/internal/election"
	"github.com/proteancarp/cu-vote/internal/election/application"
)

type stubIDs struct {
	id    string
	err   error
	calls int
}

func (s *stubIDs) NewID() (string, error) {
	s.calls++
	return s.id, s.err
}

type stubRepository struct {
	err      error
	calls    int
	ctx      context.Context
	election election.Election
}

func (s *stubRepository) Create(ctx context.Context, e election.Election) error {
	s.calls++
	s.ctx = ctx
	s.election = e
	return s.err
}

func validCommand() application.CreateElectionCommand {
	opensAt := time.Date(2030, time.January, 2, 9, 0, 0, 0, time.FixedZone("offset", 3600))
	return application.CreateElectionCommand{
		Title: " Committee Election ", OpensAt: opensAt, ClosesAt: opensAt.Add(time.Hour),
	}
}

func TestCreateElectionSuccess(t *testing.T) {
	ids := &stubIDs{id: " election-42 "}
	repository := &stubRepository{}
	uc, err := application.NewCreateElection(ids, repository)
	if err != nil {
		t.Fatal(err)
	}
	type contextKey struct{}
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), contextKey{}, "caller"))
	defer cancel()
	command := validCommand()
	result, err := uc.Execute(ctx, command)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if ids.calls != 1 || repository.calls != 1 {
		t.Fatalf("ID/repository calls = %d/%d, want 1/1", ids.calls, repository.calls)
	}
	e := repository.election
	if result.ID != ids.id || e.ID() != ids.id {
		t.Fatalf("result/domain ID = %q/%q, want %q", result.ID, e.ID(), ids.id)
	}
	if e.Title() != command.Title || e.OpensAt() != command.OpensAt || e.ClosesAt() != command.ClosesAt {
		t.Fatal("persisted election did not preserve the command title and schedule")
	}
	if e.State() != election.Draft || !e.CanConfigure() || len(e.Contests()) != 0 {
		t.Fatal("persisted election must be an unconfigured draft")
	}
	if repository.ctx != ctx || repository.ctx.Value(contextKey{}) != "caller" {
		t.Fatal("repository did not receive the caller context")
	}
	cancel()
	if !errors.Is(repository.ctx.Err(), context.Canceled) {
		t.Fatal("caller cancellation did not reach repository context")
	}
}

func TestCreateElectionFailures(t *testing.T) {
	idErr := errors.New("ID source unavailable")
	repositoryErr := errors.New("repository unavailable")
	command := validCommand()
	tests := []struct {
		name            string
		id              string
		idErr           error
		command         application.CreateElectionCommand
		repositoryErr   error
		wantErr         error
		wantCreateCalls int
	}{
		{"ID generation", "ignored", idErr, command, nil, idErr, 0},
		{"empty generated ID", "", nil, command, nil, election.ErrInvalidID, 0},
		{"blank generated ID", " \t\u2003", nil, command, nil, election.ErrInvalidID, 0},
		{"empty title", "e-1", nil, application.CreateElectionCommand{OpensAt: command.OpensAt, ClosesAt: command.ClosesAt}, nil, election.ErrInvalidTitle, 0},
		{"blank title", "e-1", nil, application.CreateElectionCommand{Title: " \t\u2003", OpensAt: command.OpensAt, ClosesAt: command.ClosesAt}, nil, election.ErrInvalidTitle, 0},
		{"equal times", "e-1", nil, application.CreateElectionCommand{Title: command.Title, OpensAt: command.OpensAt, ClosesAt: command.OpensAt}, nil, election.ErrInvalidTimeRange, 0},
		{"reversed times", "e-1", nil, application.CreateElectionCommand{Title: command.Title, OpensAt: command.ClosesAt, ClosesAt: command.OpensAt}, nil, election.ErrInvalidTimeRange, 0},
		{"repository", "e-1", nil, command, repositoryErr, repositoryErr, 1},
		{"repository cancellation", "e-1", nil, command, context.Canceled, context.Canceled, 1},
		{"repository deadline", "e-1", nil, command, context.DeadlineExceeded, context.DeadlineExceeded, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids := &stubIDs{id: tt.id, err: tt.idErr}
			repository := &stubRepository{err: tt.repositoryErr}
			uc, err := application.NewCreateElection(ids, repository)
			if err != nil {
				t.Fatal(err)
			}
			result, err := uc.Execute(context.Background(), tt.command)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if result != (application.CreateElectionResult{}) {
				t.Fatalf("failed creation returned a success result: %+v", result)
			}
			if ids.calls != 1 || repository.calls != tt.wantCreateCalls {
				t.Fatalf("ID/repository calls = %d/%d, want 1/%d", ids.calls, repository.calls, tt.wantCreateCalls)
			}
		})
	}
}

func TestNewCreateElectionRequiresDependencies(t *testing.T) {
	tests := []struct {
		name       string
		ids        application.IDGenerator
		repository application.ElectionRepository
	}{
		{"missing ID generator", nil, &stubRepository{}},
		{"missing repository", &stubIDs{}, nil},
		{"both missing", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, err := application.NewCreateElection(tt.ids, tt.repository)
			if !errors.Is(err, application.ErrMissingDependency) || uc != nil {
				t.Fatalf("constructor = %v, %v; want nil, ErrMissingDependency", uc, err)
			}
		})
	}
}
