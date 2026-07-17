package consumer

import (
	"context"
	"testing"
)

// The narrow inline-recompute adapters (W2 grade-sheet edit mode) must return a
// non-nil closure even on a nil container, and that closure must fail SAFE —
// (false, err), never a panic — so the fayna record action degrades a saved cell
// to ratingFresh:false rather than 500ing. (The recomputed/frozen classification
// paths need a fully-wired container and are covered at the use-case layer:
// grade_compute.TestUpsertJobSummary_RefusesToOverwriteAuthoritative asserts the
// ErrSummaryFrozen wrap this adapter's errors.Is depends on.)

func TestComputePhaseOutcomeAdapter_NilContainerFailsSafe(t *testing.T) {
	fn := NewComputePhaseOutcomeAdapter(nil)
	if fn == nil {
		t.Fatal("adapter factory returned a nil closure")
	}
	recomputed, err := fn(context.Background(), "jp-1")
	if err == nil {
		t.Error("nil container must surface an error (fail-safe), got nil")
	}
	if recomputed {
		t.Error("nil container must report recomputed=false")
	}
}

func TestComputeJobOutcomeAdapter_NilContainerFailsSafe(t *testing.T) {
	fn := NewComputeJobOutcomeAdapter(nil)
	if fn == nil {
		t.Fatal("adapter factory returned a nil closure")
	}
	recomputed, err := fn(context.Background(), "job-1")
	// A nil-container error is NOT the frozen sentinel, so it must propagate as a
	// genuine failure (false, err) — not be swallowed as a frozen skip.
	if err == nil {
		t.Error("nil container must surface an error (fail-safe), got nil")
	}
	if recomputed {
		t.Error("nil container must report recomputed=false")
	}
}
