//go:build postgresql

package operation

import (
	"strings"
	"testing"
)

// TestStripJobPhaseApprovalKeys proves the generic-CRUD adapter strip removes
// every server-owned approval/audit key (forgery hardening — codex CRITICAL) and
// preserves ordinary fields.
func TestStripJobPhaseApprovalKeys(t *testing.T) {
	data := map[string]any{
		"id":                "phase-1",
		"name":              "Term 1",
		"jobId":             "job-1",
		"approvalStatus":    "PHASE_APPROVAL_STATUS_PUBLISHED",
		"submittedBy":       "attacker",
		"submittedAt":       int64(1),
		"submittedAtString": "x",
		"verifiedBy":        "attacker",
		"verifiedAt":        int64(1),
		"publishedBy":       "attacker",
		"publishedAt":       int64(1),
		"returnReason":      "forged",
		"returnedBy":        "attacker",
		"returnedAt":        int64(1),
	}
	stripJobPhaseApprovalKeys(data)
	for _, k := range jobPhaseApprovalMutableKeys {
		if _, ok := data[k]; ok {
			t.Errorf("approval key %q survived strip", k)
		}
	}
	for _, k := range []string{"id", "name", "jobId"} {
		if _, ok := data[k]; !ok {
			t.Errorf("non-approval key %q was wrongly stripped", k)
		}
	}
}

func TestRequireUniformSource(t *testing.T) {
	inProgress := []lockedPhase{
		{id: "a", approvalStatus: apInProgress},
		{id: "b", approvalStatus: apInProgress},
	}
	if err := requireUniformSource(inProgress, apInProgress); err != nil {
		t.Fatalf("uniform IN_PROGRESS should pass: %v", err)
	}
	if err := requireUniformSource(inProgress, apForReview); err == nil {
		t.Fatal("wrong source state should fail")
	}
	mixed := []lockedPhase{
		{id: "a", approvalStatus: apInProgress},
		{id: "b", approvalStatus: apForReview},
	}
	if err := requireUniformSource(mixed, apInProgress); err == nil {
		t.Fatal("mixed source state should fail")
	}
}

func TestNormalizeReturnReason(t *testing.T) {
	if _, err := normalizeReturnReason("   ", true); err == nil {
		t.Fatal("published return with blank reason must error")
	}
	if r, err := normalizeReturnReason("  ", false); err != nil || r != "" {
		t.Fatalf("non-published blank reason: got (%q,%v)", r, err)
	}
	if r, _ := normalizeReturnReason("  hi  ", true); r != "hi" {
		t.Fatalf("reason not trimmed: %q", r)
	}
	long := strings.Repeat("x", returnReasonMaxLen+50)
	if r, _ := normalizeReturnReason(long, true); len(r) != returnReasonMaxLen {
		t.Fatalf("reason not capped: len=%d", len(r))
	}
}

// TestJobPhaseSheetLockSQLShape locks the serialization primitive: the sheet lock
// must lock ONLY job_phase rows (FOR UPDATE OF jp), in one global order
// (ORDER BY jp.id), under the ancestry proof + trusted workspace bind.
func TestJobPhaseSheetLockSQLShape(t *testing.T) {
	for _, frag := range []string{
		"FOR UPDATE OF jp",
		"ORDER BY jp.id",
		"jp.template_phase_id = $2",
		"j.job_template_id = $1",
		"j.workspace_id = $3",
		"jp.active = true",
	} {
		if !strings.Contains(jobPhaseSheetLockSQL, frag) {
			t.Errorf("sheet lock SQL missing %q", frag)
		}
	}
}

func TestJobPhaseParentLockSQLShape(t *testing.T) {
	for _, frag := range []string{
		"jtp.id = $1",
		"jtp.job_template_id = $2",
		// codex §1 HIGH: the parent-lock statement itself must carry the trusted
		// workspace/global ancestry, and row-mark ONLY the parent phase.
		"jt.workspace_id = $3 OR jt.workspace_id IS NULL",
		"FOR UPDATE OF jtp",
	} {
		if !strings.Contains(jobPhaseParentLockSQL, frag) {
			t.Errorf("parent lock SQL missing %q", frag)
		}
	}
}
