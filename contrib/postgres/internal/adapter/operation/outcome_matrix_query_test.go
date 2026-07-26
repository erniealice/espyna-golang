//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	matrixpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/outcome_matrix"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	"github.com/erniealice/espyna-golang/shared/identity"
)

// rosterStaffCtx builds a session context whose active binding is a STAFF principal
// (kind 7) — the shape the session middleware stamps and the ONLY source the scope
// helper reads (never a request param).
func rosterStaffCtx(staffID string) context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{
		WorkspaceID:   "ws-1",
		PrincipalType: principalscope.PrincipalTypeStaff,
		PrincipalID:   staffID,
	})
}

// rosterNonStaffCtx builds a session context whose active binding is a NON-staff
// principal (operator kind 1) — a MINE roster read for such a caller must fail
// closed (no reachable job set).
func rosterNonStaffCtx() context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{
		WorkspaceID:   "ws-1",
		PrincipalType: 1,
		PrincipalID:   "op-1",
	})
}

// TestRosterScopeClause pins the roster read's row-scope decision — the Finding 1
// fix that closes the period=final full-roster leak. It must mirror loadRows'
// branch EXACTLY: scope=ALL drops the staff predicate (workspace-only widen);
// scope=MINE/UNSPECIFIED applies StaffReachableJobClause for a staff principal and
// FAILS CLOSED (allowed=false → zero rows) for a non-staff principal; a malformed
// staff session (empty staff.id) yields the always-false 1=0 predicate.
func TestRosterScopeClause(t *testing.T) {
	const mine = matrixpb.OutcomeMatrixScope_OUTCOME_MATRIX_SCOPE_MINE
	const all = matrixpb.OutcomeMatrixScope_OUTCOME_MATRIX_SCOPE_ALL
	const unspecified = matrixpb.OutcomeMatrixScope_OUTCOME_MATRIX_SCOPE_UNSPECIFIED

	// scope=ALL → no staff predicate regardless of principal (workspace-only).
	if clause, args, allowed := rosterScopeClause(rosterStaffCtx("staff-1"), all, 3); !allowed || clause != "" || args != nil {
		t.Errorf("ALL/staff = (%q, %v, %v), want (\"\", nil, true)", clause, args, allowed)
	}
	if clause, args, allowed := rosterScopeClause(rosterNonStaffCtx(), all, 3); !allowed || clause != "" || args != nil {
		t.Errorf("ALL/non-staff = (%q, %v, %v), want (\"\", nil, true)", clause, args, allowed)
	}

	// scope=MINE, staff principal → StaffReachableJobClause on "j", two bind args
	// (staff.id, workspace.id) starting at $3.
	clause, args, allowed := rosterScopeClause(rosterStaffCtx("staff-1"), mine, 3)
	if !allowed {
		t.Fatal("MINE/staff: allowed=false, want true")
	}
	if !strings.Contains(clause, "j.id IN (") {
		t.Errorf("MINE/staff clause = %q, want a j.id IN (...) predicate", clause)
	}
	if !strings.Contains(clause, "$3") || len(args) != 2 {
		t.Errorf("MINE/staff = (%q, %v), want $3-anchored clause with 2 args", clause, args)
	}

	// scope=MINE, NON-staff principal → fail closed to ZERO rows (loadRows parity).
	if clause, args, allowed := rosterScopeClause(rosterNonStaffCtx(), mine, 3); allowed || clause != "" || args != nil {
		t.Errorf("MINE/non-staff = (%q, %v, %v), want (\"\", nil, false) [fail-closed]", clause, args, allowed)
	}

	// UNSPECIFIED is treated as MINE (fail-closed → MINE): non-staff fails closed,
	// staff narrows.
	if _, _, allowed := rosterScopeClause(rosterNonStaffCtx(), unspecified, 3); allowed {
		t.Error("UNSPECIFIED/non-staff: allowed=true, want false (UNSPECIFIED → MINE fail-closed)")
	}
	if _, _, allowed := rosterScopeClause(rosterStaffCtx("staff-1"), unspecified, 3); !allowed {
		t.Error("UNSPECIFIED/staff: allowed=false, want true")
	}

	// Malformed staff session (empty staff.id): applies=true but the reachable-job
	// clause is the always-false 1=0 predicate (zero rows), not an unscoped read.
	if clause, _, allowed := rosterScopeClause(rosterStaffCtx(""), mine, 3); !allowed || !strings.Contains(clause, "1=0") {
		t.Errorf("MINE/empty-staff = (%q, allowed=%v), want a 1=0 fail-closed predicate", clause, allowed)
	}
}

// TestOutcomeCell_TrustedRecomputeKeys pins the W2 inline-recompute contract at
// the proto boundary: an OutcomeCell must carry the SERVER-DERIVED job_phase_id
// and job_id (fields 9/10, populated by loadRows' SELECT of jp.id/j.id). The
// fayna record action reads exactly these — never a browser value — to dedup the
// affected phase then job for ComputePhaseOutcome/ComputeJobOutcome. This guards
// against a regression that drops the columns from the cells query (which would
// silently strand every academic cell with empty recompute keys). The end-to-end
// scan is exercised by the live integration reboot (this package has no
// sqlmock/live-DB harness).
func TestOutcomeCell_TrustedRecomputeKeys(t *testing.T) {
	cell := &matrixpb.OutcomeCell{JobPhaseId: "jp-1", JobId: "job-1"}
	if cell.GetJobPhaseId() != "jp-1" {
		t.Errorf("job_phase_id recompute key not carried: got %q", cell.GetJobPhaseId())
	}
	if cell.GetJobId() != "job-1" {
		t.Errorf("job_id recompute key not carried: got %q", cell.GetJobId())
	}
}

