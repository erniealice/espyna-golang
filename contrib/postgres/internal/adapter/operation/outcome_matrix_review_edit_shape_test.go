//go:build postgresql

package operation

import (
	"context"
	"fmt"
	"strings"
	"testing"

	matrixpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/outcome_matrix"

	"github.com/erniealice/espyna-golang/internal/application/shared/reviewedit"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

// Plan 20261105-reviewer-edit-grades — AC-2 (matrix review_edit_ok column shape)
// and AC-12 (denied ⇒ matrix SQL identical to base commit 2cfe57b5 except the
// added `false AS review_edit_ok` column). Shares the stub driver and the base
// goldens with task_outcome_review_edit_shape_test.go.

const (
	reviewEditFalseCol = ", false AS review_edit_ok"
	reviewEditTrueCol  = ", true AS review_edit_ok"
)

func reviewEditNorm(s string) string { return strings.Join(strings.Fields(s), " ") }

var (
	scopeMine = matrixpb.OutcomeMatrixScope_OUTCOME_MATRIX_SCOPE_MINE
	scopeAll  = matrixpb.OutcomeMatrixScope_OUTCOME_MATRIX_SCOPE_ALL
)

// AC-12: no decision on the context ⇒ identical to base apart from the one literal column.
func TestReviewEdit_DeniedSQLUnchanged_Matrix(t *testing.T) {
	for _, c := range []struct {
		name     string
		ctx      context.Context
		scope    matrixpb.OutcomeMatrixScope
		baseSQL  string
		baseArgs string
	}{
		{"staff_mine", shapeStaffCtx(false), scopeMine, baseMatrixStaffSQL, baseMatrixStaffArgs},
		{"staff_workspace_wide", shapeStaffCtx(true), scopeMine, baseMatrixWideSQL, baseMatrixWideArgs},
		{"operator_all", shapeOperatorCtx(), scopeAll, baseMatrixOperatorSQL, baseMatrixOperatorArgs},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, args := captureMatrixSQL(t, c.ctx, c.scope)
			if !strings.Contains(got, reviewEditFalseCol) {
				t.Fatalf("denied matrix SQL must project the literal `false AS review_edit_ok`:\n%s", got)
			}
			if stripped := strings.Replace(got, reviewEditFalseCol, "", 1); stripped != c.baseSQL {
				t.Errorf("denied matrix SQL drifted from base 2cfe57b5 beyond the added column:\n got: %s\nwant: %s", stripped, c.baseSQL)
			}
			if args != c.baseArgs {
				t.Errorf("denied matrix bind values = %q, want base %q (no extra bind)", args, c.baseArgs)
			}
			if strings.Contains(got, "rr.role") && !strings.Contains(c.baseSQL, "rr.role") {
				t.Errorf("denied matrix SQL must not carry the reviewer EXISTS")
			}
		})
	}
}

// AC-12: an explicit zero / not-Allowed decision (even with WorkspaceScope set —
// it is only meaningful when Allowed) is the same as no decision.
func TestReviewEdit_DeniedSQLUnchanged_MatrixExplicitDenied(t *testing.T) {
	for name, d := range map[string]reviewedit.Decision{
		"zero":                        {},
		"workspace_scope_not_allowed": {Allowed: false, WorkspaceScope: true},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := reviewedit.WithDecision(shapeStaffCtx(false), d)
			got, args := captureMatrixSQL(t, ctx, scopeMine)
			if strings.Replace(got, reviewEditFalseCol, "", 1) != baseMatrixStaffSQL || args != baseMatrixStaffArgs {
				t.Errorf("explicit denied decision changed the matrix SQL or binds:\n%s\nargs=%s", got, args)
			}
		})
	}
}

