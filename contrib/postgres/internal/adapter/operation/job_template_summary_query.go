//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/lib/pq"

	summarypb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/job_template_summary"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
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

// classEdgeEligibilityLivePredicate is the eligibility-liveness gate the dd CTE's
// class-edge branch (b) appends to its WHERE. It is the courses-fold twin of
// principalscope.go's classEdgeEligibilityLive (audit M5-G5, the two M5
// consumers) and must stay in step with it.
//
// Branch (b) resolves the delivering staff as COALESCE(pps.staff_id, e.staff_id)
// over a LEFT JOIN on the edge's product_plan_staff eligibility link (f13). That
// COALESCE alone cannot distinguish two states:
//
//   - e.product_plan_staff_id IS NULL — no v2 link (pre-M3 rows, and any future
//     unlinked row). The LEFT JOIN yields NULL and the legacy f10 fallback is the
//     CORRECT answer. Preserved.
//   - e.product_plan_staff_id IS NOT NULL AND NOT pps.active — the eligibility was
//     REVOKED. Without this predicate the fold keeps naming that staff member as
//     the class's teacher-of-record, because COALESCE falls through to the still
//     dual-written legacy f10 column. Pushing "AND pps.active" into the LEFT JOIN
//     condition does NOT fix it — it produces exactly that fall-through.
//
// It keys on e.product_plan_staff_id, a column of the DRIVING table that the
// outer join can never null out. A linked-but-missing pps row yields NULL (not
// TRUE) and is dropped: fail-closed.
//
// Placement is the OUTER WHERE, deliberately NOT the class_primary_edges CTE
// that picks the deterministic primary. Its DISTINCT ON pick rule is mirrored
// verbatim by fayna's fetchClassEdgeTeachers; adding the predicate there would
// change WHICH edge wins and desynchronise the two. Here it only suppresses
// attribution for the already-picked edge. Consequence to know: dd is
// INNER-joined to jj, so a section whose only picked primary has a revoked
// eligibility (and no subscription_seat) drops out of the courses list rather
// than showing a stale teacher. Zero live rows are in that state today (all 108 product_plan_staff
// rows are active).
const classEdgeEligibilityLivePredicate = "AND (e.product_plan_staff_id IS NULL OR pps.active)"

// maxJobTemplateSummaryLimit caps a requested page size (the common
// PaginationRequest documents "max 100"). Same cap semantics as the sibling
// list adapters.
const maxJobTemplateSummaryLimit int32 = 100

var jobTemplateSummarySortExpressions = map[string]string{
	"name":      "job_template_name",
	"group":     "subscription_group_name",
	"deliverer": "array_to_string(staff_names, ', ')",
	"items":     "job_count",
	"schedule":  "price_schedule_name",
}

var jobTemplateSummarySearchExpressions = map[string]string{
	"name":      "job_template_name",
	"group":     "subscription_group_name",
	"deliverer": "array_to_string(staff_names, ' ')",
	"items":     "job_count::text",
	"schedule":  "price_schedule_name",
}

var defaultJobTemplateSummarySearchFields = []string{"name", "group", "deliverer", "items", "schedule"}

type jobTemplateSummaryQueryOptions struct {
	jobCategoryID           string
	search                  *commonpb.SearchRequest
	sort                    *commonpb.SortRequest
	priceScheduleActive     bool
	includeTemplateFallback bool
}

