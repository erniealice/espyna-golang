//go:build postgresql

package operation

import (
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/registry/entityid"
)

// TestJobTemplateTaskListByPhase_ColumnsProjected is a read-regression lock on
// the ListByPhase SELECT projection. The positional rows.Scan(...) in ListByPhase
// binds by column position, so a column dropped from the SELECT (or from the
// Scan) comes back nil with NO column-count error from database/sql — a silent
// data-loss bug. Every column the Scan destinations expect MUST appear here.
func TestJobTemplateTaskListByPhase_ColumnsProjected(t *testing.T) {
	sql := jobTemplateTaskListByPhaseSQL()

	mustProject := []string{
		"jtt.id",
		"jtt.date_created",
		"jtt.date_modified",
		"jtt.active",
		"jtt.job_template_phase_id",
		"jtt.name",
		"jtt.step_order",
		"jtt.estimated_duration_minutes",
		"jtt.code", // stable path key for the document read path
	}
	for _, col := range mustProject {
		if !strings.Contains(sql, col) {
			t.Errorf("ListByPhase SELECT dropped %q — the scanned field would silently be nil:\n%s", col, sql)
		}
	}
}

// TestJobTemplateTaskListByPhase_ColumnOrderMatchesScan locks the SELECT column
// ORDER against the positional rows.Scan(...) in ListByPhase. A SELECT-list
// reorder not mirrored in the Scan would mis-bind columns WITHOUT a runtime
// error. If you reorder the SELECT, reorder the Scan identically and update this
// ordered list.
func TestJobTemplateTaskListByPhase_ColumnOrderMatchesScan(t *testing.T) {
	sql := jobTemplateTaskListByPhaseSQL()

	ordered := []string{
		"jtt.id",
		"jtt.date_created",
		"jtt.date_modified",
		"jtt.active",
		"jtt.job_template_phase_id",
		"jtt.name",
		"jtt.step_order",
		"jtt.estimated_duration_minutes",
		"jtt.code",
	}
	prev := -1
	prevCol := ""
	for _, col := range ordered {
		idx := strings.Index(sql, col)
		if idx < 0 {
			t.Fatalf("column %q not found in SELECT:\n%s", col, sql)
		}
		if idx <= prev {
			t.Errorf("column %q appears at/before %q — SELECT order drifted from the positional Scan order:\n%s", col, prevCol, sql)
		}
		prev = idx
		prevCol = col
	}
}

// TestJobTemplateTaskListByPhase_TableFromEntityID locks the
// infra-sql-table-name-source rule: the table identifier comes from the
// registry/entityid constant, never a hand-typed literal.
func TestJobTemplateTaskListByPhase_TableFromEntityID(t *testing.T) {
	sql := jobTemplateTaskListByPhaseSQL()
	if !strings.Contains(sql, " "+entityid.JobTemplateTask+" jtt") {
		t.Errorf("SELECT/FROM missing %q aliased jtt (table must come from entityid):\n%s", entityid.JobTemplateTask, sql)
	}
}

// TestJobTemplateTask_RawReadsWorkspaceScoped is the cross-tenant regression.
// job_template_task has no workspace_id column, so these raw-SQL reads bypass the
// workspace-aware decorator. Tenancy is derived up the two-hop parent chain
// task -> job_template_phase -> job_template; both JOINs and the jt.workspace_id
// predicate MUST be present or a workspace-A caller could enumerate workspace-B
// tasks.
func TestJobTemplateTask_RawReadsWorkspaceScoped(t *testing.T) {
	joinPhase := "JOIN " + entityid.JobTemplatePhase + " jtp ON jtp.id = jtt.job_template_phase_id"
	joinTemplate := "JOIN " + entityid.JobTemplate + " jt ON jt.id = jtp.job_template_id"

	cases := []struct {
		name string
		sql  string
		pred string
	}{
		{"ListByPhase", jobTemplateTaskListByPhaseSQL(), "$2::text = '' OR jt.workspace_id = $2::text"},
		{"ItemPageData", jobTemplateTaskItemPageDataSQL(), "$2::text = '' OR jt.workspace_id = $2::text"},
		{"ListPageData", jobTemplateTaskListPageDataSQL("ORDER BY e.step_order ASC"), "$4::text = '' OR jt.workspace_id = $4::text"},
	}
	for _, c := range cases {
		if !strings.Contains(c.sql, joinPhase) {
			t.Errorf("%s: missing phase JOIN %q — unscoped cross-tenant read:\n%s", c.name, joinPhase, c.sql)
		}
		if !strings.Contains(c.sql, joinTemplate) {
			t.Errorf("%s: missing template JOIN %q — unscoped cross-tenant read:\n%s", c.name, joinTemplate, c.sql)
		}
		if !strings.Contains(c.sql, c.pred) {
			t.Errorf("%s: missing workspace predicate %q — unscoped cross-tenant read:\n%s", c.name, c.pred, c.sql)
		}
	}
}

// TestJobTemplateTask_GenericGuardsWorkspaceScoped locks the by-id / create /
// generic-list tenant guards used by the generic (decorator-routed) ops, which
// otherwise pass through unscoped for this column-less child table.
func TestJobTemplateTask_GenericGuardsWorkspaceScoped(t *testing.T) {
	owned := jobTemplateTaskWorkspaceOwnedSQL()
	if !strings.Contains(owned, "JOIN "+entityid.JobTemplatePhase+" jtp ON jtp.id = jtt.job_template_phase_id") ||
		!strings.Contains(owned, "JOIN "+entityid.JobTemplate+" jt ON jt.id = jtp.job_template_id") ||
		!strings.Contains(owned, "jtt.id = $1") ||
		!strings.Contains(owned, "jt.workspace_id = $2::text") {
		t.Errorf("read/update/delete ownership guard not scoped up the parent chain:\n%s", owned)
	}

	parent := jobTemplatePhaseInWorkspaceSQL()
	if !strings.Contains(parent, "JOIN "+entityid.JobTemplate+" jt ON jt.id = jtp.job_template_id") ||
		!strings.Contains(parent, "jtp.id = $1") ||
		!strings.Contains(parent, "jt.workspace_id = $2::text") {
		t.Errorf("create parent-FK validation not scoped through the parent template workspace:\n%s", parent)
	}

	list := jobTemplateTaskOwnedPhaseIDsSQL()
	if !strings.Contains(list, "jtp.id = ANY($1)") || !strings.Contains(list, "jt.workspace_id = $2::text") {
		t.Errorf("generic-list owned-phase filter not scoped to workspace:\n%s", list)
	}
}