// AC-2: Allowed (reviewer-edge holder) adds the reviewer EXISTS, bound to the
// acting staff as the LAST bind and to the trusted workspace $2.
func TestOutcomeMatrixReviewEditColumnShape(t *testing.T) {
	ctx := reviewedit.WithDecision(shapeStaffCtx(false), reviewedit.Decision{Allowed: true})
	got, args := captureMatrixSQL(t, ctx, scopeMine)

	binds := strings.Split(args, "|")
	if want := strings.Split(baseMatrixStaffArgs, "|"); len(binds) != len(want)+1 || binds[len(binds)-1] != "staff-shape" {
		t.Fatalf("allowed reviewer decision must add exactly one trailing staff bind: got %v (base %v)", binds, want)
	}
	wantExists := reviewEditNorm(reviewerEdgeOwnedSQL(len(binds), 2))
	if !strings.Contains(got, wantExists+" AS review_edit_ok") {
		t.Errorf("review_edit_ok must be the reviewer-edge EXISTS (reviewerEdgeOwnedSQL(%d, 2)) as the last column:\n%s", len(binds), got)
	}
	for _, sub := range []string{
		fmt.Sprintf("rr.staff_id = $%d AND rr.workspace_id = $2", len(binds)),
		"rc.workspace_id = $2",
		"rm.workspace_id = $2",
		"rr.role = 'reviewer'",
	} {
		if !strings.Contains(got, sub) {
			t.Errorf("allowed matrix SQL missing %q", sub)
		}
	}
	if strings.Contains(got, reviewEditFalseCol) || strings.Contains(got, reviewEditTrueCol) {
		t.Errorf("allowed reviewer-edge decision must not project a literal column")
	}
	// Everything before the new column is byte-identical to base (the column is last).
	idx := strings.Index(got, ", EXISTS (")
	if idx < 0 {
		t.Fatalf("no EXISTS column found")
	}
	if prefix := got[:idx]; !strings.Contains(baseMatrixStaffSQL, prefix) {
		t.Errorf("SQL before the review_edit_ok column drifted from base")
	}
}

// AC-2 / AC-16: Allowed + WorkspaceScope ⇒ literal true, no reviewer walk, no extra bind.
func TestOutcomeMatrixReviewEditColumnShape_WorkspaceScope(t *testing.T) {
	ctx := reviewedit.WithDecision(shapeStaffCtx(true), reviewedit.Decision{Allowed: true, WorkspaceScope: true})
	got, args := captureMatrixSQL(t, ctx, scopeMine)
	if !strings.Contains(got, reviewEditTrueCol) {
		t.Errorf("workspace-scope decision must project `true AS review_edit_ok`:\n%s", got)
	}
	if strings.Replace(got, reviewEditTrueCol, "", 1) != baseMatrixWideSQL || args != baseMatrixWideArgs {
		t.Errorf("workspace-scope decision must change nothing but the literal column (no extra bind)")
	}
}

// AC-2 / AC-16: fail closed — Allowed on a non-staff principal (no acting staff id)
// cannot reach the reviewer walk and without workspace scope projects false.
func TestOutcomeMatrixReviewEditColumnShape_NonStaffFailsClosed(t *testing.T) {
	ctx := reviewedit.WithDecision(shapeOperatorCtx(), reviewedit.Decision{Allowed: true})
	got, args := captureMatrixSQL(t, ctx, scopeAll)
	if !strings.Contains(got, reviewEditFalseCol) || args != baseMatrixOperatorArgs {
		t.Errorf("Allowed decision without an acting staff must stay `false` with no extra bind:\n%s\nargs=%s", got, args)
	}
}

// AC-2 / AC-5: outcomeInReviewScopeSQL — correlated on the outcome's job task,
// workspace bound to the given placeholder, reviewer walk REUSED (not copied).
func TestOutcomeInReviewScopeSQLShape(t *testing.T) {
	got := reviewEditNorm(outcomeInReviewScopeSQL("t.job_task_id", 4, 2))
	if !strings.HasPrefix(got, "EXISTS (") {
		t.Errorf("must be an EXISTS fragment: %s", got)
	}
	for _, sub := range []string{
		"FROM " + entityid.JobTask + " jt",
		"JOIN " + entityid.JobPhase + " jp ON jp.id = jt.job_phase_id AND jp.active",
		"JOIN " + entityid.Job + " j ON j.id = jp.job_id AND j.workspace_id = $2",
		"WHERE jt.id = t.job_task_id",
		reviewEditNorm(reviewerEdgeOwnedSQL(4, 2)),
	} {
		if !strings.Contains(got, sub) {
			t.Errorf("outcomeInReviewScopeSQL missing %q in:\n%s", sub, got)
		}
	}
	if n := strings.Count(got, "rr.role = 'reviewer'"); n != 1 {
		t.Errorf("reviewer walk must appear exactly once (reuse reviewerEdgeOwnedSQL), got %d", n)
	}
	// The expression and placeholders are parameters, not constants.
	other := reviewEditNorm(outcomeInReviewScopeSQL("to.job_task_id", 7, 3))
	if !strings.Contains(other, "jt.id = to.job_task_id") || !strings.Contains(other, "j.workspace_id = $3") || !strings.Contains(other, "rr.staff_id = $7 AND rr.workspace_id = $3") {
		t.Errorf("expression / placeholders not honoured: %s", other)
	}
	if strings.Contains(got, "$4 AND rr.workspace_id = $4") {
		t.Errorf("facet and workspace placeholders must stay distinct")
	}
}
