package application_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/proteancarp/cu-vote/internal/election"
	"github.com/proteancarp/cu-vote/internal/election/application"
)

// This fake implements retrieval only: Get Election must not require Create.
type stubReader struct {
	metadata application.GetElectionResult
	err      error
	getCalls int
	id       string
	ctx      context.Context
}

func (s *stubReader) Get(ctx context.Context, id string) (application.GetElectionResult, error) {
	s.getCalls++
	s.ctx = ctx
	s.id = id
	return s.metadata, s.err
}

func TestGetElectionSuccess(t *testing.T) {
	command := validCommand()
	for _, state := range []election.State{election.Draft, election.Frozen, election.Open, election.Closed, election.Finalized} {
		t.Run(string(state), func(t *testing.T) {
			want := application.GetElectionResult{
				ID: " election-42 ", Title: command.Title, OpensAt: command.OpensAt, ClosesAt: command.ClosesAt, State: state,
			}
			repository := &stubReader{metadata: want}
			uc, err := application.NewGetElection(repository)
			if err != nil {
				t.Fatal(err)
			}
			type contextKey struct{}
			ctx, cancel := context.WithCancel(context.WithValue(t.Context(), contextKey{}, "caller"))
			defer cancel()
			result, err := uc.Execute(ctx, want.ID)
			if err != nil {
				t.Fatal(err)
			}
			if result != want {
				t.Fatalf("result = %+v, want %+v", result, want)
			}
			if repository.getCalls != 1 || repository.id != want.ID || repository.ctx != ctx {
				t.Fatal("Get did not forward the ID/context exactly once")
			}
			cancel()
			if !errors.Is(repository.ctx.Err(), context.Canceled) {
				t.Fatal("caller cancellation did not reach repository context")
			}
			result.Title = "Changed"
			if repository.metadata != want {
				t.Fatal("result modification changed repository metadata")
			}
		})
	}
}

func TestGetElectionFailures(t *testing.T) {
	repositoryErr := errors.New("repository unavailable")
	for _, wantErr := range []error{
		application.ErrElectionNotFound, repositoryErr, context.Canceled, context.DeadlineExceeded,
		election.ErrInvalidID, election.ErrInvalidTitle, election.ErrInvalidTimeRange, election.ErrInvalidState,
	} {
		t.Run(wantErr.Error(), func(t *testing.T) {
			repository := &stubReader{err: fmt.Errorf("repository failure: %w", wantErr)}
			uc, err := application.NewGetElection(repository)
			if err != nil {
				t.Fatal(err)
			}
			result, err := uc.Execute(t.Context(), "e-1")
			if !errors.Is(err, wantErr) || result != (application.GetElectionResult{}) {
				t.Fatalf("Execute = %+v, %v; want empty result, %v", result, err, wantErr)
			}
			if repository.getCalls != 1 {
				t.Fatal("unexpected repository calls")
			}
			if !strings.Contains(err.Error(), `get election "e-1"`) {
				t.Fatalf("missing use-case context: %v", err)
			}
		})
	}
}

func TestNewGetElectionRequiresRepository(t *testing.T) {
	uc, err := application.NewGetElection(nil)
	if uc != nil || !errors.Is(err, application.ErrMissingDependency) {
		t.Fatalf("NewGetElection = %v, %v; want nil, ErrMissingDependency", uc, err)
	}
}

func TestGetElectionResultHasNoAggregateOperations(t *testing.T) {
	// Metadata with a stored non-Draft state must never expose mutation methods
	// from an embedded or reconstructed, incomplete Election aggregate.
	resultType := reflect.TypeFor[*application.GetElectionResult]()
	for _, name := range []string{"Freeze", "Open", "Close", "Finalize", "AddContest", "AddChoice"} {
		if _, exists := resultType.MethodByName(name); exists {
			t.Errorf("metadata exposes aggregate operation %s", name)
		}
	}
}
