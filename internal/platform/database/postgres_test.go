package database_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/proteancarp/cu-vote/internal/platform/database"
)

func TestOpenRejectsMissingURL(t *testing.T) {
	for _, url := range []string{"", " \t\n"} {
		pool, err := database.Open(t.Context(), url)
		if pool != nil || !errors.Is(err, database.ErrMissingURL) {
			t.Fatalf("Open = %v, %v; want nil, ErrMissingURL", pool, err)
		}
	}
}

func TestOpenRejectsInvalidURL(t *testing.T) {
	pool, err := database.Open(t.Context(), "postgres://localhost:invalid/cuvote")
	if pool != nil || err == nil || errors.Unwrap(err) == nil {
		t.Fatalf("Open = %v, %v; want nil and wrapped parse error", pool, err)
	}
}

func TestOpenPreservesContextErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	expired, stop := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer stop()
	for _, tt := range []struct {
		ctx  context.Context
		want error
	}{{ctx, context.Canceled}, {expired, context.DeadlineExceeded}} {
		pool, err := database.Open(tt.ctx, "postgres://localhost/cuvote")
		if pool != nil || !errors.Is(err, tt.want) {
			t.Fatalf("Open = %v, %v; want nil, %v", pool, err, tt.want)
		}
	}
}
