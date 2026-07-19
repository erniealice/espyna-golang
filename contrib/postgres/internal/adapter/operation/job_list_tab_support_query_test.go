//go:build postgresql

package operation

import (
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/registry/entityid"
)

// TestJobListTabSupportSQL_BothKinds locks the two-branch UNION-ALL shape when
// both kinds are permitted: one category branch + one active-template branch,
// column-aligned, both bound to the session workspace $1, table names from
// entityid constants.
func TestJobListTabSupportSQL_BothKinds(t *testing.T) {
	sql := buildJobListTabSupportSQL(true, true)

	if !strings.Contains(sql, "UNION ALL") {
		t.Errorf("both kinds must UNION ALL two branches:\n%s", sql)
	}
	// Category branch: from job_category, no active filter (active AND inactive).
	if !strings.Contains(sql, "FROM "+entityid.JobCategory+" WHERE workspace_id = $1") {
		t.Errorf("category branch missing / not workspace-bound:\n%s", sql)
	}
	if strings.Contains(sql, "FROM "+entityid.JobCategory+" WHERE workspace_id = $1 AND active") {
		t.Errorf("category branch must NOT filter active (inactive tabs render too):\n%s", sql)
	}
	// Template branch: from job_template, ACTIVE only.
	if !strings.Contains(sql, "FROM "+entityid.JobTemplate+" WHERE workspace_id = $1 AND active") {
		t.Errorf("template branch missing / not active-filtered / not workspace-bound:\n%s", sql)
	}
	// Discriminator literals present.
	for _, kind := range []string{"'" + jobListTabSupportKindCategory + "'", "'" + jobListTabSupportKindTemplate + "'"} {
		if !strings.Contains(sql, kind) {
			t.Errorf("missing row-kind discriminator %s:\n%s", kind, sql)
		}
	}
	// Column alignment: both branches must project the SAME six column slots so
	// the UNION ALL is well-typed. Category NULLs job_category_id; template NULLs
	// sort_order.
	if !strings.Contains(sql, "NULL::text AS job_category_id") {
		t.Errorf("category branch must NULL-cast job_category_id for alignment:\n%s", sql)
	}
	if !strings.Contains(sql, "NULL::integer AS sort_order") {
		t.Errorf("template branch must NULL-cast sort_order for alignment:\n%s", sql)
	}
}

// TestJobListTabSupportSQL_CategoryOnly locks the single-branch shape when only
// the category kind is permitted (template denied): no UNION, only job_category.
func TestJobListTabSupportSQL_CategoryOnly(t *testing.T) {
	sql := buildJobListTabSupportSQL(true, false)
	if strings.Contains(sql, "UNION ALL") {
		t.Errorf("single permitted kind must NOT UNION:\n%s", sql)
	}
	if !strings.Contains(sql, "FROM "+entityid.JobCategory+" ") {
		t.Errorf("category-only must select job_category:\n%s", sql)
	}
	if strings.Contains(sql, "FROM "+entityid.JobTemplate+" ") {
		t.Errorf("category-only must NOT touch job_template (kind denied):\n%s", sql)
	}
}

// TestJobListTabSupportSQL_TemplateOnly locks the single-branch shape when only
// the template kind is permitted (category denied): no UNION, only job_template.
func TestJobListTabSupportSQL_TemplateOnly(t *testing.T) {
	sql := buildJobListTabSupportSQL(false, true)
	if strings.Contains(sql, "UNION ALL") {
		t.Errorf("single permitted kind must NOT UNION:\n%s", sql)
	}
	if !strings.Contains(sql, "FROM "+entityid.JobTemplate+" WHERE workspace_id = $1 AND active") {
		t.Errorf("template-only must select active job_template:\n%s", sql)
	}
	if strings.Contains(sql, "FROM "+entityid.JobCategory+" ") {
		t.Errorf("template-only must NOT touch job_category (kind denied):\n%s", sql)
	}
}

// TestJobListTabSupportSQL_NeitherKind — both denied yields an empty statement
// (the use case short-circuits before this, but the builder must be safe).
func TestJobListTabSupportSQL_NeitherKind(t *testing.T) {
	if sql := buildJobListTabSupportSQL(false, false); sql != "" {
		t.Errorf("no permitted kind must build an empty statement, got:\n%s", sql)
	}
}
