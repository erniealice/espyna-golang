//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/lib/pq"

	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	matrixpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/outcome_matrix"

	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	"github.com/erniealice/espyna-golang/shared/placeholder"
)

// Table names come from the entityid registry (single source of names).
// product_plan / price_plan carry no workspace_id: the tenant boundary for this
// chain is the workspace-verified job (query 1) and the workspace_id on the
// link (query 3) and set (query 4) rows.
const (
	ratingDescriptionSetTable            = entityid.RatingDescriptionSet
	ratingDescriptionSetEntryTable       = entityid.RatingDescriptionSetEntry
	ratingDescriptionSetProductPlanTable = entityid.RatingDescriptionSetProductPlan
)

const (
	versionStatusPublishedValue  = "VERSION_STATUS_PUBLISHED"
	versionStatusDeprecatedValue = "VERSION_STATUS_DEPRECATED"
)

// ResolveCellRatingDescriptions is the per-cell rubric-text resolver
// (schema-proposal.md §4, espyna-golang.md "Resolver contract"). It answers,
// for each requested (job_id, job_task_id, outcome_criteria_id) cell, whether
// a PUBLISHED/DEPRECATED rating_description_set is linked to the cell's
// product offering × academic year and, if so, that set's text for THIS
// criterion (every band, not just the currently-typed value — the caller
// matches by value; NO_ENTRY for a specific level is a client-side concept,
// never a server status).
//
// Workspace comes from the SESSION identity (identity.FromContext), never a
// request param — same fail-closed contract as GetOutcomeMatrix. Every
// requested cell gets EXACTLY ONE result (RD-66), including on a missing
// identity or an unregistered provider: this RPC is consulted on the grading
// save path, so an absent answer must never be mistaken for "no descriptions
// configured" (which is the distinct NO_LINK status).
func (a *PostgresOutcomeMatrixQuery) ResolveCellRatingDescriptions(
	ctx context.Context,
	req *matrixpb.ResolveCellRatingDescriptionsRequest,
) (*matrixpb.ResolveCellRatingDescriptionsResponse, error) {
	if req == nil || len(req.GetCells()) == 0 {
		return &matrixpb.ResolveCellRatingDescriptionsResponse{Success: true}, nil
	}

	id, ok := identity.FromContext(ctx)
	if !ok || id == nil || id.WorkspaceID == "" {
		return &matrixpb.ResolveCellRatingDescriptionsResponse{
			Results: failClosedResolutions(req.GetCells(),
				enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_UNRESOLVED_IDENTITY,
				"no session identity"),
			Success: true,
		}, nil
	}

	return a.resolveCellRatingDescriptionsFrom(ctx, a.db, req.GetCells(), id.WorkspaceID)
}

// failClosedResolutions builds one CellRatingResolution per cell, all sharing
// the given status/reason (RD-66: never fewer results than requested cells).
func failClosedResolutions(cells []*matrixpb.CellRatingRef, status enumspb.RatingDescriptionResolutionStatus, reason string) []*matrixpb.CellRatingResolution {
	out := make([]*matrixpb.CellRatingResolution, len(cells))
	for i, c := range cells {
		out[i] = &matrixpb.CellRatingResolution{Cell: c, Status: status, Reason: reason}
	}
	return out
}

// cellResolutionState tracks one requested cell across the batched resolution
// pipeline (ancestry -> product_plan -> link -> set -> entries). It carries
// just enough intermediate state (rating_scale_id from the slot's binding) to
// validate the set's scale at step 4 without a 6th query.
type cellResolutionState struct {
	cell   *matrixpb.CellRatingRef
	status enumspb.RatingDescriptionResolutionStatus
	setID  string
	reason string

	// carried between steps; empty once the cell is no longer "in flight"
	// (i.e. status has been set to something other than UNSPECIFIED-pending).
	productID     string
	variant       string // '' normalized, matches product_plan.product_variant_id via COALESCE(NULLIF(...,''),'')
	planID        string
	scheduleID    string
	productPlanID string
	ratingScaleID string

	resolved        bool // true once status is final and no further step should touch this cell
	descriptionsOut []*matrixpb.RatingDescription
}