type jobCategorySummaryCountJSON struct {
	JobCategoryID string `json:"job_category_id"`
	SummaryCount  int64  `json:"summary_count"`
}

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

	limit, offset, err := paginationBounds(req.GetPagination())
	if err != nil {
		return nil, fmt.Errorf("job_template_summary: pagination: %w", err)
	}

	// The principalscope clause is spliced by the pure builder at the correct
	// placeholder index (the builder computes the start param and calls this fn).
	scopeFn := func(startParam int) (string, []any) {
		if _, applies := principalscope.StaffRowScope(ctx); !applies {
			// non-staff / no identity: no row narrowing (admin sees all).
			return "", nil
		}
		return principalscope.StaffReachableJobClause(ctx, "j", startParam)
	}

	stmt, args, err := buildListJobTemplateSummariesRequestSQL(
		workspaceID, req.GetStatus(), req.GetSubscriptionGroupId(),
		limit, offset,
		jobTemplateSummaryQueryOptions{
			jobCategoryID:           req.GetJobCategoryId(),
			search:                  req.GetSearch(),
			sort:                    req.GetSort(),
			priceScheduleActive:     req.GetPriceScheduleActive(),
			includeTemplateFallback: req.GetIncludeTemplateFallback(),
		},
		scopeFn,
	)
	if err != nil {
		return nil, fmt.Errorf("job_template_summary: request query: %w", err)
	}

	rows, err := a.db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("job_template_summary: query: %w", err)
	}
	defer rows.Close()

	// Metadata is repeated beside every result row. The final SQL is anchored on
	// its singleton metadata CTE and LEFT JOINs page rows, so an empty or
	// out-of-range page still yields one row with exact totals/category counts.
	var scanned []summaryScanRow
	var totalItems int64
	var categoryCountsJSON []byte
	metadataSeen := false
	for rows.Next() {
		var (
			templateID, templateName sql.NullString
			groupID, groupName       sql.NullString
			staffIDs, staffNames     []string
			jobCount                 sql.NullInt64
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
			publishedCount, phaseCount           sql.NullInt64
			lowestRank                           sql.NullInt64
			mixedAttention                       sql.NullBool
			groupPublishedCount, groupPhaseCount sql.NullInt64
			groupLowestRank                      sql.NullInt64
			groupMixedAttention                  sql.NullBool
			templateGrainFallback                sql.NullBool
			rowTotalItems                        int64
			rowCategoryCountsJSON                []byte
		)
		if err := rows.Scan(
			&templateID, &templateName,
			&groupID, &groupName,
			pq.Array(&staffIDs), pq.Array(&staffNames),
			&jobCount,
			&priceScheduleID, &priceScheduleName,
			&outputProductID, &outputProductName,
			&jobCategoryID,
			&publishedCount, &phaseCount, &lowestRank, &mixedAttention,
			&groupPublishedCount, &groupPhaseCount, &groupLowestRank, &groupMixedAttention,
			&templateGrainFallback,
			&rowTotalItems, &rowCategoryCountsJSON,
		); err != nil {
			return nil, fmt.Errorf("job_template_summary: scan: %w", err)
		}
		if !metadataSeen {
			totalItems = rowTotalItems
			categoryCountsJSON = append([]byte(nil), rowCategoryCountsJSON...)
			metadataSeen = true
		} else if rowTotalItems != totalItems || string(rowCategoryCountsJSON) != string(categoryCountsJSON) {
			return nil, fmt.Errorf("job_template_summary: inconsistent repeated metadata")
		}
		if !templateID.Valid {
			continue
		}
		jobCount32, err := summaryCountInt32("job_count", jobCount)
		if err != nil {
			return nil, err
		}
		publishedCount32, err := summaryCountInt32("published_count", publishedCount)
		if err != nil {
			return nil, err
		}
		phaseCount32, err := summaryCountInt32("phase_count", phaseCount)
		if err != nil {
			return nil, err
		}
		groupPublishedCount32, err := summaryCountInt32("group_published_count", groupPublishedCount)
		if err != nil {
			return nil, err
		}
		groupPhaseCount32, err := summaryCountInt32("group_phase_count", groupPhaseCount)
		if err != nil {
			return nil, err
		}
		scanned = append(scanned, summaryScanRow{
			templateID: templateID.String, templateName: templateName.String,
			groupID: groupID.String, groupName: groupName.String,
			deliverers:            deliveryPairs(staffIDs, staffNames),
			jobCount:              jobCount32,
			priceScheduleID:       priceScheduleID.String,
			priceScheduleName:     priceScheduleName.String,
			outputProductID:       outputProductID.String,
			outputProductName:     outputProductName.String,
			jobCategoryID:         jobCategoryID.String,
			publishedCount:        publishedCount32,
			phaseCount:            phaseCount32,
			lowestRank:            nullRankToInt(lowestRank),
			mixedAttention:        mixedAttention.Bool,
			groupPublishedCount:   groupPublishedCount32,
			groupPhaseCount:       groupPhaseCount32,
			groupLowestRank:       nullRankToInt(groupLowestRank),
			groupMixedAttention:   groupMixedAttention.Bool,
			templateGrainFallback: templateGrainFallback.Bool,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("job_template_summary: rows: %w", err)
	}

	if !metadataSeen {
		return nil, fmt.Errorf("job_template_summary: query returned no metadata row")
	}
	if totalItems < 0 || totalItems > math.MaxInt32 {
		return nil, fmt.Errorf("job_template_summary: total_items %d exceeds response capacity", totalItems)
	}

	summaries := make([]*summarypb.JobTemplateSummary, 0, len(scanned))
	for _, row := range scanned {
		summaries = append(summaries, summaryFromScanRow(row))
	}

	var categoryCountRows []jobCategorySummaryCountJSON
	if err := json.Unmarshal(categoryCountsJSON, &categoryCountRows); err != nil {
		return nil, fmt.Errorf("job_template_summary: category counts metadata: %w", err)
	}
	categoryCounts := make([]*summarypb.JobCategorySummaryCount, 0, len(categoryCountRows))
	for _, count := range categoryCountRows {
		if count.SummaryCount < 0 || count.SummaryCount > math.MaxInt32 {
			return nil, fmt.Errorf("job_template_summary: category %q count %d exceeds response capacity", count.JobCategoryID, count.SummaryCount)
		}
		categoryCounts = append(categoryCounts, &summarypb.JobCategorySummaryCount{
			JobCategoryId: count.JobCategoryID,
			SummaryCount:  int32(count.SummaryCount),
		})
	}

	resp := &summarypb.ListJobTemplateSummariesResponse{
		Summaries:         summaries,
		Success:           true,
		JobCategoryCounts: categoryCounts,
	}
	if p := req.GetPagination(); p != nil && limit > 0 {
		page := int32(1)
		if off := p.GetOffset(); off != nil && off.GetPage() > 1 {
			page = off.GetPage()
		}
		totalPages := int32(0)
		if totalItems > 0 {
			totalPages = int32((totalItems + int64(limit) - 1) / int64(limit))
		}
		resp.Pagination = &commonpb.PaginationResponse{
			TotalItems:  int32(totalItems),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     page < totalPages,
			HasPrev:     page > 1,
		}
	}
	return resp, nil
}

func trimSummaryPage(summaries []*summarypb.JobTemplateSummary, limit int32) ([]*summarypb.JobTemplateSummary, bool) {
	if limit > 0 && int32(len(summaries)) > limit {
		return summaries[:limit], true
	}
	return summaries, false
}

// summaryScanRow is one SQL response-grain aggregate row.
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
	deliverers               []*summarypb.Deliverer
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
	templateGrainFallback                bool
}