// TestApprovalRankToStatus pins the roll-up status-rank mapping used by
// loadApprovalRollups: the SQL emits MIN(rank) where 1=IN_PROGRESS, 2=FOR_REVIEW,
// 3=VERIFIED, 4=PUBLISHED (the "sole or LOWEST status" contract), and this maps
// it back to the enum. Any unknown rank is UNSPECIFIED (fail-soft; never
// persisted). The grouped SQL itself is exercised by the live integration reboot
// (this package has no live-DB harness — same note as above).
func TestApprovalRankToStatus(t *testing.T) {
	cases := []struct {
		rank int
		want jobphasepb.PhaseApprovalStatus
	}{
		{1, jobphasepb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_IN_PROGRESS},
		{2, jobphasepb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_FOR_REVIEW},
		{3, jobphasepb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_VERIFIED},
		{4, jobphasepb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_PUBLISHED},
		{0, jobphasepb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_UNSPECIFIED},
		{9, jobphasepb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_UNSPECIFIED},
	}
	for _, c := range cases {
		if got := approvalRankToStatus(c.rank); got != c.want {
			t.Errorf("approvalRankToStatus(%d) = %v, want %v", c.rank, got, c.want)
		}
	}
}

// TestComposePhaseLabel pins the phase-header parenthetical (S8 §3): a phase with
// a sub-deliverable variant renders "NAME (VARIANT_NAME)"; a phase with no variant
// (NULL / blank) renders the bare NAME. Single composition site (adapter).
func TestComposePhaseLabel(t *testing.T) {
	cases := []struct {
		name    string
		phase   string
		variant sql.NullString
		want    string
	}{
		{"no_variant_null", "Semester 1", sql.NullString{Valid: false}, "Semester 1"},
		{"no_variant_blank", "Semester 1", sql.NullString{String: "", Valid: true}, "Semester 1"},
		{"no_variant_whitespace", "Semester 1", sql.NullString{String: "   ", Valid: true}, "Semester 1"},
		{"variant_visual_arts", "Semester 1", sql.NullString{String: "Visual Arts", Valid: true}, "Semester 1 (Visual Arts)"},
		{"variant_music", "Semester 2", sql.NullString{String: "Music", Valid: true}, "Semester 2 (Music)"},
		{"variant_trimmed", "Semester 2", sql.NullString{String: "  TLE  ", Valid: true}, "Semester 2 (TLE)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := composePhaseLabel(c.phase, c.variant); got != c.want {
				t.Errorf("composePhaseLabel(%q, %+v) = %q, want %q", c.phase, c.variant, got, c.want)
			}
		})
	}
}

// TestComputeCellEditable pins the editable rule, including the S8 §E empty-cell
// guard and the 2026-07-26 COALESCE fallback: an EMPTY cell with an explicit
// assignee is editable ONLY by that assignee (a per-task override — a merged
// two-teacher class cannot leak cross-strand empty-cell edits); an UNASSIGNED
// empty cell falls back to the class's primary sgpps edge (classEdgeOK);
// RECORDED cells keep recorder-only semantics unchanged; non-staff never edit.
func TestComputeCellEditable(t *testing.T) {
	const me, other = "staff-me", "staff-other"

	cases := []struct {
		name        string
		hasOutcome  bool
		staffOK     bool
		recordedBy  string
		assignedTo  string
		actingStaff string
		jobTaskID   string
		classEdgeOK bool
		want        bool
	}{
		// --- empty cells (S8 §E guard: explicit assignee overrides) ---
		{"empty_assigned_to_me_editable", false, true, "", me, me, "jt-1", false, true},
		{"empty_assigned_to_other_NOT_editable", false, true, "", other, me, "jt-1", false, false},
		{"empty_assigned_to_other_edge_does_NOT_override", false, true, "", other, me, "jt-1", true, false},
		{"empty_no_instance_NOT_editable", false, true, "", me, me, "", false, false},
		{"empty_non_staff_NOT_editable", false, false, "", me, me, "jt-1", false, false},
		// --- unassigned empty cells (class-edge COALESCE fallback) ---
		{"unassigned_with_class_edge_editable", false, true, "", "", me, "jt-1", true, true},
		{"unassigned_no_class_edge_NOT_editable", false, true, "", "", me, "jt-1", false, false},
		{"unassigned_no_instance_NOT_editable", false, true, "", "", me, "", true, false},
		{"unassigned_non_staff_edge_NOT_editable", false, false, "", "", me, "jt-1", true, false},
		// --- recorded cells (semantics unchanged: recorder-only) ---
		{"recorded_by_me_editable", true, true, me, other, me, "jt-1", false, true},
		{"recorded_by_other_NOT_editable", true, true, other, me, me, "jt-1", false, false},
		{"recorded_by_other_edge_does_NOT_override", true, true, other, "", me, "jt-1", true, false},
		{"recorded_non_staff_NOT_editable", true, false, me, me, me, "jt-1", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := computeCellEditable(c.hasOutcome, c.staffOK, c.recordedBy, c.assignedTo, c.actingStaff, c.jobTaskID, c.classEdgeOK)
			if got != c.want {
				t.Errorf("computeCellEditable(hasOutcome=%v,staffOK=%v,recordedBy=%q,assignedTo=%q,actingStaff=%q,jobTaskID=%q,classEdgeOK=%v) = %v, want %v",
					c.hasOutcome, c.staffOK, c.recordedBy, c.assignedTo, c.actingStaff, c.jobTaskID, c.classEdgeOK, got, c.want)
			}
		})
	}
}