// resolveCellRatingDescriptionsFrom runs the 5 bounded batch queries (jobs+
// ancestry+slot mode, product_plans, links, sets, entries — schema-proposal.md
// §4 / espyna-golang.md rule 6, one query per stage, keys deduplicated so the
// query count never scales with len(cells)), plus at most ONE placeholder-
// value query (schema-proposal.md §10) only when a resolved entry carries a
// `client.*` tag — never more than 6 queries per batch. queryer is the same DB/Tx seam
// outcomeMatrixRowsQueryer already establishes for loadColumnTreeFrom, so the
// rollback-only integration test can drive real SQL inside one transaction.
func (a *PostgresOutcomeMatrixQuery) resolveCellRatingDescriptionsFrom(
	ctx context.Context,
	queryer outcomeMatrixRowsQueryer,
	cells []*matrixpb.CellRatingRef,
	workspaceID string,
) (*matrixpb.ResolveCellRatingDescriptionsResponse, error) {
	states := make([]*cellResolutionState, len(cells))
	for i, c := range cells {
		states[i] = &cellResolutionState{cell: c}
	}

	if err := a.loadCellAncestryAndSlotMode(ctx, queryer, states, workspaceID); err != nil {
		return nil, fmt.Errorf("resolve_cell_rating_descriptions: ancestry/slot query: %w", err)
	}
	if err := a.resolveProductPlans(ctx, queryer, states); err != nil {
		return nil, fmt.Errorf("resolve_cell_rating_descriptions: product_plan query: %w", err)
	}
	if err := a.resolveRatingLinks(ctx, queryer, states, workspaceID); err != nil {
		return nil, fmt.Errorf("resolve_cell_rating_descriptions: link query: %w", err)
	}
	if err := a.resolveRatingSets(ctx, queryer, states, workspaceID); err != nil {
		return nil, fmt.Errorf("resolve_cell_rating_descriptions: set query: %w", err)
	}
	if err := a.resolveRatingEntries(ctx, queryer, states, workspaceID); err != nil {
		return nil, fmt.Errorf("resolve_cell_rating_descriptions: entries query: %w", err)
	}
	if err := a.renderRatingPlaceholders(ctx, queryer, states, workspaceID); err != nil {
		return nil, fmt.Errorf("resolve_cell_rating_descriptions: placeholder values query: %w", err)
	}

	results := make([]*matrixpb.CellRatingResolution, len(states))
	for i, st := range states {
		// Defensive: every state must be terminal by here. A state that
		// reaches the end of the pipeline still "in flight" (resolved=false,
		// e.g. because it passed set validation but resolveRatingEntries
		// somehow skipped it) fails closed rather than reporting the zero-
		// value UNSPECIFIED, which a caller would silently ignore.
		if !st.resolved {
			st.status = enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_UNRESOLVED_IDENTITY
			st.reason = "resolution incomplete"
		}
		results[i] = &matrixpb.CellRatingResolution{
			Cell:                   st.cell,
			Status:                 st.status,
			RatingDescriptionSetId: st.setID,
			Descriptions:           st.descriptionsOut,
			Reason:                 st.reason,
		}
	}
	return &matrixpb.ResolveCellRatingDescriptionsResponse{Results: results, Success: true}, nil
}