// deliveryPairs joins the two ordered SQL arrays into the response deliverers.
// The arrays are produced by the same ARRAY_AGG ORDER BY expression; reject a
// malformed unequal pair fail-closed rather than inventing an attribution.
func deliveryPairs(ids, names []string) []*summarypb.Deliverer {
	if len(ids) != len(names) {
		return nil
	}
	out := make([]*summarypb.Deliverer, 0, len(ids))
	for i := range ids {
		if ids[i] == "" {
			continue
		}
		out = append(out, &summarypb.Deliverer{StaffId: ids[i], StaffName: names[i]})
	}
	return out
}

func summaryFromScanRow(r summaryScanRow) *summarypb.JobTemplateSummary {
	return &summarypb.JobTemplateSummary{
		JobTemplateId: r.templateID, JobTemplateName: r.templateName,
		SubscriptionGroupId: r.groupID, SubscriptionGroupName: r.groupName,
		JobCount: r.jobCount, PriceScheduleId: r.priceScheduleID,
		PriceScheduleName: r.priceScheduleName, OutputProductId: r.outputProductID,
		OutputProductName: r.outputProductName, JobCategoryId: r.jobCategoryID,
		Deliverers:     r.deliverers,
		PublishedCount: r.publishedCount, PhaseCount: r.phaseCount,
		LowestStatus: approvalRankToStatus(r.lowestRank), MixedAttention: r.mixedAttention,
		GroupPublishedCount: r.groupPublishedCount, GroupPhaseCount: r.groupPhaseCount,
		GroupLowestStatus: approvalRankToStatus(r.groupLowestRank), GroupMixedAttention: r.groupMixedAttention,
		TemplateGrainFallback: r.templateGrainFallback,
	}
}

