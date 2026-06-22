package grade_compute

import (
	"sort"
	"testing"

	taskoutcomepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

func fp(v float64) *float64 { return &v }

func outcome(critID string, val *float64, active bool) *taskoutcomepb.TaskOutcome {
	return &taskoutcomepb.TaskOutcome{
		CriteriaVersionId: critID,
		NumericValue:      val,
		Active:            active,
	}
}

// TestBucketByCriterion locks the load-bearing modeling decision: a
// task_outcome is keyed to a criterion by CriteriaVersionId (FK to
// outcome_criteria.id), its numeric value comes from NumericValue, only
// in-scope + active + numeric outcomes contribute, and the within-criterion
// list is the set of all recorded values (gradecompute takes the MAX).
func TestBucketByCriterion(t *testing.T) {
	inScope := map[string]bool{"A": true, "B": true, "C": true, "D": true}

	outcomes := []*taskoutcomepb.TaskOutcome{
		outcome("A", fp(6), true),
		outcome("A", fp(8), true), // MAX 8 within A
		outcome("A", fp(7), true),
		outcome("B", fp(7), true),
		outcome("B", fp(5), true), // MAX 7 within B
		outcome("C", fp(6), true), // MAX 6
		outcome("D", fp(7), true), // MAX 7
		// Skipped: out-of-scope criterion.
		outcome("Z", fp(9), true),
		// Skipped: inactive outcome.
		outcome("A", fp(99), false),
		// Skipped: non-numeric outcome (text/categorical) -> nil NumericValue.
		outcome("B", nil, true),
	}

	inputs, contributing := bucketByCriterion(outcomes, inScope)

	if contributing != 4 {
		t.Fatalf("contributing got %d want 4 (A,B,C,D)", contributing)
	}

	got := make(map[string][]float64)
	for _, in := range inputs {
		vs := append([]float64(nil), in.Values...)
		sort.Float64s(vs)
		got[in.CriterionID] = vs
	}

	// A keeps all three recorded values; the inactive 99 must be excluded.
	if a := got["A"]; len(a) != 3 || a[2] != 8 {
		t.Fatalf("criterion A values got %v want [6 7 8]", a)
	}
	// B drops the nil-numeric outcome -> only the two numeric values.
	if b := got["B"]; len(b) != 2 || b[1] != 7 {
		t.Fatalf("criterion B values got %v want [5 7]", b)
	}
	// Out-of-scope Z never appears.
	if _, ok := got["Z"]; ok {
		t.Fatalf("out-of-scope criterion Z leaked into inputs: %v", got)
	}
}

// TestBucketByCriterion_EmptyCriterionContributesNothing verifies a scoped
// criterion with no recorded numeric values yields an empty Values slice and is
// not counted as contributing (so RollUpCriteria treats it as absent, not 0).
func TestBucketByCriterion_EmptyCriterionContributesNothing(t *testing.T) {
	inScope := map[string]bool{"A": true, "B": true}
	outcomes := []*taskoutcomepb.TaskOutcome{outcome("A", fp(5), true)}

	inputs, contributing := bucketByCriterion(outcomes, inScope)
	if contributing != 1 {
		t.Fatalf("contributing got %d want 1", contributing)
	}
	if len(inputs) != 2 {
		t.Fatalf("inputs got %d want 2 (one per scoped criterion)", len(inputs))
	}
	for _, in := range inputs {
		if in.CriterionID == "B" && len(in.Values) != 0 {
			t.Fatalf("empty criterion B should have no values, got %v", in.Values)
		}
	}
}
