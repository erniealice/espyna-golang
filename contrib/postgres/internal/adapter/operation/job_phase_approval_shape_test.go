//go:build postgresql

package operation

import (
	"database/sql"
	"strings"
	"testing"

	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
)

// TestJobPhaseApprovalCols_ProjectsEveryReadField locks the P1 approval read
// surface: every persisted approval column the projections scan MUST appear in
// the shared jobPhaseApprovalCols SELECT fragment. A dropped column here is the
// f2c80100 silent-nil bug class (SELECT and positional Scan drift symmetrically,
// so database/sql raises no error and the field just returns nil).
func TestJobPhaseApprovalCols_ProjectsEveryReadField(t *testing.T) {
	mustProject := []string{
		"jp.approval_status",
		"jp.submitted_by", "jp.submitted_at",
		"jp.verified_by", "jp.verified_at",
		"jp.published_by", "jp.published_at",
		"jp.return_reason",
		"jp.returned_by", "jp.returned_at",
	}
	for _, col := range mustProject {
		if !strings.Contains(jobPhaseApprovalCols, col) {
			t.Errorf("jobPhaseApprovalCols dropped %q — read rows would silently lose this field:\n%s", col, jobPhaseApprovalCols)
		}
	}
}

// TestJobPhaseApprovalScan_ColumnCountSymmetry locks the count symmetry between
// the SELECT fragment (jobPhaseApprovalCols) and the positional scan targets
// (scanDest). Since every projection appends approval.scanDest() right after the
// eight base columns, a mismatch would mis-bind columns WITHOUT a runtime error.
func TestJobPhaseApprovalScan_ColumnCountSymmetry(t *testing.T) {
	// "jp." occurs once per projected approval column in the fragment.
	cols := strings.Count(jobPhaseApprovalCols, "jp.")
	var a jobPhaseApprovalScan
	dests := len(a.scanDest())
	const want = 10
	if cols != want {
		t.Errorf("jobPhaseApprovalCols projects %d approval columns, want %d", cols, want)
	}
	if dests != want {
		t.Errorf("scanDest() returns %d targets, want %d", dests, want)
	}
	if cols != dests {
		t.Errorf("SELECT/Scan asymmetry: %d projected columns vs %d scan targets — columns would mis-bind", cols, dests)
	}
}

// TestJobPhaseApprovalScan_Apply verifies the scan->proto mapping: the enum name
// maps to the ladder value, present audit pairs become non-nil pointers with the
// scanned values, and a null pair leaves both proto fields nil.
func TestJobPhaseApprovalScan_Apply(t *testing.T) {
	a := jobPhaseApprovalScan{
		approvalStatus: "PHASE_APPROVAL_STATUS_VERIFIED",
		submittedBy:    sql.NullString{String: "user-sub", Valid: true},
		submittedAt:    sql.NullInt64{Int64: 1111, Valid: true},
		verifiedBy:     sql.NullString{String: "user-ver", Valid: true},
		verifiedAt:     sql.NullInt64{Int64: 2222, Valid: true},
		// published + returned pairs intentionally null.
		returnReason: sql.NullString{Valid: false},
	}
	phase := &pb.JobPhase{}
	a.apply(phase)

	if phase.GetApprovalStatus() != pb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_VERIFIED {
		t.Errorf("ApprovalStatus = %v, want VERIFIED", phase.GetApprovalStatus())
	}
	if phase.SubmittedBy == nil || *phase.SubmittedBy != "user-sub" {
		t.Errorf("SubmittedBy = %v, want user-sub", phase.SubmittedBy)
	}
	if phase.SubmittedAt == nil || *phase.SubmittedAt != 1111 {
		t.Errorf("SubmittedAt = %v, want 1111", phase.SubmittedAt)
	}
	if phase.VerifiedBy == nil || *phase.VerifiedBy != "user-ver" {
		t.Errorf("VerifiedBy = %v, want user-ver", phase.VerifiedBy)
	}
	if phase.VerifiedAt == nil || *phase.VerifiedAt != 2222 {
		t.Errorf("VerifiedAt = %v, want 2222", phase.VerifiedAt)
	}
	// Null pairs must stay nil (not zero-valued pointers).
	if phase.PublishedBy != nil || phase.PublishedAt != nil {
		t.Errorf("null published pair leaked non-nil: by=%v at=%v", phase.PublishedBy, phase.PublishedAt)
	}
	if phase.ReturnedBy != nil || phase.ReturnedAt != nil {
		t.Errorf("null returned pair leaked non-nil: by=%v at=%v", phase.ReturnedBy, phase.ReturnedAt)
	}
	if phase.ReturnReason != nil {
		t.Errorf("null return_reason leaked non-nil: %v", phase.ReturnReason)
	}
}

// TestJobPhaseApprovalScan_ApplyUnknownTokenFailSoft verifies an unrecognized
// approval token leaves the proto at its zero value rather than panicking. The
// DB CHECK guarantees only the four persisted tokens exist, but the read path
// must not crash on drift.
func TestJobPhaseApprovalScan_ApplyUnknownTokenFailSoft(t *testing.T) {
	a := jobPhaseApprovalScan{approvalStatus: "PHASE_APPROVAL_STATUS_BOGUS"}
	phase := &pb.JobPhase{}
	a.apply(phase)
	if phase.GetApprovalStatus() != pb.PhaseApprovalStatus_PHASE_APPROVAL_STATUS_UNSPECIFIED {
		t.Errorf("unknown token mapped to %v, want UNSPECIFIED (zero)", phase.GetApprovalStatus())
	}
}
