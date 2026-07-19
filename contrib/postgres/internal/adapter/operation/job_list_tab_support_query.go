//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	portsdomain "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	internalregistry "github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"

	jobcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_category"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
)

// jobListTabSupportKind* are the UNION-ALL discriminator VALUE literals (not
// table identifiers). Each SELECT stamps its row kind so the scan loop routes the
// row into the right slice.
const (
	jobListTabSupportKindCategory = "category"
	jobListTabSupportKindTemplate = "template"
)

// init self-registers the postgres job-list tab-support query with the
// composition-root factory registry (mirrors job_template_summary_query.go). The
// registry file is tag-free; only THIS file (build-tagged postgresql) calls
// Register, so non-postgres builds never wire it and the composition initializer
// degrades to a nil port (fail-closed empty response).
func init() {
	internalregistry.RegisterJobListTabSupportFactory(func(db any) any {
		sqlDB, ok := db.(*sql.DB)
		if !ok || sqlDB == nil {
			return nil
		}
		return NewPostgresJobListTabSupportQuery(sqlDB)
	})
}

// PostgresJobListTabSupportQuery implements the hand-written
// domain.JobListTabSupportQueryService port (proto-less internal-service shape).
// It answers the job-list tabstrip's category + active-template needs in ONE
// UNION-ALL statement (the 20260718 courses-list-perf Rank-1 counter-plan),
// replacing the former 12 generic-List statements.
type PostgresJobListTabSupportQuery struct {
	db *sql.DB
}

// NewPostgresJobListTabSupportQuery constructs the PG-backed tab-support reader.
func NewPostgresJobListTabSupportQuery(db *sql.DB) *PostgresJobListTabSupportQuery {
	return &PostgresJobListTabSupportQuery{db: db}
}

// ListJobListTabSupport runs the single UNION-ALL statement returning both row
// kinds. Scoping:
//   - workspace_id: taken from the SESSION identity (identity.FromContext), NEVER
//     a request param. A missing/empty workspace fails closed to an empty response.
//   - per-kind gates: req.IncludeCategories / req.IncludeTemplates are bound by
//     the use case from two INDEPENDENT ActionGatekeeper checks. A denied kind
//     arrives false and its UNION branch is omitted — the denied kind's rows never
//     load (no silent partial, no fail-open). BOTH false → empty response, no SQL.
func (a *PostgresJobListTabSupportQuery) ListJobListTabSupport(
	ctx context.Context,
	req *portsdomain.JobListTabSupportRequest,
) (*portsdomain.JobListTabSupportResponse, error) {
	empty := func() *portsdomain.JobListTabSupportResponse {
		return &portsdomain.JobListTabSupportResponse{
			Categories: make([]*jobcategorypb.JobCategory, 0),
			Templates:  make([]*jobtemplatepb.JobTemplate, 0),
		}
	}
	if req == nil || (!req.IncludeCategories && !req.IncludeTemplates) {
		return empty(), nil
	}

	// workspace_id from the session identity — required for multi-tenancy.
	// FromContext (not identity.Must) so a missing identity fails closed to an
	// empty response rather than panicking.
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil || id.WorkspaceID == "" {
		return empty(), nil
	}

	stmt := buildJobListTabSupportSQL(req.IncludeCategories, req.IncludeTemplates)
	rows, err := a.db.QueryContext(ctx, stmt, id.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("job_list_tab_support: query: %w", err)
	}
	defer rows.Close()

	resp := empty()
	for rows.Next() {
		var (
			kind, rowID, name string
			active            bool
			sortOrder         sql.NullInt32
			jobCategoryID     sql.NullString
		)
		if err := rows.Scan(&kind, &rowID, &name, &active, &sortOrder, &jobCategoryID); err != nil {
			return nil, fmt.Errorf("job_list_tab_support: scan: %w", err)
		}
		switch kind {
		case jobListTabSupportKindCategory:
			cat := &jobcategorypb.JobCategory{Id: rowID, Name: name, Active: active}
			if sortOrder.Valid {
				so := sortOrder.Int32
				cat.SortOrder = &so
			}
			resp.Categories = append(resp.Categories, cat)
		case jobListTabSupportKindTemplate:
			tpl := &jobtemplatepb.JobTemplate{Id: rowID, Name: name}
			if jobCategoryID.Valid {
				jc := jobCategoryID.String
				tpl.JobCategoryId = &jc
			}
			resp.Templates = append(resp.Templates, tpl)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("job_list_tab_support: rows: %w", err)
	}
	return resp, nil
}

// buildJobListTabSupportSQL is the pure SQL builder (no ctx, no DB — the
// permission_query_test.go testing idiom). Both branches select the SAME six
// aligned columns (kind, id, name, active, sort_order, job_category_id) so the
// UNION ALL is well-typed; the non-applicable column is NULL-cast per branch
// (category has no job_category_id; template has no sort_order). $1 is the
// session workspace, referenced by BOTH branches. Table identifiers come from
// registry/entityid constants (infra-sql-table-name-source rule). A denied kind
// is simply not appended, so the statement is a single SELECT (no UNION) or the
// two-branch UNION ALL.
func buildJobListTabSupportSQL(includeCategories, includeTemplates bool) string {
	parts := make([]string, 0, 2)
	if includeCategories {
		// ALL categories (active AND inactive) — the tabstrip renders both.
		parts = append(parts, "SELECT '"+jobListTabSupportKindCategory+"' AS kind, id, name, active, sort_order, NULL::text AS job_category_id"+
			" FROM "+entityid.JobCategory+" WHERE workspace_id = $1")
	}
	if includeTemplates {
		// ACTIVE templates only — the aggregate joins jt.active, and the deportment
		// fallback surfaces active templates at template grain.
		parts = append(parts, "SELECT '"+jobListTabSupportKindTemplate+"' AS kind, id, name, active, NULL::integer AS sort_order, job_category_id"+
			" FROM "+entityid.JobTemplate+" WHERE workspace_id = $1 AND active")
	}
	return strings.Join(parts, "\nUNION ALL\n")
}
