//go:build postgresql

package operation

import (
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/registry/entityid"
)

// TestJobTemplatePhaseListByTemplate_ColumnsProjected is a read-regression lock
// on the ListByJobTemplate SELECT projection. It exists because commit f2c80100
// shipped a SILENT production bug: the SELECT list and the positional
// rows.Scan(...) were symmetrically missing jtp.scoring_scheme_id, so
// database/sql raised NO column-count error — the field simply came back nil in
// every spawned job_phase, invisible to the education grading pipeline. Item #3
// then restored predecessor_template_phase_id via the exact same class of fix.
//
// Scope note (item #3, Q3-A/Q3-B): the billing_* columns are DELIBERATELY not
// projected — they activate the milestone-billing event materializer, a revenue
// surface deferred to a separate follow-up ticket. See the scope note on
// jobTemplatePhaseListByTemplateSQL().
//
// The use-case unit tests cannot catch this class of bug: their
// stubJobTemplatePhaseRepo returns the caller-populated proto verbatim and never
// touches SQL. A string assertion on the real projection is the cheapest test
// that would have failed on the original f2c80100 defect (and on any future
// re-drop of one of these columns).
func TestJobTemplatePhaseListByTemplate_ColumnsProjected(t *testing.T) {
	sql := jobTemplatePhaseListByTemplateSQL()

	// Every column the adapter's rows.Scan(...) destinations expect MUST appear
	// in the SELECT list. A dropped column here is the f2c80100 bug class.
	mustProject := []string{
		"jtp.id",
		"jtp.date_created",
		"jtp.date_modified",
		"jtp.active",
		"jtp.job_template_id",
		"jtp.name",
		"jtp.phase_order",
		"jtp.scoring_scheme_id",             // the f2c80100 regression
		"jtp.predecessor_template_phase_id", // item #3 restore
	}
	for _, col := range mustProject {
		if !strings.Contains(sql, col) {
			t.Errorf("ListByJobTemplate SELECT dropped %q — spawned job_phase rows would silently lose this field:\n%s", col, sql)
		}
	}
}

// TestJobTemplatePhaseListByTemplate_ColumnOrderMatchesScan locks the SELECT
// column ORDER against the positional rows.Scan(...) in ListByJobTemplate. The
// Scan is positional, so a SELECT-list reorder that is not mirrored in the Scan
// destinations would mis-bind columns (e.g. scoring_scheme_id scanned into the
// predecessor field) WITHOUT a runtime error — a distinct, silent failure mode
// from the dropped-column bug above. If you reorder the SELECT here, reorder the
// Scan in ListByJobTemplate identically and update this ordered list.
func TestJobTemplatePhaseListByTemplate_ColumnOrderMatchesScan(t *testing.T) {
	sql := jobTemplatePhaseListByTemplateSQL()

	// The exact projection order, matching the rows.Scan(...) destination order.
	ordered := []string{
		"jtp.id",
		"jtp.date_created",
		"jtp.date_modified",
		"jtp.active",
		"jtp.job_template_id",
		"jtp.name",
		"jtp.phase_order",
		"jtp.scoring_scheme_id",
		"jtp.predecessor_template_phase_id",
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

// TestJobTemplatePhaseListByTemplate_TableFromEntityID locks the
// infra-sql-table-name-source rule (same as TestJobTemplateSummarySQL_*): the
// table identifier comes from the registry/entityid constant, never a
// hand-typed literal.
func TestJobTemplatePhaseListByTemplate_TableFromEntityID(t *testing.T) {
	sql := jobTemplatePhaseListByTemplateSQL()
	if !strings.Contains(sql, " "+entityid.JobTemplatePhase+" jtp") {
		t.Errorf("SELECT/FROM missing %q aliased jtp (table must come from entityid):\n%s", entityid.JobTemplatePhase, sql)
	}
}
