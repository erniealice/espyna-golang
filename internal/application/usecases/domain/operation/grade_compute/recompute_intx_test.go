package grade_compute

import (
	"context"
	"errors"
	"testing"
)

// fakePhaseRecomputer / fakeJobRecomputer let recomputeSheet be tested without a DB.
type fakePhaseRecomputer struct {
	calls []string
	ret   map[string]struct {
		ok  bool
		err error
	}
}

func (f *fakePhaseRecomputer) RecomputePhaseInAmbientTx(_ context.Context, id string) (bool, error) {
	f.calls = append(f.calls, id)
	r := f.ret[id]
	return r.ok, r.err
}

type fakeJobRecomputer struct {
	calls []string
	ret   map[string]struct {
		ok  bool
		err error
	}
}

func (f *fakeJobRecomputer) RecomputeJobInAmbientTx(_ context.Context, id string) (bool, error) {
	f.calls = append(f.calls, id)
	r := f.ret[id]
	return r.ok, r.err
}

// TestRecomputeSheet_FailurePropagates proves that a genuine recompute failure on any
// phase surfaces as an error — which the submit adapter returns, rolling the
// transition transaction back (FIX-3: "Failure of recompute ⇒ transition fails").
func TestRecomputeSheet_FailurePropagates(t *testing.T) {
	boom := errors.New("db read failed")
	phase := &fakePhaseRecomputer{ret: map[string]struct {
		ok  bool
		err error
	}{
		"p1": {ok: true},
		"p2": {err: boom}, // genuine failure
	}}
	job := &fakeJobRecomputer{}
	err := recomputeSheet(context.Background(), phase, job, []string{"p1", "p2", "p3"}, []string{"j1"})
	if err == nil {
		t.Fatal("expected recomputeSheet to fail when a phase recompute errors")
	}
	if !errors.Is(err, boom) {
		t.Fatalf("expected wrapped boom, got %v", err)
	}
	// It must stop at the failing phase and never reach the job leg.
	if len(job.calls) != 0 {
		t.Fatalf("job recompute must not run after a phase failure; got %v", job.calls)
	}
	if len(phase.calls) != 2 || phase.calls[1] != "p2" {
		t.Fatalf("expected to stop at p2, phase calls = %v", phase.calls)
	}
}

// TestRecomputeSheet_SkipsTolerated proves that expected skips (false,nil) — a
// non-gradable or blank phase, or a frozen/no-graded-phases job — do NOT fail the
// transition (D6 permits partial/blank submission).
func TestRecomputeSheet_SkipsTolerated(t *testing.T) {
	phase := &fakePhaseRecomputer{ret: map[string]struct {
		ok  bool
		err error
	}{
		"p1": {ok: false}, // skip (not gradable / blank)
		"p2": {ok: true},  // finalized
	}}
	job := &fakeJobRecomputer{ret: map[string]struct {
		ok  bool
		err error
	}{
		"j1": {ok: false}, // skip (no graded phases / frozen)
	}}
	if err := recomputeSheet(context.Background(), phase, job, []string{"p1", "p2", ""}, []string{"j1", ""}); err != nil {
		t.Fatalf("skips must be tolerated, got %v", err)
	}
	if len(phase.calls) != 2 { // the "" id is skipped without a call
		t.Fatalf("expected 2 phase calls (empty id skipped), got %v", phase.calls)
	}
	if len(job.calls) != 1 {
		t.Fatalf("expected 1 job call (empty id skipped), got %v", job.calls)
	}
}

// TestRecomputeSheet_JobFailurePropagates proves a job-leg failure also fails the
// transition (after the phase leg succeeded).
func TestRecomputeSheet_JobFailurePropagates(t *testing.T) {
	boom := errors.New("job summary write failed")
	phase := &fakePhaseRecomputer{ret: map[string]struct {
		ok  bool
		err error
	}{"p1": {ok: true}}}
	job := &fakeJobRecomputer{ret: map[string]struct {
		ok  bool
		err error
	}{"j1": {err: boom}}}
	err := recomputeSheet(context.Background(), phase, job, []string{"p1"}, []string{"j1"})
	if !errors.Is(err, boom) {
		t.Fatalf("expected job failure to propagate, got %v", err)
	}
}

// TestSheetRecompute_UnwiredFailsClosed proves the port fails closed when the
// grade-compute use cases are not wired (a nil sub-aggregate).
func TestSheetRecompute_UnwiredFailsClosed(t *testing.T) {
	var uc *UseCases
	if err := uc.SheetRecompute(context.Background(), []string{"p1"}, []string{"j1"}); err == nil {
		t.Fatal("expected SheetRecompute to fail closed on a nil use-case aggregate")
	}
	uc2 := &UseCases{} // ComputePhaseOutcome/ComputeJobOutcome nil
	if err := uc2.SheetRecompute(context.Background(), []string{"p1"}, nil); err == nil {
		t.Fatal("expected SheetRecompute to fail closed when the compute use cases are nil")
	}
}
