//go:build postgresql

package operation

import (
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/reviewedit"
)

// Plan 20261106-review-editor-override (design-v2) — AC-O3 (review_fill_ok /
// class_override_ok column shape) and AC-O12 (no new code => base SQL plus the
// two literal columns only, no extra bind).

const (
	fillFalseCol     = ", false AS review_fill_ok"
	fillTrueCol      = ", true AS review_fill_ok"
	overrideFalseCol = ", false AS class_override_ok"
)

// AC-O12: denied / zero / flag-off decisions emit the two literal false columns
// and nothing else changes (SQL equals base 2cfe57b5 + review_edit_ok false).
func TestReviewFill_DeniedSQLUnchanged(t *testing.T) {
	for _, c := range []struct {
		name string
		d    reviewedit.Decision
	}{
		{"zero", reviewedit.Decision{}},
		{"workspace_scope_without_flags", reviewedit.Decision{WorkspaceScope: true}},
		{"allowed_only_ws", reviewedit.Decision{Allowed: true, WorkspaceScope: true}},
	} {
		t.Run(c.name, func(t *testing.T) {
			ctx := reviewedit.WithDecision(shapeStaffCtx(c.d.WorkspaceScope), c.d)
			got, args := captureMatrixSQL(t, ctx, scopeMine)
			if !strings.Contains(got, fillFalseCol+overrideFalseCol+" FROM job j") {
				t.Fatalf("must end with the two literal false columns:\n%s", got)
			}
			if strings.Contains(got, "rr.role") && !c.d.Allowed {
				t.Errorf("no Fill/Allowed => no reviewer walk")
			}
			if !c.d.Allowed {
				base := baseMatrixStaffSQL
				if c.d.WorkspaceScope {
					base = baseMatrixWideSQL
				}
				if strings.Replace(stripNewCols(got), reviewEditFalseCol, "", 1) != base {
					t.Errorf("SQL drifted from base beyond the literal columns:\n%s", got)
				}
			}
			if c.d.Allowed && args != baseMatrixWideArgs {
				t.Errorf("binds changed: %s", args)
			}
		})
	}
}

// AC-O3: Fill + WorkspaceScope => literal true, no extra bind, no reviewer walk
// (and no owner term).
func TestOutcomeMatrixReviewFillColumnShape_WorkspaceScope(t *testing.T) {
	ctx := reviewedit.WithDecision(shapeStaffCtx(true), reviewedit.Decision{Fill: true, WorkspaceScope: true})
	got, args := captureMatrixSQL(t, ctx, scopeMine)
	if !strings.Contains(got, fillTrueCol+overrideFalseCol) {
		t.Errorf("workspace-scope Fill must project `true AS review_fill_ok`:\n%s", got)
	}
	if args != baseMatrixWideArgs || strings.Contains(got, "rr.role") {
		t.Errorf("workspace-scope Fill adds no bind / walk: %s", args)
	}
}

// AC-O3: Fill, reviewer edge => reviewerEdgeOwnedSQL on the LAST bind (acting
// staff), workspace $2, and nothing about cell owners.
func TestOutcomeMatrixReviewFillColumnShape(t *testing.T) {
	ctx := reviewedit.WithDecision(shapeStaffCtx(false), reviewedit.Decision{Fill: true})
	got, args := captureMatrixSQL(t, ctx, scopeMine)
	binds := strings.Split(args, "|")
	if len(binds) != len(strings.Split(baseMatrixStaffArgs, "|"))+1 || binds[len(binds)-1] != "staff-shape" {
		t.Fatalf("exactly one trailing staff bind expected: %v", binds)
	}
	want := reviewEditNorm(reviewerEdgeOwnedSQL(len(binds), 2))
	if !strings.Contains(got, want+" AS review_fill_ok") {
		t.Errorf("review_fill_ok must be the reviewer EXISTS (n=%d, ws=$2):\n%s", len(binds), got)
	}
	if !strings.Contains(got, overrideFalseCol+" FROM job j") || strings.Contains(got, fillFalseCol) {
		t.Errorf("fill on, override off:\n%s", got)
	}
	if strings.Contains(got, "ORDER BY") && strings.Contains(got, "LIMIT 1") || strings.Contains(got, "assigned_to <>") {
		t.Errorf("no owner-exists term allowed in review_fill_ok")
	}
}

// Fill + Allowed share ONE reviewer bind (no duplicate staff bind).
func TestOutcomeMatrixReviewFillSharesReviewerBind(t *testing.T) {
	ctx := reviewedit.WithDecision(shapeStaffCtx(false), reviewedit.Decision{Allowed: true, Fill: true})
	got, args := captureMatrixSQL(t, ctx, scopeMine)
	if n := len(strings.Split(args, "|")); n != len(strings.Split(baseMatrixStaffArgs, "|"))+1 {
		t.Errorf("Allowed+Fill must add one shared staff bind, got %d binds", n)
	}
	if strings.Count(got, "rr.role = 'reviewer'") != 2 {
		t.Errorf("expected the reviewer walk in both columns:\n%s", got)
	}
}

// design-v2: ClassOverride => primary class edge on the class-edge bind (reused).
func TestOutcomeMatrixClassOverrideColumnShape(t *testing.T) {
	ctx := reviewedit.WithDecision(shapeStaffCtx(false), reviewedit.Decision{ClassOverride: true})
	got, args := captureMatrixSQL(t, ctx, scopeMine)
	base := strings.Split(baseMatrixStaffArgs, "|")
	if len(strings.Split(args, "|")) != len(base) {
		t.Errorf("class_override_ok must reuse the class-edge bind, args=%s", args)
	}
	// class-edge bind is the 5th parameter in the base staff query
	n := len(base)
	want := reviewEditNorm(classPrimaryEdgeOwnedSQL(n, 2))
	if !strings.Contains(got, want+" AS class_override_ok") {
		t.Errorf("class_override_ok must be classPrimaryEdgeOwnedSQL(%d,2):\n%s", n, got)
	}
	if !strings.Contains(got, fillFalseCol) {
		t.Errorf("fill column must stay literal false")
	}
}

// Non-staff / operator: flags can never open anything (no acting staff => literals).
func TestOutcomeMatrixOverrideNonStaffFailsClosed(t *testing.T) {
	ctx := reviewedit.WithDecision(shapeOperatorCtx(), reviewedit.Decision{Fill: true, ClassOverride: true})
	got, args := captureMatrixSQL(t, ctx, scopeAll)
	if !strings.Contains(got, fillFalseCol+overrideFalseCol+" FROM job j") || args != baseMatrixOperatorArgs {
		t.Errorf("operator must fail closed with literals and no binds:\n%s\n%s", got, args)
	}
}
