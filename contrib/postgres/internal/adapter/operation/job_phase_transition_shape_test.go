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
		if !strings.Contains(jobPhaseSheetLockSQL(""), frag) {
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

// groupNarrowPredicate gates the whole delivery-group feature: an absent group
// must leave every transition byte-identical to its pre-20260725 form, and a
// present one must narrow through the TIGHT delivery link (membership AND the
// job's originating subscription), never bare membership — a student holds
// active membership in a group for every year they were enrolled, so bare
// membership would let a foreign-year group transition a plausible subset of
// this template's jobs.
func TestGroupNarrowPredicate(t *testing.T) {
	t.Run("absent group changes nothing", func(t *testing.T) {
		sql, args := groupNarrowPredicate("", 4, 3)
		if sql != "" {
			t.Errorf("empty group must yield no predicate, got %q", sql)
		}
		if len(args) != 0 {
			t.Errorf("empty group must yield no args, got %v", args)
		}
		if got := jobPhaseSheetLockSQL(sql); strings.Contains(got, "sgm_g") {
			t.Error("sheet lock SQL gained a group join with no group supplied")
		}
	})

	t.Run("present group narrows through the tight delivery link", func(t *testing.T) {
		sql, args := groupNarrowPredicate("grp-1", 4, 3)
		for _, frag := range []string{
			"sgm_g.client_id = j.client_id",
			// THE load-bearing term — without it a foreign-year group matches.
			"sgm_g.subscription_id = j.origin_id",
			"sgm_g.subscription_group_id = $4",
			"sgm_g.workspace_id = $3", // workspace-bound inside the subquery
			"sgm_g.active = true",
		} {
			if !strings.Contains(sql, frag) {
				t.Errorf("group predicate missing %q", frag)
			}
		}
		if len(args) != 1 || args[0] != "grp-1" {
			t.Errorf("expected exactly the group id as arg, got %v", args)
		}
	})

	t.Run("arg placeholder is caller-supplied", func(t *testing.T) {
		// assertAllTasksOwned's probe already uses $4 for the staff facet, so its
		// group arg must be $5. A hardcoded $4 there would silently compare the
		// group against the facet.
		sql, _ := groupNarrowPredicate("grp-1", 5, 3)
		if !strings.Contains(sql, "subscription_group_id = $5") {
			t.Error("predicate ignored the caller's placeholder index")
		}
	})

	t.Run("narrowed lock SQL keeps every original guard", func(t *testing.T) {
		narrow, _ := groupNarrowPredicate("grp-1", 4, 3)
		got := jobPhaseSheetLockSQL(narrow)
		for _, frag := range []string{
			"FOR UPDATE OF jp", "ORDER BY jp.id",
			"jp.template_phase_id = $2", "j.job_template_id = $1",
			"j.workspace_id = $3", "jp.active = true", "sgm_g",
		} {
			if !strings.Contains(got, frag) {
				t.Errorf("narrowed lock SQL missing %q", frag)
			}
		}
	})
}
