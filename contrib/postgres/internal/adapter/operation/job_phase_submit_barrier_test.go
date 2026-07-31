//go:build postgresql

package operation

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
)

// These are the NO-DB unit tests for the FIX-3 submit freshness barrier
// (submitFreshnessBarrier — the exact helper SubmitJobPhaseApproval invokes
// between the D6 blank count and the status flip).

// TestSubmitFreshnessBarrier_FailureFailsSubmit proves a recompute failure fails
// the submit: the barrier wraps and propagates the error, which Submit returns —
// rolling the transition transaction back.
func TestSubmitFreshnessBarrier_FailureFailsSubmit(t *testing.T) {
	boom := errors.New("phase summary write failed")
	locked := []lockedPhase{{id: "p1", jobID: "j1"}, {id: "p2", jobID: "j2"}}
	err := submitFreshnessBarrier(context.Background(), "submit",
		func(_ context.Context, _, _ []string) error { return boom }, locked)
	if err == nil {
		t.Fatal("a recompute failure MUST fail the submit")
	}
	if !errors.Is(err, boom) {
		t.Fatalf("the recompute error must be wrapped for the caller, got %v", err)
	}
	if !strings.Contains(err.Error(), "freshness-barrier recompute failed") {
		t.Fatalf("error must name the freshness barrier, got %v", err)
	}
}

// TestSubmitFreshnessBarrier_NilPortFailsClosed proves an unwired barrier refuses
// the submit outright (a raw/custom repository cannot transition without it).
func TestSubmitFreshnessBarrier_NilPortFailsClosed(t *testing.T) {
	err := submitFreshnessBarrier(context.Background(), "submit", nil, []lockedPhase{{id: "p1", jobID: "j1"}})
	if err == nil || !strings.Contains(err.Error(), "not wired") {
		t.Fatalf("nil recompute port must fail closed, got %v", err)
	}
}

// TestSubmitFreshnessBarrier_PassesFullLockedSet proves the barrier hands the
// recompute the FULL locked phase set plus the deduped owning-job set — the
// recompute finalizes the whole sheet, not a staff-scoped subset.
func TestSubmitFreshnessBarrier_PassesFullLockedSet(t *testing.T) {
	locked := []lockedPhase{
		{id: "p1", jobID: "jA"},
		{id: "p2", jobID: "jB"},
		{id: "p3", jobID: "jA"}, // duplicate job
		{id: "p4", jobID: ""},   // empty job id excluded from the job leg
	}
	var gotPhases, gotJobs []string
	err := submitFreshnessBarrier(context.Background(), "verify",
		func(_ context.Context, phaseIDs, jobIDs []string) error {
			gotPhases = append([]string(nil), phaseIDs...)
			gotJobs = append([]string(nil), jobIDs...)
			return nil
		}, locked)
	if err != nil {
		t.Fatalf("successful recompute must not error, got %v", err)
	}
	sort.Strings(gotPhases)
	if len(gotPhases) != 4 || gotPhases[0] != "p1" || gotPhases[3] != "p4" {
		t.Fatalf("expected all 4 locked phases, got %v", gotPhases)
	}
	sort.Strings(gotJobs)
	if len(gotJobs) != 2 || gotJobs[0] != "jA" || gotJobs[1] != "jB" {
		t.Fatalf("expected deduped jobs [jA jB], got %v", gotJobs)
	}
}
