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

	// The aggregate yields ONE row per (template, staff): a merged, multi-
	// deliverer template (one deliverer per delivered phase — each holds an active
	// subscription_seat whose product_plan matches the template's umbrella output
	// product at umbrella grain) produces >1 row. Scan raw rows, then collate them
	// into one summary per template carrying all deliverers (collateDeliverySummaries).
	var scanned []summaryScanRow
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
			// job_template.job_category_id is a NULLABLE FK (esqyma job_template.proto
			// field 32) — a legacy/uncategorized template row is NULL. Scan through
			// sql.NullString and map NULL → "" (the read-model empty string); the
			// landing view maps "" → the single "Uncategorized" bucket (R9 §3.0/§3.1),
			// mirroring the price_schedule_id/output_product_id nullable-scan pattern
			// above.
			jobCategoryID sql.NullString
			// R7 P4 approval preaggregate — template-wide (fields 14-17) + group+
			// template (fields 18-21) grains. Counts/mixed are COALESCEd in SQL; the
			// lowest ranks stay NULLable (NULL = zero data-bearing sheets → the enum
			// UNSPECIFIED via approvalRankToStatus, the neutral not-started default).
			publishedCount, phaseCount           int32
			lowestRank                           sql.NullInt64
			mixedAttention                       bool
			groupPublishedCount, groupPhaseCount int32
			groupLowestRank                      sql.NullInt64
			groupMixedAttention                  bool
		)
		if err := rows.Scan(
			&templateID, &templateName,
			&groupID, &groupName,
			&staffID, &staffName,
			&jobCount,
			&priceScheduleID, &priceScheduleName,
			&outputProductID, &outputProductName,
			&jobCategoryID,
			&publishedCount, &phaseCount, &lowestRank, &mixedAttention,
			&groupPublishedCount, &groupPhaseCount, &groupLowestRank, &groupMixedAttention,
		); err != nil {
			return nil, fmt.Errorf("job_template_summary: scan: %w", err)
		}
		scanned = append(scanned, summaryScanRow{
			templateID: templateID, templateName: templateName,
			groupID: groupID, groupName: groupName,
			staffID: staffID, staffName: staffName,
			jobCount:            jobCount,
			priceScheduleID:     priceScheduleID.String,
			priceScheduleName:   priceScheduleName.String,
			outputProductID:     outputProductID.String,
			outputProductName:   outputProductName.String,
			jobCategoryID:       jobCategoryID.String,
			publishedCount:      publishedCount,
			phaseCount:          phaseCount,
			lowestRank:          nullRankToInt(lowestRank),
			mixedAttention:      mixedAttention,
			groupPublishedCount: groupPublishedCount,
			groupPhaseCount:     groupPhaseCount,
			groupLowestRank:     nullRankToInt(groupLowestRank),
			groupMixedAttention: groupMixedAttention,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("job_template_summary: rows: %w", err)
	}

	summaries := collateDeliverySummaries(scanned)

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

// summaryScanRow is one raw (template, staff) aggregate row before deliverer
// collation.
type summaryScanRow struct {
	templateID, templateName string
	groupID, groupName       string
	staffID, staffName       string
	jobCount                 int32
	priceScheduleID          string
	priceScheduleName        string
	outputProductID          string
	outputProductName        string
	jobCategoryID            string // "" when the template's job_category FK is NULL (→ Uncategorized bucket)
	// R7 P4 approval preaggregate. Ranks are the SQL ladder ranks (1..4); 0 =
	// NULL = zero data-bearing sheets (→ PhaseApprovalStatus UNSPECIFIED).
	// The template-wide quadruple is functionally determined by templateID; the
	// group quadruple by (templateID, groupID) — both ⊆ the collation key, so
	// they are constant across a key's folded staff rows (set-once first-seen).
	publishedCount, phaseCount           int32
	lowestRank                           int
	mixedAttention                       bool
	groupPublishedCount, groupPhaseCount int32
	groupLowestRank                      int
	groupMixedAttention                  bool
}

// nullRankToInt maps a NULLable SQL ladder rank to its int form (0 = NULL = no
// data-bearing sheet at that grain; approvalRankToStatus(0) → UNSPECIFIED).
func nullRankToInt(r sql.NullInt64) int {
	if !r.Valid {
		return 0
	}
	return int(r.Int64)
}

// collateDeliverySummaries folds the per-(template,staff) aggregate rows into ONE
// JobTemplateSummary per (template, group, schedule, product), gathering every
// staff row as a Deliverer in ARRIVAL order — the adapter orders rows by
// (group, template, staff_name, staff_id), a stable, deterministic deliverer order.
// A merged deliverable legitimately has >1 deliverer (one per delivered phase);
// single-deliverer templates collapse to one summary with one Deliverer (unchanged
// shape for today's data). job_count is the MAX across a template's deliverer rows:
// every deliverer's active seats reach the FULL merged roster (umbrella-grain seat
// match), so the per-staff DISTINCT counts are equal and MAX is the roster size
// (over a single row it is that row's count — today's behavior byte-for-byte). The
// collation key includes group/schedule/product so a template that ever spanned two
// delivery groups keeps its distinct rows (folds ONLY the staff axis).
func collateDeliverySummaries(scanned []summaryScanRow) []*summarypb.JobTemplateSummary {
	var out []*summarypb.JobTemplateSummary
	byKey := map[string]*summarypb.JobTemplateSummary{}
	for _, r := range scanned {
		key := r.templateID + "\x00" + r.groupID + "\x00" + r.priceScheduleID + "\x00" + r.outputProductID
		s := byKey[key]
		if s == nil {
			s = &summarypb.JobTemplateSummary{
				JobTemplateId:         r.templateID,
				JobTemplateName:       r.templateName,
				SubscriptionGroupId:   r.groupID,
				SubscriptionGroupName: r.groupName,
				JobCount:              r.jobCount,
				PriceScheduleId:       r.priceScheduleID,
				PriceScheduleName:     r.priceScheduleName,
				OutputProductId:       r.outputProductID,
				OutputProductName:     r.outputProductName,
				// job_category_id is functionally determined by the template, so it is
				// constant across a collation key's folded staff rows — set once on
				// first-seen. "" (NULL FK) collates to the Uncategorized bucket (§3.0).
				JobCategoryId: r.jobCategoryID,
				// R7 P4 approval quadruples — template-wide (14-17, determined by the
				// template) and group+template (18-21, determined by (template, group)):
				// both ⊆ the collation key, so set-once first-seen, constant across the
				// folded staff rows. Rank 0 (no data-bearing sheet) → UNSPECIFIED.
				PublishedCount:      r.publishedCount,
				PhaseCount:          r.phaseCount,
				LowestStatus:        approvalRankToStatus(r.lowestRank),
				MixedAttention:      r.mixedAttention,
				GroupPublishedCount: r.groupPublishedCount,
				GroupPhaseCount:     r.groupPhaseCount,
				GroupLowestStatus:   approvalRankToStatus(r.groupLowestRank),
				GroupMixedAttention: r.groupMixedAttention,
			}
			byKey[key] = s
			out = append(out, s)
		}
		if r.jobCount > s.JobCount {
			s.JobCount = r.jobCount
		}
		if r.staffID != "" {
			s.Deliverers = append(s.Deliverers, &summarypb.Deliverer{
				StaffId:   r.staffID,
				StaffName: r.staffName,
			})
		}
	}
	return out
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
// permission_query_test.go testing idiom). It assembles ONE statement — the
// jj/dd MATERIALIZED CTEs + the R7 P4 approval preaggregate CTEs (pa/tp/ta/ga)
// + the `base` delivery aggregation + the top-level approval join — with
// positional args in a fixed order:
//
//	$1                = workspaceID (referenced by EVERY table's workspace_id)
//	$2 (if status!="")= the job status token (bound INSIDE the jj CTE)
//	$next (if group)  = subscription_group_id filter (bound on the OUTER sg join)
//	scope args        = scopeFn(startParam) result, spliced into the jj CTE
//	$next,$next+1     = LIMIT, OFFSET (when limit>0)
//
// The 20260718 courses-list-perf rewrite (P1: 1024ms→395ms, 310-row parity)
// forces a hash-join plan structurally instead of relying on planner knobs:
//
//	jj  MATERIALIZED CTE = job ⋈ job_template, pre-filtered (workspace, active,
//	     subscription origin, optional status, optional STAFF row-scope). Emits
//	     one row per (job, template) carrying the subscription_id/client_id/
//	     output_product_id hash-join keys.
//	dd  MATERIALIZED CTE = subscription_seat ⋈ product_plan ⋈ staff ⋈ "user"
//	     (LEFT) ⋈ subscription_group_member — the deliverer/group side, keyed by
//	     (subscription_id, client_id, product_id). sgm is correlated to the seat
//	     on (subscription_id, client_id): in the original both sgm AND ss bind to
//	     the job's (origin_id, client_id), so binding them to each other is
//	     transitively identical once jj joins in.
//
// jj ⋈ dd on (subscription_id, client_id, output_product_id/product_id) restores
// the original inner-join semantics (the seat's product_plan.product_id must
// equal the template's output_product_id); sg/ps/op then join at the outer level
// and GROUP BY folds the staff axis. Row grain, column order, GROUP BY and ORDER
// BY are byte-equivalent to the pre-rewrite aggregate.
//
// scopeFn receives the placeholder index at which its args begin, so the caller
// (principalscope.StaffReachableJobClause) and this builder never disagree on
// numbering. The scope clause aliases the job table as "j" — which is exactly
// the jj CTE's inner alias — so it splices into the jj CTE's WHERE. A nil scopeFn
// (or one returning "") leaves the query unscoped.
func buildListJobTemplateSummariesSQL(
	workspaceID, status, groupID string,
	limit, offset int32,
	scopeFn func(startParam int) (clause string, args []any),
) (stmt string, args []any) {
	args = []any{workspaceID}
	p := 2

	// jjWhere is the jj CTE's inner WHERE: base subscription-job predicates +
	// optional status + optional STAFF row-scope (all on the job aliased "j").
	jjWhere := "WHERE j.job_template_id IS NOT NULL" +
		" AND j.workspace_id = $1" +
		" AND j.active" +
		" AND j.origin_type = '" + originTypeSubscriptionToken + "'"

	if status != "" {
		jjWhere += fmt.Sprintf(" AND j.status = $%d", p)
		args = append(args, status)
		p++
	}

	// The optional group filter stays on the OUTER sg join (sg is joined after
	// the jj×dd hash join). Its placeholder is numbered before the scope args, so
	// the arg order remains (ws, status, group, scope…, limit, offset) — identical
	// to the pre-rewrite builder.
	outerWhere := ""
	if groupID != "" {
		outerWhere = fmt.Sprintf("\nWHERE sg.id = $%d", p)
		args = append(args, groupID)
		p++
	}

	if scopeFn != nil {
		clause, scopeArgs := scopeFn(p)
		jjWhere += clause
		args = append(args, scopeArgs...)
		p += len(scopeArgs)
	}

	limitClause := ""
	if limit > 0 {
		limitClause = fmt.Sprintf(" LIMIT $%d OFFSET $%d", p, p+1)
		args = append(args, limit, offset)
		p += 2
	}

	stmt = jobTemplateSummaryCTEs(jjWhere) + ",\nbase AS (\n" +
		jobTemplateSummarySelectFrom() + outerWhere + "\n" +
		jobTemplateSummaryGroupOrder() + "\n)\n" +
		jobTemplateSummaryApprovalSelect() + limitClause
	return stmt, args
}

// approvalStatusRankCASE is the approval_status → ladder-rank mapping the
// preaggregate CTEs use (1=IN_PROGRESS … 4=PUBLISHED; unknown token → 1,
// fail-conservative). EXACTLY mirrors outcome_matrix_query.go's roll-up CASE so
// the courses chip and the matrix bar can never disagree on rank semantics;
// approvalRankToStatus (same package) maps the rank back to the enum.
const approvalStatusRankCASE = `CASE jp.approval_status
             WHEN 'PHASE_APPROVAL_STATUS_IN_PROGRESS' THEN 1
             WHEN 'PHASE_APPROVAL_STATUS_FOR_REVIEW'  THEN 2
             WHEN 'PHASE_APPROVAL_STATUS_VERIFIED'    THEN 3
             WHEN 'PHASE_APPROVAL_STATUS_PUBLISHED'   THEN 4
             ELSE 1 END`

// jobTemplateSummaryCTEs builds the two MATERIALIZED CTE definitions (jj, dd)
// PLUS the R7 P4 phase-approval preaggregate CTEs (pa → tp → ta / ga).
// EVERY table identifier comes from registry/entityid constants (infra-sql-
// table-name-source rule — never a quoted literal). "user" is the ONE double-
// quoted identifier (reserved word). jjWhere carries the job-side predicates
// (built by buildListJobTemplateSummariesSQL: workspace/active/origin + optional
// status + optional STAFF row-scope on the "j" alias). MATERIALIZED pins the
// hash-join plan (P1 empirical win) regardless of small-table stats.
//
// Approval preaggregate (plan 20260718-phase-approval-workflow §4.5 + the
// 20260719-report-cards-landing §3.5 grain amendment; codex-tandem "Courses-list
// chip" contract). Sourced from the SAME scoped jj job set as the delivery row
// (the STAFF splice lives in jjWhere, so STAFF and admin see a chip over exactly
// the job scope of their row), and fully COLLAPSED to its two output grains
// BEFORE it ever meets the delivery join — raw job_phase/job_task/task_outcome
// rows never enter the delivery aggregate (no job/deliverer fanout):
//
//	pa  (template, group, template_phase) grain: MIN/MAX approval ladder rank
//	     over the slice's active template-backed job_phase rows + has_data
//	     (>=1 active task_outcome under an active job_task — the SAME data seam
//	     as the matrix roll-up / D6). All three aggregates are duplicate-
//	     insensitive, so the task/outcome LEFT-join fanout inside pa is harmless
//	     by construction. group_id comes from subscription_group_member on the
//	     job's (origin subscription, client) — NULL (kept) when the job is in no
//	     group: such rows still feed the template-wide grain but never a group.
//	tp  (template, template_phase): pa collapsed across groups — the R7 sheet
//	     grain (ALL rows of one (template, template_phase)).
//	ta  one row/template (fields 14-17): the courses-row TEMPLATE-WIDE roll-up.
//	ga  one row/(template, group) (fields 18-21): the R9 cell GROUP+TEMPLATE
//	     roll-up. Both compute the SAME quadruple: phase_count counts only
//	     DATA-BEARING sheets (no-data sheets are EXCLUDED from every aggregate —
//	     the D3/Q-R9-1 denominator contract), published_count counts data-bearing
//	     sheets whose every row is PUBLISHED (min_rank=4), lowest_rank is the
//	     conservative LOWEST rank across data-bearing sheets (NULL when none —
//	     scanned to UNSPECIFIED), mixed_attention is true when any data-bearing
//	     sheet is internally mixed at that grain.
func jobTemplateSummaryCTEs(jjWhere string) string {
	return `WITH jj AS MATERIALIZED (
    SELECT
        j.id               AS job_id,
        j.origin_id        AS subscription_id,
        j.client_id        AS client_id,
        jt.id              AS template_id,
        jt.name            AS template_name,
        jt.output_product_id AS output_product_id,
        jt.job_category_id AS job_category_id
    FROM ` + entityid.Job + ` j
    JOIN ` + entityid.JobTemplate + ` jt
           ON jt.id = j.job_template_id AND jt.workspace_id = $1 AND jt.active
    ` + jjWhere + `
),
dd AS MATERIALIZED (
    SELECT
        ss.subscription_id       AS subscription_id,
        ss.client_id             AS client_id,
        pl.product_id            AS product_id,
        st.id                    AS staff_id,
        u.first_name             AS first_name,
        u.last_name              AS last_name,
        sgm.subscription_group_id AS subscription_group_id
    FROM ` + entityid.SubscriptionSeat + ` ss
    JOIN ` + entityid.ProductPlan + ` pl
           ON pl.id = ss.product_plan_id
    JOIN ` + entityid.Staff + ` st
           ON st.id = ss.staff_id AND st.workspace_id = $1
    LEFT JOIN "` + entityid.User + `" u
           ON u.id = st.user_id AND u.active
    JOIN ` + entityid.SubscriptionGroupMember + ` sgm
           ON sgm.subscription_id = ss.subscription_id AND sgm.client_id = ss.client_id
          AND sgm.workspace_id = $1 AND sgm.active
    WHERE ss.status = 'active' AND ss.active AND ss.workspace_id = $1
),
pa AS (
    SELECT
        jj.template_id            AS template_id,
        gm.subscription_group_id  AS group_id,
        jp.template_phase_id      AS template_phase_id,
        MIN(` + approvalStatusRankCASE + `) AS min_rank,
        MAX(` + approvalStatusRankCASE + `) AS max_rank,
        BOOL_OR(tox.job_task_id IS NOT NULL) AS has_data
    FROM jj
    JOIN ` + entityid.JobPhase + ` jp
           ON jp.job_id = jj.job_id AND jp.active AND jp.template_phase_id IS NOT NULL
    LEFT JOIN ` + entityid.JobTask + ` tk
           ON tk.job_phase_id = jp.id AND tk.active
    LEFT JOIN ` + entityid.TaskOutcome + ` tox
           ON tox.job_task_id = tk.id AND tox.active
    LEFT JOIN ` + entityid.SubscriptionGroupMember + ` gm
           ON gm.subscription_id = jj.subscription_id AND gm.client_id = jj.client_id
          AND gm.workspace_id = $1 AND gm.active
    GROUP BY jj.template_id, gm.subscription_group_id, jp.template_phase_id
),
tp AS (
    SELECT template_id, template_phase_id,
           MIN(min_rank)     AS min_rank,
           MAX(max_rank)     AS max_rank,
           BOOL_OR(has_data) AS has_data
    FROM pa
    GROUP BY template_id, template_phase_id
),
ta AS (
    SELECT template_id,
           COUNT(*) FILTER (WHERE has_data AND min_rank = 4) AS published_count,
           COUNT(*) FILTER (WHERE has_data)                  AS phase_count,
           MIN(min_rank) FILTER (WHERE has_data)             AS lowest_rank,
           COALESCE(BOOL_OR(min_rank <> max_rank) FILTER (WHERE has_data), false) AS mixed_attention
    FROM tp
    GROUP BY template_id
),
ga AS (
    SELECT template_id, group_id,
           COUNT(*) FILTER (WHERE has_data AND min_rank = 4) AS published_count,
           COUNT(*) FILTER (WHERE has_data)                  AS phase_count,
           MIN(min_rank) FILTER (WHERE has_data)             AS lowest_rank,
           COALESCE(BOOL_OR(min_rank <> max_rank) FILTER (WHERE has_data), false) AS mixed_attention
    FROM pa
    WHERE group_id IS NOT NULL
    GROUP BY template_id, group_id
)`
}

// jobTemplateSummarySelectFrom is the DELIVERY SELECT + FROM over the two CTEs
// (the body of the `base` CTE). It hash-joins jj×dd on (subscription_id,
// client_id, output_product_id/product_id) — restoring the original
// seat/plan-product ↔ template-output match — then joins subscription_group
// (sg), price_schedule (ps, LEFT) and product (op, LEFT). Every table with a
// workspace_id column is bound to $1; product_plan and "user" (inside dd) have
// none and are bound transitively. The approval preaggregates do NOT appear
// here — they join AFTER the delivery aggregation (jobTemplateSummary-
// ApprovalSelect), per the codex-tandem contract.
func jobTemplateSummarySelectFrom() string {
	return `SELECT
    jj.template_id                 AS job_template_id,
    jj.template_name               AS job_template_name,
    sg.id                          AS subscription_group_id,
    sg.name                        AS subscription_group_name,
    dd.staff_id                    AS staff_id,
    COALESCE(NULLIF(TRIM(COALESCE(dd.first_name, '') || ' ' || COALESCE(dd.last_name, '')), ''), dd.staff_id) AS staff_name,
    COUNT(DISTINCT jj.job_id)      AS job_count,
    ps.id                          AS price_schedule_id,
    ps.name                        AS price_schedule_name,
    jj.output_product_id           AS output_product_id,
    op.name                        AS output_product_name,
    jj.job_category_id             AS job_category_id
FROM jj
JOIN dd
       ON dd.subscription_id = jj.subscription_id
      AND dd.client_id = jj.client_id
      AND dd.product_id = jj.output_product_id
JOIN ` + entityid.SubscriptionGroup + ` sg
       ON sg.id = dd.subscription_group_id AND sg.workspace_id = $1 AND sg.active
LEFT JOIN ` + entityid.PriceSchedule + ` ps
       ON ps.id = sg.price_schedule_id AND ps.workspace_id = $1
LEFT JOIN ` + entityid.Product + ` op
       ON op.id = jj.output_product_id AND op.workspace_id = $1`
}

// jobTemplateSummaryApprovalSelect is the TOP-LEVEL SELECT: the delivery
// aggregation (`base`, one row per (template, group, staff, schedule, product))
// LEFT-joined to the two already-collapsed approval roll-ups — ta unique per
// template_id (fields 14-17) and ga unique per (template_id, group_id) (fields
// 18-21). Joining ONLY AFTER the delivery aggregation is the codex-tandem
// contract ("join that one-row/template result only after delivery
// aggregation"): the join keys are unique per CTE row and base is already at
// its final grain, so job_count/deliverer fanout is impossible AND the join
// touches ~310 aggregated rows instead of ~8.5k pre-aggregation rows (the
// single-level form measured 1.04s vs 0.31s on education1 — the planner
// nested-looped the aggregates under its rows=1 estimate).
//
// A template with no approval rows LEFT-joins to NULL → counts COALESCE to 0
// and the ranks scan NULL → UNSPECIFIED (neutral not-started default). The
// ORDER BY keys are the LOCKED view order (group name, template name,
// staff_name, staff_id) — the SAME key semantics as the pre-P4 statement,
// referenced through base's output aliases because the sort now sits above the
// wrapper.
func jobTemplateSummaryApprovalSelect() string {
	return `SELECT
    base.*,
    COALESCE(ta.published_count, 0)        AS published_count,
    COALESCE(ta.phase_count, 0)            AS phase_count,
    ta.lowest_rank                         AS lowest_rank,
    COALESCE(ta.mixed_attention, false)    AS mixed_attention,
    COALESCE(ga.published_count, 0)        AS group_published_count,
    COALESCE(ga.phase_count, 0)            AS group_phase_count,
    ga.lowest_rank                         AS group_lowest_rank,
    COALESCE(ga.mixed_attention, false)    AS group_mixed_attention
FROM base
LEFT JOIN ta
       ON ta.template_id = base.job_template_id
LEFT JOIN ga
       ON ga.template_id = base.job_template_id AND ga.group_id = base.subscription_group_id
ORDER BY base.subscription_group_name, base.job_template_name, base.staff_name, base.staff_id`
}

// jobTemplateSummaryGroupOrder is the GROUP BY + ORDER BY tail. The grain is one
// row per (template, group, staff, schedule, product); a merged, multi-deliverer
// template produces one row per staff (collateDeliverySummaries folds them into one
// summary carrying all deliverers). ORDER BY group name then template name is the
// LOCKED view order; the trailing staff_name, dd.staff_id keys make the per-template
// DELIVERER order deterministic (a stable multi-name render). The leading
// `sg.name, jj.template_name` prefix is unchanged, so a template's rows stay contiguous.
// jj.job_category_id (R9 W-A1) is appended to GROUP BY only: it is functionally
// determined by jj.template_id (one nullable job_category FK per template), so it
// widens each row without fanning it out — row grain UNCHANGED. This is the
// `base` CTE's GROUP BY (R7 P4 moved the delivery aggregation into `base`); the
// ORDER BY now lives at the top level (jobTemplateSummaryApprovalSelect) with
// byte-identical key SEMANTICS (group name, template name, staff_name, staff_id)
// because the approval roll-ups join above this aggregation.
func jobTemplateSummaryGroupOrder() string {
	return `GROUP BY jj.template_id, jj.template_name, sg.id, sg.name, dd.staff_id, dd.first_name, dd.last_name,
         ps.id, ps.name, jj.output_product_id, op.name, jj.job_category_id`
}
