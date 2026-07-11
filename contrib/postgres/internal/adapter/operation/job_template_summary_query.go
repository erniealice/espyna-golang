//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"fmt"

	summarypb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/job_template_summary"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	internalregistry "github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// originTypeSubscriptionToken is the text token the job table stores for the
// esqyma domain.operation.v1.OriginType member ORIGIN_TYPE_SUBSCRIPTION (jobs
// persist the full protojson enum name). The delivery-group / seat joins below
// are only meaningful for subscription-originated jobs, so the WHERE binds it
// exactly (belt-and-suspenders — the inner joins to subscription_group_member /
// subscription_seat already require j.origin_id to be a subscription id). It is
// a VALUE literal, not a table identifier; principalscope.go inlines the same
// token.
const originTypeSubscriptionToken = "ORIGIN_TYPE_SUBSCRIPTION"

// maxJobTemplateSummaryLimit caps a requested page size (the common
// PaginationRequest documents "max 100"). Same cap semantics as the sibling
// list adapters.
const maxJobTemplateSummaryLimit int32 = 100

// init self-registers the postgres job-template-summary query with the
// composition-root factory registry (mirrors operation/outcome_matrix_query.go).
// The registry file is tag-free; only THIS file (build-tagged postgresql) calls
// Register, so non-postgres builds never wire it and the composition initializer
// degrades to a nil port (fail-closed empty response).
func init() {
	internalregistry.RegisterJobTemplateSummaryFactory(func(db any) any {
		sqlDB, ok := db.(*sql.DB)
		if !ok {
			return nil
		}
		return NewPostgresJobTemplateSummaryQuery(sqlDB)
	})
}

// PostgresJobTemplateSummaryQuery implements the GENERATED
// operationv1.JobTemplateSummaryServiceServer interface (Q-PROTO-MODE:
// service{rpc}). Embedding UnimplementedJobTemplateSummaryServiceServer is
// MANDATORY — the interface carries an unexported marker method that only the
// Unimplemented struct can satisfy.
type PostgresJobTemplateSummaryQuery struct {
	summarypb.UnimplementedJobTemplateSummaryServiceServer
	db *sql.DB
}

// NewPostgresJobTemplateSummaryQuery constructs the PG-backed summary reader.
func NewPostgresJobTemplateSummaryQuery(db *sql.DB) summarypb.JobTemplateSummaryServiceServer {
	return &PostgresJobTemplateSummaryQuery{db: db}
}

