//go:build postgresql

package operation

import (
	"strings"
	"testing"
)

// TestJobOutcomeSummaryReadsAreWorkspaceBound pins the HAZ-02 close (Q-SEC-7):
// every custom-SQL job_outcome_summary read MUST carry a workspace_id predicate
// bound to a positional arg (session identity, never a request param), alongside
// the existing staff clause. A non-staff (admin/workspace_user) principal's read
// is otherwise workspace-unbounded (cross-tenant leak). This table is shared with
// service-admin (professional tier), so the predicate is behavior-preserving in a
// single-workspace deployment but load-bearing multi-tenant.
func TestJobOutcomeSummaryReadsAreWorkspaceBound(t *testing.T) {
	// GetJobOutcomeSummaryListPageData: workspace $4, staff $5.
	list := jobOutcomeSummaryListPageDataSQL("jos.id", " AND jos.issued_by = $5", "ORDER BY date_created DESC")
	if !strings.Contains(list, "jos.workspace_id = $4") {
		t.Errorf("list page-data SQL missing workspace predicate ($4):\n%s", list)
	}
	if !strings.Contains(list, "issued_by = $5") {
		t.Errorf("list page-data SQL dropped the staff clause ($5):\n%s", list)
	}

	// GetJobOutcomeSummaryItemPageData: workspace $2, staff $3, no suffix.
	item := jobOutcomeSummaryByColumnSQL("jos.id = $1", " AND jos.issued_by = $3", "")
	if !strings.Contains(item, "jos.workspace_id = $2") {
		t.Errorf("item SQL missing workspace predicate ($2):\n%s", item)
	}
	if !strings.Contains(item, "jos.id = $1") {
		t.Errorf("item SQL lost its id filter ($1):\n%s", item)
	}

	// GetByJob: workspace $2, no staff clause here, ORDER BY/LIMIT suffix retained.
	byJob := jobOutcomeSummaryByColumnSQL("jos.job_id = $1", "", "ORDER BY jos.date_created DESC\n\t\tLIMIT 1")
	if !strings.Contains(byJob, "jos.workspace_id = $2") {
		t.Errorf("get-by-job SQL missing workspace predicate ($2):\n%s", byJob)
	}
	if !strings.Contains(byJob, "LIMIT 1") {
		t.Errorf("get-by-job SQL lost its ORDER BY/LIMIT suffix:\n%s", byJob)
	}

	// All three project workspace_id (scan faithfulness) — the single-row builder
	// projects it, and the list builder is fed josColumns by the caller.
	if !strings.Contains(item, "jos.workspace_id") || !strings.Contains(byJob, "jos.workspace_id") {
		t.Errorf("single-row projection dropped workspace_id column")
	}
}