func summaryCountInt32(name string, value sql.NullInt64) (int32, error) {
	if !value.Valid {
		return 0, nil
	}
	if value.Int64 < 0 || value.Int64 > math.MaxInt32 {
		return 0, fmt.Errorf("job_template_summary: %s %d exceeds response capacity", name, value.Int64)
	}
	return int32(value.Int64), nil
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
				PublishedCount:        r.publishedCount,
				PhaseCount:            r.phaseCount,
				LowestStatus:          approvalRankToStatus(r.lowestRank),
				MixedAttention:        r.mixedAttention,
				GroupPublishedCount:   r.groupPublishedCount,
				GroupPhaseCount:       r.groupPhaseCount,
				GroupLowestStatus:     approvalRankToStatus(r.groupLowestRank),
				GroupMixedAttention:   r.groupMixedAttention,
				TemplateGrainFallback: r.templateGrainFallback,
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

// paginationBounds preserves the Courses compatibility contract where a nil or
// non-positive limit means "no pagination". Positive-limit requests use the
// shared finite-work validator; the shallow copy preserves caller-owned proto
// state while retaining the historical clamp-to-100 behavior.
func paginationBounds(p *commonpb.PaginationRequest) (limit, offset int32, err error) {
	if p == nil || p.GetLimit() <= 0 {
		return 0, 0, nil
	}
	bounded := *p
	if bounded.GetLimit() > maxJobTemplateSummaryLimit {
		bounded.Limit = maxJobTemplateSummaryLimit
	}
	limit, offset, _, err = postgresCore.BoundedOffsetPagination(&bounded, maxJobTemplateSummaryLimit)
	if err != nil {
		return 0, 0, err
	}
	return limit, offset, nil
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
//	dd  MATERIALIZED CTE = a UNION of two deliverer/group branches, both keyed by
//	     (subscription_id, client_id, product_id) and emitting the identical
//	     7-column shape:
//	       (a) subscription_seat ⋈ product_plan ⋈ staff ⋈ "user" (LEFT) ⋈
//	           subscription_group_member. sgm is correlated to the seat on
//	           (subscription_id, client_id): in the original both sgm AND ss bind
//	           to the job's (origin_id, client_id), so binding them to each other
//	           is transitively identical once jj joins in.
//	       (b) subscription_group_product_plan_staff (the class edge) ⋈
//	           subscription_group_member ⋈ product_plan ⋈ product_plan_staff
//	           (LEFT, the v2 eligibility link) ⋈ staff ⋈ "user" (LEFT), filtered
//	           role='primary' (teacher-of-record). This covers sections that have
//	           class-edge servicers but zero seats (AY-2627); without it jj⋈dd's
//	           INNER join drops every such job (C11). UNION (not UNION ALL)
//	           dedupes a (sub, client, product, staff) pair reachable via both.
//	           Staff resolution: COALESCE(pps.staff_id, e.staff_id) — v2 f13-linked
//	           preferred, legacy f10 fallback for rows predating the M3 link-up.
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
	stmt, args, err := buildListJobTemplateSummariesRequestSQL(
		workspaceID, status, groupID, limit, offset,
		jobTemplateSummaryQueryOptions{}, scopeFn,
	)
	if err != nil {
		return "", nil
	}
	return stmt, args
}

func buildListJobTemplateSummariesRequestSQL(
	workspaceID, status, groupID string,
	limit, offset int32,
	options jobTemplateSummaryQueryOptions,
	scopeFn func(startParam int) (clause string, args []any),
) (stmt string, args []any, err error) {
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
	var outerClauses []string
	if groupID != "" {
		outerClauses = append(outerClauses, fmt.Sprintf("sg.id = $%d", p))
		args = append(args, groupID)
		p++
	}
	if options.priceScheduleActive {
		outerClauses = append(outerClauses, "ps.active")
	}
	outerWhere := ""
	if len(outerClauses) > 0 {
		outerWhere = "\nWHERE " + strings.Join(outerClauses, " AND ")
	}

	var selectedClauses []string
	if options.jobCategoryID != "" {
		selectedClauses = append(selectedClauses, fmt.Sprintf("job_category_id = $%d", p))
		args = append(args, options.jobCategoryID)
		p++
	}
	searchClause, searchArgs, nextParam, err := jobTemplateSummarySearchClause(options.search, p)
	if err != nil {
		return "", nil, err
	}
	if searchClause != "" {
		selectedClauses = append(selectedClauses, searchClause)
		args = append(args, searchArgs...)
		p = nextParam
	}
	selectedWhere := ""
	if len(selectedClauses) > 0 {
		selectedWhere = "\n    WHERE " + strings.Join(selectedClauses, " AND ")
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
	pageOrder, finalOrder, err := jobTemplateSummaryOrderBy(options.sort)
	if err != nil {
		return "", nil, err
	}
	includeFallback := options.includeTemplateFallback &&
		status == "JOB_STATUS_ACTIVE" && groupID == "" && !options.priceScheduleActive

	stmt = jobTemplateSummaryCTEs(jjWhere) + ",\ndelivery AS MATERIALIZED (\n" +
		jobTemplateSummarySelectFrom() + outerWhere + "\n),\n" +
		jobTemplateSummaryDeliveryFold() + ",\n" +
		jobTemplateSummaryUniverse(includeFallback) + ",\n" +
		jobTemplateSummaryMetadata(selectedWhere) + ",\n" +
		jobTemplateSummaryPageBase(pageOrder, limitClause) + ",\n" +
		jobTemplateSummaryApprovalCTEs() + "\n" +
		jobTemplateSummaryApprovalSelect(finalOrder)
	return stmt, args, nil
}

func jobTemplateSummarySearchClause(search *commonpb.SearchRequest, startParam int) (string, []any, int, error) {
	pattern, err := postgresCore.BoundedContainsSearchPattern(search)
	if err != nil {
		return "", nil, startParam, fmt.Errorf("bounded summary search: %w", err)
	}
	if pattern == "" {
		return "", nil, startParam, nil
	}

	fields := defaultJobTemplateSummarySearchFields
	if requested := search.GetOptions().GetSearchFields(); len(requested) > 0 {
		if len(requested) > len(jobTemplateSummarySearchExpressions) {
			return "", nil, startParam, fmt.Errorf("too many summary search fields: %d", len(requested))
		}
		fields = requested
	}
	seen := make(map[string]struct{}, len(fields))
	clauses := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		expression, ok := jobTemplateSummarySearchExpressions[field]
		if !ok {
			return "", nil, startParam, fmt.Errorf("unknown summary search field %q", field)
		}
		if _, duplicate := seen[field]; duplicate {
			continue
		}
		seen[field] = struct{}{}
		clauses = append(clauses, fmt.Sprintf("COALESCE((%s)::text, '') ILIKE $%d ESCAPE '\\'", expression, startParam))
	}
	if len(clauses) == 0 {
		return "", nil, startParam, fmt.Errorf("summary search has no usable fields")
	}
	return "(" + strings.Join(clauses, " OR ") + ")", []any{pattern}, startParam + 1, nil
}

func jobTemplateSummaryOrderBy(sortRequest *commonpb.SortRequest) (pageOrder, finalOrder string, err error) {
	pageParts, err := jobTemplateSummarySortParts(sortRequest, "")
	if err != nil {
		return "", "", err
	}
	finalParts, err := jobTemplateSummarySortParts(sortRequest, "page_enriched.")
	if err != nil {
		return "", "", err
	}
	return strings.Join(pageParts, ", "), strings.Join(finalParts, ", "), nil
}

func jobTemplateSummarySortParts(sortRequest *commonpb.SortRequest, prefix string) ([]string, error) {
	tieBreakers := []string{
		prefix + "subscription_group_id ASC NULLS FIRST",
		prefix + "job_template_id ASC",
		prefix + "price_schedule_id ASC NULLS FIRST",
		prefix + "output_product_id ASC NULLS FIRST",
	}
	if sortRequest == nil || len(sortRequest.GetFields()) == 0 {
		return append([]string{
			prefix + "subscription_group_name ASC NULLS FIRST",
			prefix + "job_template_name ASC",
		}, tieBreakers...), nil
	}
	if len(sortRequest.GetFields()) > 4 {
		return nil, fmt.Errorf("too many summary sort fields: %d", len(sortRequest.GetFields()))
	}

	parts := make([]string, 0, len(sortRequest.GetFields())+len(tieBreakers))
	for i, field := range sortRequest.GetFields() {
		if field == nil {
			return nil, fmt.Errorf("summary sort field %d is nil", i)
		}
		name := strings.TrimSpace(field.GetField())
		if _, ok := jobTemplateSummarySortExpressions[name]; !ok {
			return nil, fmt.Errorf("unknown summary sort field %q", name)
		}
		expression := jobTemplateSummarySortExpression(name, prefix)
		direction := "ASC"
		switch field.GetDirection() {
		case commonpb.SortDirection_ASC:
		case commonpb.SortDirection_DESC:
			direction = "DESC"
		default:
			return nil, fmt.Errorf("invalid summary sort direction %d", field.GetDirection())
		}
		nullOrder := "NULLS FIRST"
		if field.GetNullOrder() == commonpb.NullOrder_NULLS_LAST {
			nullOrder = "NULLS LAST"
		}
		parts = append(parts, expression+" "+direction+" "+nullOrder)
	}
	return append(parts, tieBreakers...), nil
}

func jobTemplateSummarySortExpression(field, prefix string) string {
	switch field {
	case "deliverer":
		return "array_to_string(" + prefix + "staff_names, ', ')"
	default:
		return prefix + jobTemplateSummarySortExpressions[field]
	}
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
-- Pick the newest active primary class edge set-wise, once per
-- (subscription_group_id, product_plan_id).  Do not apply the eligibility
-- liveness gate here: a newly revoked edge must suppress attribution rather
-- than make an older primary edge appear current.
class_primary_edges AS MATERIALIZED (
    SELECT DISTINCT ON (e.subscription_group_id, e.product_plan_id)
        e.id,
        e.subscription_group_id,
        e.product_plan_id,
        e.staff_id,
        e.product_plan_staff_id
    FROM ` + entityid.SubscriptionGroupProductPlanStaff + ` e
    WHERE e.active AND e.workspace_id = $1 AND e.role = 'primary'
    ORDER BY e.subscription_group_id, e.product_plan_id,
             e.date_created DESC, e.id DESC
),
dd AS MATERIALIZED (
    -- Branch (a): the SUBSCRIPTION_SEAT deliverer/group side (the original dd).
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
    UNION
    -- Branch (b): class-edge (sgpps) deliverers, role primary only (C11 — full
    -- rationale in the jobTemplateSummaryCTEs doc comment). Emits branch (a)'s
    -- exact 7-column shape (member-sourced keys + the edge plan product). Staff
    -- resolution is v2-native (docs/plan/20260724-section-assignment-merged
    -- espyna.md §1b/M5, consumer 2 of 2): the edge's product_plan_staff
    -- eligibility link (f13, "pps") supplies pps.staff_id, COALESCEd with the
    -- edge's own legacy staff_id (f10) as a fallback for rows that predate the
    -- M3 link-up (the LEFT JOIN keeps such rows resolvable instead of dropping
    -- them) -- the fallback retires at M7 alongside the rest of legacy f8/f9/f10.
    -- The WHERE also carries classEdgeEligibilityLivePredicate (audit M5-G5): a
    -- LINKED-but-REVOKED eligibility (product_plan_staff_id set, pps.active
    -- false) must NOT fall through to legacy f10 and keep attributing a staff
    -- member who is no longer eligible to deliver this plan.
    SELECT
        m.subscription_id        AS subscription_id,
        m.client_id              AS client_id,
        pl.product_id            AS product_id,
        st.id                    AS staff_id,
        u.first_name             AS first_name,
        u.last_name              AS last_name,
        m.subscription_group_id  AS subscription_group_id
    FROM class_primary_edges e
    JOIN ` + entityid.SubscriptionGroupMember + ` m
           ON m.subscription_group_id = e.subscription_group_id AND m.active
    JOIN ` + entityid.ProductPlan + ` pl
           ON pl.id = e.product_plan_id
    LEFT JOIN ` + entityid.ProductPlanStaff + ` pps
           ON pps.id = e.product_plan_staff_id
    JOIN ` + entityid.Staff + ` st
           ON st.id = COALESCE(pps.staff_id, e.staff_id) AND st.workspace_id = $1
    LEFT JOIN "` + entityid.User + `" u
           ON u.id = st.user_id AND u.active
    -- CF-3: class_primary_edges has already picked ONE primary edge for each
    -- (group, product_plan), deterministically (newest date_created, id breaks
    -- ties). Apply the eligibility gate only after that pick so a newer revoked
    -- edge suppresses attribution rather than falling back to an older edge.
    WHERE TRUE
      ` + classEdgeEligibilityLivePredicate + `
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
	    jj.job_id                      AS job_id,
	    jj.template_id                 AS job_template_id,
    jj.template_name               AS job_template_name,
    sg.id                          AS subscription_group_id,
    sg.name                        AS subscription_group_name,
    dd.staff_id                    AS staff_id,
    COALESCE(NULLIF(TRIM(COALESCE(dd.first_name, '') || ' ' || COALESCE(dd.last_name, '')), ''), dd.staff_id) AS staff_name,
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

// jobTemplateSummaryDeliveryFold collapses the materialized delivery join to the
// public Courses row grain before approval joins, ordering, and pagination. The
// separate counts and deliverers CTEs prevent the job×staff join from making a
// staff appear once per roster job while preserving COUNT(DISTINCT job_id).
func jobTemplateSummaryDeliveryFold() string {
	return `counts AS (
    SELECT job_template_id, job_template_name, subscription_group_id, subscription_group_name,
           price_schedule_id, price_schedule_name, output_product_id, output_product_name,
           job_category_id, COUNT(DISTINCT job_id) AS job_count
    FROM delivery
    GROUP BY job_template_id, job_template_name, subscription_group_id, subscription_group_name,
             price_schedule_id, price_schedule_name, output_product_id, output_product_name,
             job_category_id
),
delivery_staff AS (
    SELECT DISTINCT job_template_id, subscription_group_id, price_schedule_id,
           output_product_id, staff_id, staff_name
    FROM delivery
),
deliverers AS (
    SELECT job_template_id, subscription_group_id, price_schedule_id, output_product_id,
           ARRAY_AGG(staff_id ORDER BY staff_name, staff_id) AS staff_ids,
           ARRAY_AGG(staff_name ORDER BY staff_name, staff_id) AS staff_names
    FROM delivery_staff
    GROUP BY job_template_id, subscription_group_id, price_schedule_id, output_product_id
),
base AS (
    SELECT c.*, d.staff_ids, d.staff_names
    FROM counts c
    JOIN deliverers d
      ON d.job_template_id = c.job_template_id
     AND d.subscription_group_id = c.subscription_group_id
     AND d.price_schedule_id IS NOT DISTINCT FROM c.price_schedule_id
     AND d.output_product_id IS NOT DISTINCT FROM c.output_product_id
)`
}

func jobTemplateSummaryUniverse(includeTemplateFallback bool) string {
	if !includeTemplateFallback {
		return `summary_universe AS MATERIALIZED (
    SELECT base.*, false AS template_grain_fallback
    FROM base
)`
	}
	return `fallback_base AS MATERIALIZED (
    SELECT
        jt.id                  AS job_template_id,
        jt.name                AS job_template_name,
        ''::text               AS subscription_group_id,
        ''::text               AS subscription_group_name,
        NULL::text             AS price_schedule_id,
        NULL::text             AS price_schedule_name,
        NULL::text             AS output_product_id,
        NULL::text             AS output_product_name,
        jt.job_category_id     AS job_category_id,
        0::bigint              AS job_count,
        ARRAY[]::text[]        AS staff_ids,
        ARRAY[]::text[]        AS staff_names,
        true                   AS template_grain_fallback
    FROM ` + entityid.JobTemplate + ` jt
    WHERE jt.workspace_id = $1 AND jt.active
      AND NOT EXISTS (
          SELECT 1
          FROM base
          WHERE base.job_category_id IS NOT DISTINCT FROM jt.job_category_id
      )
),
summary_universe AS MATERIALIZED (
    SELECT base.*, false AS template_grain_fallback
    FROM base
    UNION ALL
    SELECT * FROM fallback_base
)`
}

func jobTemplateSummaryMetadata(selectedWhere string) string {
	return `category_counts AS MATERIALIZED (
    SELECT COALESCE(job_category_id, '') AS job_category_id,
           COUNT(*) AS summary_count
    FROM summary_universe
    GROUP BY COALESCE(job_category_id, '')
),
selected_rows AS MATERIALIZED (
    SELECT *
    FROM summary_universe` + selectedWhere + `
),
metadata AS MATERIALIZED (
    SELECT
        (SELECT COUNT(*) FROM selected_rows) AS total_items,
        COALESCE(
            (SELECT jsonb_agg(
                jsonb_build_object(
                    'job_category_id', category_counts.job_category_id,
                    'summary_count', category_counts.summary_count
                ) ORDER BY category_counts.job_category_id
            ) FROM category_counts),
            '[]'::jsonb
        ) AS category_counts_json
)`
}

func jobTemplateSummaryPageBase(orderBy, limitClause string) string {
	return `page_base AS MATERIALIZED (
    SELECT selected_rows.*
    FROM selected_rows
    ORDER BY ` + orderBy + limitClause + `
)`
}

// jobTemplateSummaryApprovalCTEs computes approval only for rows selected into
// page_base. data_phases materializes the unique set of phases bearing an active
// outcome once, so template_phase and group_phase can reuse it without correlated
// probes or task/outcome fanout before their MIN/MAX rollups. ta and ga are
// MATERIALIZED planner fences: page_base's live cardinality was severely
// underestimated, which otherwise let PostgreSQL re-execute final GroupAggregates
// under each page row.
func jobTemplateSummaryApprovalCTEs() string {
	return `candidate_templates AS (
    SELECT DISTINCT job_template_id AS template_id FROM page_base
),
candidate_groups AS (
    SELECT DISTINCT job_template_id AS template_id, subscription_group_id AS group_id FROM page_base
),
data_phases AS MATERIALIZED (
    SELECT DISTINCT tk.job_phase_id
    FROM ` + entityid.JobTask + ` tk
    JOIN ` + entityid.TaskOutcome + ` tox ON tox.job_task_id = tk.id AND tox.active
    WHERE tk.active
),
template_phase AS (
    SELECT ct.template_id, jp.template_phase_id,
           MIN(` + approvalStatusRankCASE + `) AS min_rank,
           MAX(` + approvalStatusRankCASE + `) AS max_rank,
           BOOL_OR(data_phases.job_phase_id IS NOT NULL) AS has_data
    FROM candidate_templates ct
    JOIN jj ON jj.template_id = ct.template_id
    JOIN ` + entityid.JobPhase + ` jp
           ON jp.job_id = jj.job_id AND jp.active AND jp.template_phase_id IS NOT NULL
    LEFT JOIN data_phases ON data_phases.job_phase_id = jp.id
    GROUP BY ct.template_id, jp.template_phase_id
),
ta AS MATERIALIZED (
    SELECT template_id,
           COUNT(*) FILTER (WHERE has_data AND min_rank = 4) AS published_count,
           COUNT(*) FILTER (WHERE has_data)                  AS phase_count,
           MIN(min_rank) FILTER (WHERE has_data)             AS lowest_rank,
           COALESCE(BOOL_OR(min_rank <> max_rank) FILTER (WHERE has_data), false) AS mixed_attention
    FROM template_phase
    GROUP BY template_id
),
group_phase AS (
    SELECT cg.template_id, cg.group_id, jp.template_phase_id,
           MIN(` + approvalStatusRankCASE + `) AS min_rank,
           MAX(` + approvalStatusRankCASE + `) AS max_rank,
           BOOL_OR(data_phases.job_phase_id IS NOT NULL) AS has_data
    FROM candidate_groups cg
    JOIN jj ON jj.template_id = cg.template_id
    JOIN ` + entityid.SubscriptionGroupMember + ` gm
           ON gm.subscription_group_id = cg.group_id
          AND gm.subscription_id = jj.subscription_id AND gm.client_id = jj.client_id
          AND gm.workspace_id = $1 AND gm.active
    JOIN ` + entityid.JobPhase + ` jp
           ON jp.job_id = jj.job_id AND jp.active AND jp.template_phase_id IS NOT NULL
    LEFT JOIN data_phases ON data_phases.job_phase_id = jp.id
    GROUP BY cg.template_id, cg.group_id, jp.template_phase_id
),
ga AS MATERIALIZED (
    SELECT template_id, group_id,
           COUNT(*) FILTER (WHERE has_data AND min_rank = 4) AS published_count,
           COUNT(*) FILTER (WHERE has_data)                  AS phase_count,
           MIN(min_rank) FILTER (WHERE has_data)             AS lowest_rank,
           COALESCE(BOOL_OR(min_rank <> max_rank) FILTER (WHERE has_data), false) AS mixed_attention
    FROM group_phase
    GROUP BY template_id, group_id
)`
}

// jobTemplateSummaryApprovalSelect is the TOP-LEVEL SELECT: the delivery
// aggregation (`base`, one row per (template, group, schedule, product))
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
// ORDER BY preserves the locked group/template presentation order and adds the
// complete response identity as deterministic pagination tie-breakers.
func jobTemplateSummaryApprovalSelect(finalOrders ...string) string {
	finalOrder := "page_enriched.subscription_group_name ASC NULLS FIRST, " +
		"page_enriched.job_template_name ASC, " +
		"page_enriched.subscription_group_id ASC NULLS FIRST, " +
		"page_enriched.job_template_id ASC, " +
		"page_enriched.price_schedule_id ASC NULLS FIRST, " +
		"page_enriched.output_product_id ASC NULLS FIRST"
	if len(finalOrders) > 0 && finalOrders[0] != "" {
		finalOrder = finalOrders[0]
	}
	return `,
page_enriched AS MATERIALIZED (
    SELECT
	page_base.job_template_id,
	page_base.job_template_name,
	page_base.subscription_group_id,
	page_base.subscription_group_name,
	page_base.staff_ids,
	page_base.staff_names,
	page_base.job_count,
	page_base.price_schedule_id,
	page_base.price_schedule_name,
	page_base.output_product_id,
	page_base.output_product_name,
	page_base.job_category_id,
    COALESCE(ta.published_count, 0)        AS published_count,
    COALESCE(ta.phase_count, 0)            AS phase_count,
    ta.lowest_rank                         AS lowest_rank,
    COALESCE(ta.mixed_attention, false)    AS mixed_attention,
    COALESCE(ga.published_count, 0)        AS group_published_count,
    COALESCE(ga.phase_count, 0)            AS group_phase_count,
    ga.lowest_rank                         AS group_lowest_rank,
    COALESCE(ga.mixed_attention, false)    AS group_mixed_attention,
    page_base.template_grain_fallback
    FROM page_base
    LEFT JOIN ta
           ON ta.template_id = page_base.job_template_id
    LEFT JOIN ga
           ON ga.template_id = page_base.job_template_id AND ga.group_id = page_base.subscription_group_id
)
SELECT
    page_enriched.job_template_id,
    page_enriched.job_template_name,
    page_enriched.subscription_group_id,
    page_enriched.subscription_group_name,
    page_enriched.staff_ids,
    page_enriched.staff_names,
    page_enriched.job_count,
    page_enriched.price_schedule_id,
    page_enriched.price_schedule_name,
    page_enriched.output_product_id,
    page_enriched.output_product_name,
    page_enriched.job_category_id,
    page_enriched.published_count,
    page_enriched.phase_count,
    page_enriched.lowest_rank,
    page_enriched.mixed_attention,
    page_enriched.group_published_count,
    page_enriched.group_phase_count,
    page_enriched.group_lowest_rank,
    page_enriched.group_mixed_attention,
    page_enriched.template_grain_fallback,
    metadata.total_items,
    metadata.category_counts_json
FROM metadata
LEFT JOIN page_enriched ON TRUE
ORDER BY ` + finalOrder
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