// ListJobTemplateSummaries runs the single GROUP-BY aggregate: one row per
// job_template with >=1 resolver-scoped job for the requested status.
//
// Scoping (every predicate is workspace_id-bound from the SESSION identity,
// never a request param):
//   - workspace_id: bound on EVERY joined table that carries the column (job,
//     job_template, subscription_group_member, subscription_group,
//     price_schedule, subscription_seat, staff, product). product_plan and the
//     "user" table have NO workspace_id column (verified against the live
//     schema) — they are bound transitively (product_plan via the workspace-
//     scoped seat/template product match; user via the workspace-scoped staff),
//     exactly as outcome_matrix_query.go / the staff CTE join "user".
//   - row scope: principalscope.StaffReachableJobClause narrows a STAFF
//     principal to its reachable jobs (the subscription_seat tier widens that
//     to the class grain); a non-staff / admin principal leaves the query
//     unscoped (sees all). A staff principal with a malformed (empty) id fails
//     closed to zero rows.
func (a *PostgresJobTemplateSummaryQuery) ListJobTemplateSummaries(
	ctx context.Context,
	req *summarypb.ListJobTemplateSummariesRequest,
) (*summarypb.ListJobTemplateSummariesResponse, error) {
	if req == nil {
		return &summarypb.ListJobTemplateSummariesResponse{Success: true}, nil
	}

	// workspace_id from the session identity — required for multi-tenancy.
	// FromContext (not identity.Must) so a missing identity fails closed to an
	// empty response rather than panicking.
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil || id.WorkspaceID == "" {
		return &summarypb.ListJobTemplateSummariesResponse{Success: true}, nil
	}
	workspaceID := id.WorkspaceID

	limit, offset := paginationBounds(req.GetPagination())

	// The principalscope clause is spliced by the pure builder at the correct
	// placeholder index (the builder computes the start param and calls this fn).
	scopeFn := func(startParam int) (string, []any) {
		if _, applies := principalscope.StaffRowScope(ctx); !applies {
			// non-staff / no identity: no row narrowing (admin sees all).
			return "", nil
		}
		return principalscope.StaffReachableJobClause(ctx, "j", startParam)
	}

	stmt, args := buildListJobTemplateSummariesSQL(
		workspaceID, req.GetStatus(), req.GetSubscriptionGroupId(),
		limit, offset, scopeFn,
	)

	rows, err := a.db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("job_template_summary: query: %w", err)
	}
	defer rows.Close()

	var summaries []*summarypb.JobTemplateSummary
	for rows.Next() {
		var (
			templateID, templateName string
			groupID, groupName       string
			staffID, staffName       string
			jobCount                 int32
			priceScheduleID          sql.NullString
			priceScheduleName        sql.NullString
			outputProductID          sql.NullString
			outputProductName        sql.NullString
		)
		if err := rows.Scan(
			&templateID, &templateName,
			&groupID, &groupName,
			&staffID, &staffName,
			&jobCount,
			&priceScheduleID, &priceScheduleName,
			&outputProductID, &outputProductName,
		); err != nil {
			return nil, fmt.Errorf("job_template_summary: scan: %w", err)
		}
		summaries = append(summaries, &summarypb.JobTemplateSummary{
			JobTemplateId:         templateID,
			JobTemplateName:       templateName,
			SubscriptionGroupId:   groupID,
			SubscriptionGroupName: groupName,
			StaffId:               staffID,
			StaffName:             staffName,
			JobCount:              jobCount,
			PriceScheduleId:       priceScheduleID.String,
			PriceScheduleName:     priceScheduleName.String,
			OutputProductId:       outputProductID.String,
			OutputProductName:     outputProductName.String,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("job_template_summary: rows: %w", err)
	}

	resp := &summarypb.ListJobTemplateSummariesResponse{
		Summaries: summaries,
		Success:   true,
	}
	if p := req.GetPagination(); p != nil && limit > 0 {
		page := int32(1)
		if off := p.GetOffset(); off != nil && off.GetPage() > 1 {
			page = off.GetPage()
		}
		resp.Pagination = &commonpb.PaginationResponse{
			CurrentPage: &page,
			HasNext:     int32(len(summaries)) == limit,
			HasPrev:     page > 1,
		}
	}
	return resp, nil
}

// paginationBounds clamps a PaginationRequest to (limit, offset). limit==0 means
// "no pagination" (return every scoped row). A requested limit is clamped to
// [1, maxJobTemplateSummaryLimit]; offset is derived from the 1-based page.
func paginationBounds(p *commonpb.PaginationRequest) (limit, offset int32) {
	if p == nil {
		return 0, 0
	}
	limit = p.GetLimit()
	if limit <= 0 {
		return 0, 0
	}
	if limit > maxJobTemplateSummaryLimit {
		limit = maxJobTemplateSummaryLimit
	}
	if off := p.GetOffset(); off != nil {
		if page := off.GetPage(); page > 1 {
			offset = (page - 1) * limit
		}
	}
	return limit, offset
}

// buildListJobTemplateSummariesSQL is the pure SQL builder (no ctx, no DB — the
// permission_query_test.go testing idiom). It assembles the full statement +
// positional args in a fixed order:
//
//	$1                = workspaceID (referenced by EVERY table's workspace_id)
//	$2 (if status!="")= the job status token
//	$next (if group)  = subscription_group_id filter
//	scope args        = scopeFn(startParam) result, spliced verbatim
//	$next,$next+1     = LIMIT, OFFSET (when limit>0)
//
// scopeFn receives the placeholder index at which its args begin, so the caller
// (principalscope.StaffReachableJobClause) and this builder never disagree on
// numbering. A nil scopeFn (or one returning "") leaves the query unscoped.
func buildListJobTemplateSummariesSQL(
	workspaceID, status, groupID string,
	limit, offset int32,
	scopeFn func(startParam int) (clause string, args []any),
) (stmt string, args []any) {
	args = []any{workspaceID}
	p := 2

	where := "WHERE j.job_template_id IS NOT NULL" +
		" AND j.workspace_id = $1" +
		" AND j.active" +
		" AND j.origin_type = '" + originTypeSubscriptionToken + "'"

	if status != "" {
		where += fmt.Sprintf(" AND j.status = $%d", p)
		args = append(args, status)
		p++
	}
	if groupID != "" {
		where += fmt.Sprintf(" AND sg.id = $%d", p)
		args = append(args, groupID)
		p++
	}
	if scopeFn != nil {
		clause, scopeArgs := scopeFn(p)
		where += clause
		args = append(args, scopeArgs...)
		p += len(scopeArgs)
	}

	limitClause := ""
	if limit > 0 {
		limitClause = fmt.Sprintf(" LIMIT $%d OFFSET $%d", p, p+1)
		args = append(args, limit, offset)
		p += 2
	}

	stmt = jobTemplateSummarySelectFrom() + "\n" + where + "\n" + jobTemplateSummaryGroupOrder() + limitClause
	return stmt, args
}

// jobTemplateSummarySelectFrom is the SELECT + FROM + JOIN skeleton. EVERY table
// identifier comes from registry/entityid constants (infra-sql-table-name-source
// rule — never a quoted literal). "user" is the ONE double-quoted identifier
// (reserved word), exactly as the staff CTE join / outcome_matrix adapter do.
// Every table with a workspace_id column is bound to $1; product_plan and "user"
// have none and are bound transitively (see the method doc).
func jobTemplateSummarySelectFrom() string {
	return `SELECT
    jt.id                          AS job_template_id,
    jt.name                        AS job_template_name,
    sg.id                          AS subscription_group_id,
    sg.name                        AS subscription_group_name,
    st.id                          AS staff_id,
    COALESCE(NULLIF(TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')), ''), st.id) AS staff_name,
    COUNT(DISTINCT j.id)           AS job_count,
    ps.id                          AS price_schedule_id,
    ps.name                        AS price_schedule_name,
    jt.output_product_id           AS output_product_id,
    op.name                        AS output_product_name
FROM ` + entityid.Job + ` j
JOIN ` + entityid.JobTemplate + ` jt
       ON jt.id = j.job_template_id AND jt.workspace_id = $1 AND jt.active
JOIN ` + entityid.SubscriptionGroupMember + ` sgm
       ON sgm.subscription_id = j.origin_id AND sgm.client_id = j.client_id
      AND sgm.workspace_id = $1 AND sgm.active
JOIN ` + entityid.SubscriptionGroup + ` sg
       ON sg.id = sgm.subscription_group_id AND sg.workspace_id = $1 AND sg.active
LEFT JOIN ` + entityid.PriceSchedule + ` ps
       ON ps.id = sg.price_schedule_id AND ps.workspace_id = $1
JOIN ` + entityid.SubscriptionSeat + ` ss
       ON ss.subscription_id = j.origin_id AND ss.client_id = j.client_id
      AND ss.status = 'active' AND ss.active AND ss.workspace_id = $1
JOIN ` + entityid.ProductPlan + ` pl
       ON pl.id = ss.product_plan_id AND pl.product_id = jt.output_product_id
JOIN ` + entityid.Staff + ` st
       ON st.id = ss.staff_id AND st.workspace_id = $1
LEFT JOIN "` + entityid.User + `" u
       ON u.id = st.user_id AND u.active
LEFT JOIN ` + entityid.Product + ` op
       ON op.id = jt.output_product_id AND op.workspace_id = $1`
}

// jobTemplateSummaryGroupOrder is the GROUP BY + ORDER BY tail. The grain is one
// row per (template, group, staff, schedule, product); education1 has exactly
// one group+staff+schedule per template, so this collapses to one row per
// template. ORDER BY group name then template name is the LOCKED view order.
func jobTemplateSummaryGroupOrder() string {
	return `GROUP BY jt.id, jt.name, sg.id, sg.name, st.id, u.first_name, u.last_name,
         ps.id, ps.name, jt.output_product_id, op.name
ORDER BY sg.name, jt.name`
}
