//go:build postgresql

package operation

// Frozen reference for plan 20260927-db-query-performance P3 (audit DB-01).
//
// These are the Courses-list SQL builders exactly as they stood before the P3
// rewrite, renamed with a legacy prefix. TestJobTemplateSummaryDB_Parity runs
// this SQL and the current builder against a live clone and requires identical
// rows in identical order. Delete this file one release after P3 ships.

import (
	"fmt"
	"strings"

	entityid "github.com/erniealice/espyna-golang/registry/entityid"
)

func legacyBuildListJobTemplateSummariesRequestSQL(
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

	stmt = legacyJobTemplateSummaryCTEs(jjWhere) + ",\ndelivery AS MATERIALIZED (\n" +
		legacyJobTemplateSummarySelectFrom() + outerWhere + "\n),\n" +
		legacyJobTemplateSummaryDeliveryFold() + ",\n" +
		legacyJobTemplateSummaryUniverse(includeFallback) + ",\n" +
		legacyJobTemplateSummaryMetadata(selectedWhere) + ",\n" +
		legacyJobTemplateSummaryPageBase(pageOrder, limitClause) + ",\n" +
		legacyJobTemplateSummaryApprovalCTEs() + "\n" +
		legacyJobTemplateSummaryApprovalSelect(finalOrder)
	return stmt, args, nil
}

func legacyJobTemplateSummaryCTEs(jjWhere string) string {
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
-- Include every active primary class edge. The dd eligibility predicate
-- suppresses each revoked edge individually.
class_primary_edges AS MATERIALIZED (
    SELECT e.id,
        e.subscription_group_id,
        e.product_plan_id,
        e.staff_id,
        e.product_plan_staff_id
    FROM ` + entityid.SubscriptionGroupProductPlanStaff + ` e
    WHERE e.active AND e.workspace_id = $1 AND e.role = 'primary'
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
    -- rationale in the legacyJobTemplateSummaryCTEs doc comment). Emits branch (a)'s
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
    -- Filter linked but revoked eligibility per edge. The downstream
    -- delivery_staff DISTINCT and ordered ARRAY_AGG dedupe and sort names.
    WHERE TRUE
      ` + classEdgeEligibilityLivePredicate + `
)`
}

func legacyJobTemplateSummarySelectFrom() string {
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

func legacyJobTemplateSummaryDeliveryFold() string {
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

func legacyJobTemplateSummaryUniverse(includeTemplateFallback bool) string {
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

func legacyJobTemplateSummaryMetadata(selectedWhere string) string {
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

func legacyJobTemplateSummaryPageBase(orderBy, limitClause string) string {
	return `page_base AS MATERIALIZED (
    SELECT selected_rows.*
    FROM selected_rows
    ORDER BY ` + orderBy + limitClause + `
)`
}

func legacyJobTemplateSummaryApprovalCTEs() string {
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

func legacyJobTemplateSummaryApprovalSelect(finalOrders ...string) string {
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