// loadCellAncestryAndSlotMode is query 1/5. For each requested cell it
// validates job_task -> job_phase -> job ancestry (must resolve to the
// REQUESTED job_id, workspace-scoped — a mismatch or cross-workspace job_task
// is indistinguishable from "not found", never leaked). Codex review round 1
// item 3: job_task.workspace_id, job_phase.workspace_id and
// subscription.workspace_id (all present and NOT NULL-populated on this
// education2clone20260925a data — confirmed by read-only psql \d + a 0-row
// NULL-workspace count) are ALSO required to equal the caller's workspace,
// not just the job itself — defense in depth against a denormalized-copy
// mismatch anywhere in the ancestry chain. This is a strict equality, not an
// "OR IS NULL" fallback: unlike score_scale/score_scale_band (proto field 11/2
// is `optional string workspace_id`, an explicit global-row convention),
// job_task/job_phase/subscription's own `optional string workspace_id` proto
// fields carry no such global-row convention for tenant-owned operational
// rows, and the live data has zero NULL rows across all three tables. A
// mismatch on any of the three simply makes that LEFT JOIN leg NULL, which
// cascades to the existing "job/task ancestry not found" /
// "job has no product / subscription / price plan" UNRESOLVED_IDENTITY
// fallbacks below — no separate branch needed. It reads the slot's
// rating_mode/rating_scale_id off template_task_criteria (via job_task's own
// template_task binding — independent of the job/subscription/price_plan
// chain, per interfaces.md §2 "resubmit_same_value ... derived from
// rating_mode"). Cells whose slot is not RATING_MODE_NUMERIC_WITH_DESCRIPTION
// are terminal here (UNSPECIFIED, "caller ignores" — never product/AY work).
// DISTINCT ON (req.ord) guarantees exactly one row per requested cell even if
// a data-integrity issue ever produced more than one active
// template_task_criteria row for (job_template_task, criterion).
func (a *PostgresOutcomeMatrixQuery) loadCellAncestryAndSlotMode(
	ctx context.Context,
	queryer outcomeMatrixRowsQueryer,
	states []*cellResolutionState,
	workspaceID string,
) error {
	jobIDs := make([]string, len(states))
	jobTaskIDs := make([]string, len(states))
	criteriaIDs := make([]string, len(states))
	for i, st := range states {
		jobIDs[i] = st.cell.GetJobId()
		jobTaskIDs[i] = st.cell.GetJobTaskId()
		criteriaIDs[i] = st.cell.GetOutcomeCriteriaId()
	}

	const q = `
		WITH req(job_id, job_task_id, outcome_criteria_id, ord) AS (
			SELECT * FROM unnest($1::text[], $2::text[], $3::text[])
				WITH ORDINALITY AS t(job_id, job_task_id, outcome_criteria_id, ord)
		)
		SELECT DISTINCT ON (req.ord)
			req.ord,
			(j.id IS NOT NULL) AS job_found,
			j.output_product_id,
			NULLIF(j.output_product_variant_id, '') AS variant,
			pr.plan_id,
			pr.price_schedule_id,
			ttc.rating_mode,
			ttc.rating_scale_id
		FROM req
		LEFT JOIN ` + entityid.JobTask + ` jt
			ON jt.id = req.job_task_id AND jt.active AND jt.workspace_id = $4
		LEFT JOIN ` + entityid.JobPhase + ` jp
			ON jp.id = jt.job_phase_id AND jp.active AND jp.workspace_id = $4
		LEFT JOIN ` + entityid.Job + ` j
			ON j.id = jp.job_id AND j.id = req.job_id AND j.workspace_id = $4 AND j.active
		LEFT JOIN ` + entityid.Subscription + ` s
			ON s.id = j.origin_id AND s.workspace_id = $4
		LEFT JOIN ` + entityid.PricePlan + ` pr
			ON pr.id = s.price_plan_id
		LEFT JOIN ` + entityid.JobTemplateTask + ` jtt
			ON jtt.id = jt.template_task_id AND jtt.active
		LEFT JOIN ` + entityid.TemplateTaskCriteria + ` ttc
			ON ttc.job_template_task_id = jtt.id AND ttc.outcome_criteria_id = req.outcome_criteria_id AND ttc.active
		ORDER BY req.ord, ttc.id`

	rows, err := queryer.QueryContext(ctx, q, pq.Array(jobIDs), pq.Array(jobTaskIDs), pq.Array(criteriaIDs), workspaceID)
	if err != nil {
		return err
	}
	defer rows.Close()

	seen := make([]bool, len(states)+1)
	for rows.Next() {
		var (
			ord                       int
			jobFound                  bool
			outputProductID           sql.NullString
			variant                   sql.NullString
			planID, priceScheduleID   sql.NullString
			ratingMode, ratingScaleID sql.NullString
		)
		if err := rows.Scan(&ord, &jobFound, &outputProductID, &variant, &planID, &priceScheduleID, &ratingMode, &ratingScaleID); err != nil {
			return err
		}
		if ord < 1 || ord > len(states) {
			continue
		}
		seen[ord] = true
		st := states[ord-1]

		if !ratingMode.Valid {
			// No template_task_criteria binding at all for (this job_task's
			// template task, this criterion). If the job ancestry itself
			// didn't resolve either, we cannot identify anything about this
			// cell; otherwise the slot simply isn't bound in description
			// mode (a legitimate, common case — most criteria are STANDARD).
			if !jobFound {
				st.status = enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_UNRESOLVED_IDENTITY
				st.reason = "job/task ancestry not found"
			} else {
				st.status = enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_UNSPECIFIED
			}
			st.resolved = true
			continue
		}

		mode := parseRatingMode(ratingMode)
		if mode != enumspb.RatingMode_RATING_MODE_NUMERIC_WITH_DESCRIPTION {
			st.status = enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_UNSPECIFIED
			st.resolved = true
			continue
		}

		if !jobFound || nullStringVal(outputProductID) == "" || nullStringVal(planID) == "" || nullStringVal(priceScheduleID) == "" {
			st.status = enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_UNRESOLVED_IDENTITY
			st.reason = "job has no product / subscription / price plan"
			st.resolved = true
			continue
		}

		// Proceeds to product_plan resolution — stays "in flight"
		// (resolved=false, status left at the zero value).
		st.productID = outputProductID.String
		st.variant = nullStringVal(variant)
		st.planID = planID.String
		st.scheduleID = priceScheduleID.String
		st.ratingScaleID = nullStringVal(ratingScaleID)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// A requested cell whose job_task_id doesn't exist at all yields NO row
	// from the LEFT-JOIN chain above only if... it always yields a row (req
	// is the driving table), so absence here would only happen if the SQL
	// itself mis-projected an ordinal; fail closed rather than leaving a
	// zero-value UNSPECIFIED unexplained.
	for i, st := range states {
		if !seen[i+1] && !st.resolved {
			st.status = enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_UNRESOLVED_IDENTITY
			st.reason = "job/task ancestry not found"
			st.resolved = true
		}
	}
	return nil
}

// productPlanKey/linkKey/entryKey are local, comparable map keys used to
// dedupe batch-2..5 lookups by distinct value (rule 6: one query per stage,
// never one query per cell).
type productPlanKey struct{ productID, planID, variant string }
type ratingLinkKey struct{ productPlanID, scheduleID string }
type ratingEntryKey struct{ setID, criteriaID string }

// resolveProductPlans is query 2/5 (schema-proposal.md §4 step "product_plan
// ... exactly one active; 0 -> UNRESOLVED_IDENTITY, >1 -> AMBIGUOUS"). The
// null-safe variant match (COALESCE(NULLIF(product_variant_id,”),”)) mirrors
// the unique index the design relies on for "impossible while the unique
// index holds" — but AMBIGUOUS is still computed from the real row count, not
// assumed, so a corrupted index is caught rather than silently picking one.
func (a *PostgresOutcomeMatrixQuery) resolveProductPlans(ctx context.Context, queryer outcomeMatrixRowsQueryer, states []*cellResolutionState) error {
	order := make([]productPlanKey, 0, len(states))
	seen := make(map[productPlanKey]int, len(states))
	for _, st := range states {
		if st.resolved {
			continue
		}
		k := productPlanKey{st.productID, st.planID, st.variant}
		if _, ok := seen[k]; !ok {
			seen[k] = len(order) + 1
			order = append(order, k)
		}
	}
	if len(order) == 0 {
		return nil
	}

	productIDs := make([]string, len(order))
	planIDs := make([]string, len(order))
	variants := make([]string, len(order))
	for i, k := range order {
		productIDs[i] = k.productID
		planIDs[i] = k.planID
		variants[i] = k.variant
	}

	q := `
		WITH keys(product_id, plan_id, variant_key, ord) AS (
			SELECT * FROM unnest($1::text[], $2::text[], $3::text[])
				WITH ORDINALITY AS t(product_id, plan_id, variant_key, ord)
		)
		SELECT k.ord, pp.id
		FROM keys k
		JOIN ` + entityid.ProductPlan + ` pp
			ON pp.product_id = k.product_id AND pp.plan_id = k.plan_id
		AND COALESCE(NULLIF(pp.product_variant_id, ''), '') = k.variant_key
		AND pp.active
		ORDER BY k.ord`

	rows, err := queryer.QueryContext(ctx, q, pq.Array(productIDs), pq.Array(planIDs), pq.Array(variants))
	if err != nil {
		return err
	}
	defer rows.Close()

	counts := make([]int, len(order)+1)
	ids := make([]string, len(order)+1)
	for rows.Next() {
		var ord int
		var id string
		if err := rows.Scan(&ord, &id); err != nil {
			return err
		}
		if ord < 1 || ord > len(order) {
			continue
		}
		counts[ord]++
		ids[ord] = id
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, st := range states {
		if st.resolved {
			continue
		}
		ord := seen[productPlanKey{st.productID, st.planID, st.variant}]
		switch counts[ord] {
		case 0:
			st.status = enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_UNRESOLVED_IDENTITY
			st.reason = "no product offering configured for this job's product/plan/variant"
			st.resolved = true
		case 1:
			st.productPlanID = ids[ord]
		default:
			st.status = enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_AMBIGUOUS
			st.reason = "more than one product offering matches this job's product/plan/variant"
			st.resolved = true
		}
	}
	return nil
}

// resolveRatingLinks is query 3/5 — active rating_description_set_product_plan
// row for (product_plan_id, price_schedule_id), workspace-scoped (Q21: none ->
// NO_LINK, like a level without a description; >1 -> AMBIGUOUS, the partial
// unique index guards this in normal operation).
func (a *PostgresOutcomeMatrixQuery) resolveRatingLinks(ctx context.Context, queryer outcomeMatrixRowsQueryer, states []*cellResolutionState, workspaceID string) error {
	order := make([]ratingLinkKey, 0, len(states))
	seen := make(map[ratingLinkKey]int, len(states))
	for _, st := range states {
		if st.resolved {
			continue
		}
		k := ratingLinkKey{st.productPlanID, st.scheduleID}
		if _, ok := seen[k]; !ok {
			seen[k] = len(order) + 1
			order = append(order, k)
		}
	}
	if len(order) == 0 {
		return nil
	}

	productPlanIDs := make([]string, len(order))
	scheduleIDs := make([]string, len(order))
	for i, k := range order {
		productPlanIDs[i] = k.productPlanID
		scheduleIDs[i] = k.scheduleID
	}

	q := `
		WITH keys(product_plan_id, price_schedule_id, ord) AS (
			SELECT * FROM unnest($1::text[], $2::text[])
				WITH ORDINALITY AS t(product_plan_id, price_schedule_id, ord)
		)
		SELECT k.ord, rdsp.rating_description_set_id
		FROM keys k
		JOIN ` + ratingDescriptionSetProductPlanTable + ` rdsp
			ON rdsp.product_plan_id = k.product_plan_id AND rdsp.price_schedule_id = k.price_schedule_id
		AND rdsp.active AND rdsp.workspace_id = $3
		ORDER BY k.ord`

	rows, err := queryer.QueryContext(ctx, q, pq.Array(productPlanIDs), pq.Array(scheduleIDs), workspaceID)
	if err != nil {
		return err
	}
	defer rows.Close()

	counts := make([]int, len(order)+1)
	setIDs := make([]string, len(order)+1)
	for rows.Next() {
		var ord int
		var setID string
		if err := rows.Scan(&ord, &setID); err != nil {
			return err
		}
		if ord < 1 || ord > len(order) {
			continue
		}
		counts[ord]++
		setIDs[ord] = setID
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, st := range states {
		if st.resolved {
			continue
		}
		ord := seen[ratingLinkKey{st.productPlanID, st.scheduleID}]
		switch counts[ord] {
		case 0:
			st.status = enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_NO_LINK
			st.reason = "no active rating-description link for this offering and academic year"
			st.resolved = true
		case 1:
			st.setID = setIDs[ord]
		default:
			st.status = enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_AMBIGUOUS
			st.reason = "more than one active rating-description link for this offering and academic year"
			st.resolved = true
		}
	}
	return nil
}

// ratingSetInfo is the row shape query 4/5 reads per distinct set id.
type ratingSetInfo struct {
	workspaceID   string
	versionStatus string
	scoreScaleID  string
}

// resolveRatingSets is query 4/5 (espyna-golang.md resolver rule 4): the
// linked set must be in the job's own workspace, PUBLISHED or DEPRECATED (Q19
// — DRAFT never resolves), and use the SAME scale as the slot's
// rating_scale_id; any miss is INVALID_CONFIG (integrity failure, not a user
// action). A single id lookup (no ambiguity dimension here — the link query
// already proved exactly one set id per cell).
func (a *PostgresOutcomeMatrixQuery) resolveRatingSets(ctx context.Context, queryer outcomeMatrixRowsQueryer, states []*cellResolutionState, workspaceID string) error {
	ids := make([]string, 0, len(states))
	seen := make(map[string]bool, len(states))
	for _, st := range states {
		if st.resolved || st.setID == "" {
			continue
		}
		if !seen[st.setID] {
			seen[st.setID] = true
			ids = append(ids, st.setID)
		}
	}
	if len(ids) == 0 {
		return nil
	}

	q := `SELECT id, workspace_id, version_status, score_scale_id FROM ` + ratingDescriptionSetTable + ` WHERE id = ANY($1::text[])`
	rows, err := queryer.QueryContext(ctx, q, pq.Array(ids))
	if err != nil {
		return err
	}
	defer rows.Close()

	byID := make(map[string]ratingSetInfo, len(ids))
	for rows.Next() {
		var id string
		var wsID, status, scaleID sql.NullString
		if err := rows.Scan(&id, &wsID, &status, &scaleID); err != nil {
			return err
		}
		byID[id] = ratingSetInfo{
			workspaceID:   nullStringVal(wsID),
			versionStatus: nullStringVal(status),
			scoreScaleID:  nullStringVal(scaleID),
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, st := range states {
		if st.resolved || st.setID == "" {
			continue
		}
		info, ok := byID[st.setID]
		switch {
		case !ok:
			st.status = enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_INVALID_CONFIG
			st.reason = "linked rating-description set not found"
			st.resolved = true
		case info.workspaceID != workspaceID:
			st.status = enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_INVALID_CONFIG
			st.reason = "linked rating-description set belongs to a different workspace"
			st.resolved = true
		case info.versionStatus != versionStatusPublishedValue && info.versionStatus != versionStatusDeprecatedValue:
			st.status = enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_INVALID_CONFIG
			st.reason = "linked rating-description set is not published"
			st.resolved = true
		case st.ratingScaleID == "" || st.ratingScaleID != info.scoreScaleID:
			st.status = enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_INVALID_CONFIG
			st.reason = "linked rating-description set uses a different scale than this criterion's slot"
			st.resolved = true
		default:
			// Passes every check — proceeds to entries, still "in flight".
		}
	}
	return nil
}

// resolveRatingEntries is query 5/5 — every band's text for THIS criterion in
// the validated set (schema-proposal.md §2: UNIQUE (set, criterion, band), so
// up to one row per level; the caller matches by numeric value — an empty
// result here is still RESOLVED, "may lack this value -> NO_ENTRY at match
// time" per the enum's own doc comment, never a distinct server status).
//
// Codex review round 1 item 3: this query additionally flags, per row, two
// integrity conditions that must reject the WHOLE cell as INVALID_CONFIG
// rather than silently drop the offending row (which would look identical to
// a legitimate "no entry at this level" — exactly the "silently 'no entry'"
// failure the review called out):
//   - ws_ok: the entry's OWN workspace_id must equal the caller's workspace
//     (rating_description_set_entry.workspace_id is NOT NULL in proto/
//     migration — no global-row convention here, unlike score_scale/
//     score_scale_band).
//   - scale_ok: the entry's band must belong to the SAME score_scale as the
//     validated set (resolveRatingSets already proved the set's scale matches
//     the slot's rating_scale_id; this additionally proves the band FK a
//     specific entry row carries hasn't drifted onto a different scale's
//     band — a data-integrity condition the independent FKs in this schema do
//     not otherwise prevent).
//
// A band that is simply inactive or missing is unaffected by this change —
// that row is excluded from the JOIN exactly as before (a legitimate "no
// entry for this level", not an integrity failure).
func (a *PostgresOutcomeMatrixQuery) resolveRatingEntries(ctx context.Context, queryer outcomeMatrixRowsQueryer, states []*cellResolutionState, workspaceID string) error {
	order := make([]ratingEntryKey, 0, len(states))
	seen := make(map[ratingEntryKey]int, len(states))
	for _, st := range states {
		if st.resolved {
			continue
		}
		k := ratingEntryKey{st.setID, st.cell.GetOutcomeCriteriaId()}
		if _, ok := seen[k]; !ok {
			seen[k] = len(order) + 1
			order = append(order, k)
		}
	}

	descByOrd := make(map[int][]*matrixpb.RatingDescription, len(order))
	invalidOrd := make(map[int]bool, len(order))
	if len(order) > 0 {
		setIDs := make([]string, len(order))
		criteriaIDs := make([]string, len(order))
		for i, k := range order {
			setIDs[i] = k.setID
			criteriaIDs[i] = k.criteriaID
		}

		q := `
			WITH keys(set_id, criterion_id, ord) AS (
				SELECT * FROM unnest($1::text[], $2::text[])
					WITH ORDINALITY AS t(set_id, criterion_id, ord)
			)
			SELECT k.ord,
				-- fix2-backend (codex impl2 #4): entry, band AND scale ownership
				-- (band/scale may be global: workspace_id IS NULL).
				(rdse.workspace_id = $3
				 AND (ssb.workspace_id = $3 OR ssb.workspace_id IS NULL)
				 AND (ss.workspace_id = $3 OR ss.workspace_id IS NULL)) AS ws_ok,
				(ssb.score_scale_id = rds.score_scale_id) AS scale_ok,
				ss.scale_kind, ssb.input_min, ssb.input_max, ssb.input_match, rdse.description
			FROM keys k
			JOIN ` + ratingDescriptionSetEntryTable + ` rdse
				ON rdse.rating_description_set_id = k.set_id AND rdse.outcome_criteria_id = k.criterion_id AND rdse.active
			JOIN ` + ratingDescriptionSetTable + ` rds
				ON rds.id = k.set_id
			JOIN ` + entityid.ScoreScaleBand + ` ssb
				ON ssb.id = rdse.score_scale_band_id AND ssb.active
			JOIN ` + entityid.ScoreScale + ` ss
				ON ss.id = ssb.score_scale_id AND ss.active
			ORDER BY k.ord, rdse.sequence_order, rdse.id`

		rows, err := queryer.QueryContext(ctx, q, pq.Array(setIDs), pq.Array(criteriaIDs), workspaceID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var ord int
			var wsOK, scaleOK bool
			var scaleKind sql.NullString
			var inputMin, inputMax sql.NullFloat64
			var inputMatch sql.NullString
			var description string
			if err := rows.Scan(&ord, &wsOK, &scaleOK, &scaleKind, &inputMin, &inputMax, &inputMatch, &description); err != nil {
				return err
			}
			if !wsOK || !scaleOK {
				invalidOrd[ord] = true
				continue
			}
			descByOrd[ord] = append(descByOrd[ord], &matrixpb.RatingDescription{
				ScaleKind:   parseScaleKind(scaleKind),
				InputMin:    nullFloat64Ptr(inputMin),
				InputMax:    nullFloat64Ptr(inputMax),
				InputMatch:  nullStringPtr(inputMatch),
				Description: description,
			})
		}
		if err := rows.Err(); err != nil {
			return err
		}
	}

	for _, st := range states {
		if st.resolved {
			continue
		}
		ord := seen[ratingEntryKey{st.setID, st.cell.GetOutcomeCriteriaId()}]
		if invalidOrd[ord] {
			st.status = enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_INVALID_CONFIG
			st.reason = "rating description entry violates tenant or scale integrity — rejected, not treated as no entry"
			st.resolved = true
			continue
		}
		st.status = enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_RESOLVED
		st.descriptionsOut = descByOrd[ord]
		st.resolved = true
	}
	return nil
}

// ── placeholder rendering (schema-proposal.md §10) ─────────────────────────

// placeholderClientColumns is the explicit field→column map for every
// allowlisted `client.*` placeholder tag (shared/placeholder registry). The
// batched loader selects ONLY the columns mapped for the tags actually
// present in the batch. Aliases: j = job, c = client, u = "user". Adding a
// tag = one registry entry + one row here (TestPlaceholderRegistry_
// EveryAllowlistedTagHasLoaderMapping enforces the pairing). Values are
// constant SQL fragments, never request input.
var placeholderClientColumns = map[string]string{
	placeholder.TagClientUserFirstName: "u.first_name",
}

// placeholderLoaderRoots lists the tag roots this adapter can load values
// for; a registry tag whose root has no loader here is a wiring bug the
// registry test catches.
var placeholderLoaderRoots = map[string]map[string]string{
	"client": placeholderClientColumns,
}

// renderRatingPlaceholders is the optional query 6/6. It scans every RESOLVED
// cell's descriptions for placeholder tags; only when a `client.*` tag is
// present does it run ONE batched query (requested job ids → job.client_id →
// client.user_id → "user"), scoped job.workspace_id = client.workspace_id =
// caller workspace. Each cell's descriptions are then rendered with that
// cell's own values (single pass — placeholder.Render never re-scans a
// substituted value). Any unknown tag, or an allowlisted tag whose value is
// empty/unresolvable for THIS cell, fails that cell closed: status
// PLACEHOLDER_UNRESOLVED, descriptions cleared, bounded reason (tag names
// only, never values). Rendered descriptions are fresh messages — the entries
// query shares *RatingDescription pointers across cells with the same
// (set, criterion), so they are never mutated in place.
func (a *PostgresOutcomeMatrixQuery) renderRatingPlaceholders(ctx context.Context, queryer outcomeMatrixRowsQueryer, states []*cellResolutionState, workspaceID string) error {
	type pending struct {
		st   *cellResolutionState
		tags []string
	}
	var work []pending
	clientTags := map[string]bool{}
	jobSeen := map[string]bool{}
	var jobIDs []string
	for _, st := range states {
		if st.status != enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_RESOLVED || len(st.descriptionsOut) == 0 {
			continue
		}
		var tags []string
		seen := map[string]bool{}
		for _, d := range st.descriptionsOut {
			for _, t := range placeholder.Tags(d.GetDescription()) {
				if !seen[t] {
					seen[t] = true
					tags = append(tags, t)
				}
			}
		}
		if len(tags) == 0 {
			continue
		}
		work = append(work, pending{st: st, tags: tags})
		needsClient := false
		for _, t := range tags {
			if placeholder.Root(t) == "client" {
				if _, mapped := placeholderClientColumns[t]; mapped && placeholder.IsAllowed(t) {
					clientTags[t] = true
					needsClient = true
				}
			}
		}
		if jid := st.cell.GetJobId(); needsClient && !jobSeen[jid] {
			jobSeen[jid] = true
			jobIDs = append(jobIDs, jid)
		}
	}
	if len(work) == 0 {
		return nil
	}

	valuesByJob := map[string]map[string]string{}
	if len(jobIDs) > 0 {
		tagOrder := make([]string, 0, len(clientTags))
		for t := range clientTags {
			tagOrder = append(tagOrder, t)
		}
		sort.Strings(tagOrder)
		cols := make([]string, len(tagOrder))
		for i, t := range tagOrder {
			cols[i] = placeholderClientColumns[t]
		}
		q := `
			SELECT j.id, ` + strings.Join(cols, ", ") + `
			FROM ` + entityid.Job + ` j
			JOIN ` + entityid.Client + ` c
				ON c.id = j.client_id AND c.workspace_id = $2
			JOIN "` + entityid.User + `" u
				ON u.id = c.user_id
			WHERE j.id = ANY($1::text[]) AND j.workspace_id = $2`
		rows, err := queryer.QueryContext(ctx, q, pq.Array(jobIDs), workspaceID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var jobID string
			vals := make([]sql.NullString, len(tagOrder))
			dest := make([]any, 0, len(tagOrder)+1)
			dest = append(dest, &jobID)
			for i := range vals {
				dest = append(dest, &vals[i])
			}
			if err := rows.Scan(dest...); err != nil {
				return err
			}
			m := make(map[string]string, len(tagOrder))
			for i, t := range tagOrder {
				m[t] = nullStringVal(vals[i])
			}
			valuesByJob[jobID] = m
		}
		if err := rows.Err(); err != nil {
			return err
		}
	}

	for _, w := range work {
		values := valuesByJob[w.st.cell.GetJobId()] // nil map → every tag missing
		rendered := make([]*matrixpb.RatingDescription, 0, len(w.st.descriptionsOut))
		var failed error
		for _, d := range w.st.descriptionsOut {
			text, err := placeholder.Render(d.GetDescription(), values)
			if err != nil {
				failed = err
				break
			}
			rendered = append(rendered, &matrixpb.RatingDescription{
				ScaleKind:   d.GetScaleKind(),
				InputMin:    d.InputMin,
				InputMax:    d.InputMax,
				InputMatch:  d.InputMatch,
				Description: text,
			})
		}
		if failed != nil {
			w.st.status = enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_PLACEHOLDER_UNRESOLVED
			w.st.descriptionsOut = nil
			w.st.reason = placeholderFailureReason(failed)
			continue
		}
		w.st.descriptionsOut = rendered
	}
	return nil
}

// placeholderFailureReason is the bounded diagnostic for a PLACEHOLDER_
// UNRESOLVED cell: the error code and tag names only (never a value).
func placeholderFailureReason(err error) string {
	if pe, ok := err.(*placeholder.Error); ok {
		return "rating description placeholder unresolved: " + pe.Code + " (" + strings.Join(pe.Tags, ", ") + ")"
	}
	return "rating description placeholder unresolved"
}
