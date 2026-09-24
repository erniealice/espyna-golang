//go:build postgresql

package operation

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"

	"github.com/erniealice/espyna-golang/registry/entityid"
)

// TestApprovalRollupLastReturnReasonLive runs the REAL approval roll-up SQL
// (buildApprovalRollupSQL) against a database that has a returned, IN_PROGRESS
// sheet with a stored reason, section-narrowed, and asserts the reason surfaces
// as last_return_reason (plan 20260924-approval-role-workflow D5). Read-only.
// Skips without TEST_DATABASE_URL or when no such sheet exists.
func TestApprovalRollupLastReturnReasonLive(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Skipf("cannot reach TEST_DATABASE_URL: %v", err)
	}

	var templateID, phaseID, groupID, wsID, want string
	err = db.QueryRow(`
		SELECT j.job_template_id, jp.template_phase_id, m.subscription_group_id, j.workspace_id, jp.return_reason
		FROM `+entityid.JobPhase+` jp
		JOIN `+entityid.Job+` j ON j.id = jp.job_id
		JOIN `+entityid.SubscriptionGroupMember+` m ON m.client_id = j.client_id AND m.subscription_id = j.origin_id AND m.active
		WHERE jp.active AND jp.approval_status = 'PHASE_APPROVAL_STATUS_IN_PROGRESS'
		  AND jp.returned_at IS NOT NULL AND btrim(COALESCE(jp.return_reason, '')) <> ''
		ORDER BY jp.returned_at DESC
		LIMIT 1`).Scan(&templateID, &phaseID, &groupID, &wsID, &want)
	if err == sql.ErrNoRows {
		t.Skip("no returned sheet with a reason on this database")
	}
	if err != nil {
		t.Fatalf("pick returned sheet: %v", err)
	}

	narrow, narrowArgs := groupNarrowPredicate(groupID, 3, 2)
	rows, err := db.Query(buildApprovalRollupSQL(narrow), append([]any{templateID, wsID}, narrowArgs...)...)
	if err != nil {
		t.Fatalf("roll-up query: %v", err)
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var (
			pid                      string
			target, lowest, distinct int
			hasData, frozen          bool
			blank                    int
			lastReturnReason         string
		)
		if err := rows.Scan(&pid, &target, &lowest, &distinct, &hasData, &blank, &frozen, &lastReturnReason); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if pid == phaseID {
			found = true
			if lastReturnReason == "" {
				t.Fatalf("phase %s: last_return_reason empty, want a reason (stored %q)", pid, want)
			}
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	if !found {
		t.Fatalf("roll-up did not return phase %s", phaseID)
	}
}
