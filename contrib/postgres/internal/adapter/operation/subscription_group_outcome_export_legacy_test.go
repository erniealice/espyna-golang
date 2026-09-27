//go:build postgresql

package operation

// Frozen pre-P5 export SQL: compare live output and plans before changing the builders.
const legacyClientOutcomeSummaryCTEs = `
WITH group_context AS MATERIALIZED (
  SELECT sg.id, sg.name, sg.active, NOT sg.active AS historical,
         sg.price_schedule_id, COALESCE(ps.name, '') AS price_schedule_name,
         sg.plan_id, COALESCE(pl.name, '') AS plan_name
    FROM {{subscription_group}} sg
    LEFT JOIN {{price_schedule}} ps
      ON ps.id = sg.price_schedule_id AND ps.workspace_id = sg.workspace_id
    LEFT JOIN {{plan}} pl
      ON pl.id = sg.plan_id AND pl.workspace_id = sg.workspace_id
   WHERE sg.id = $2 AND sg.workspace_id = $1
     AND (sg.price_schedule_id IS NULL OR ps.id IS NOT NULL)
     AND (sg.plan_id IS NULL OR pl.id IS NOT NULL)
     AND (
       $6::boolean
       OR EXISTS (
         SELECT 1
           FROM {{workspace_user}} wu
           JOIN {{subscription_group_workspace_user}} sgwu
             ON sgwu.workspace_user_id = wu.id
            AND sgwu.workspace_id = wu.workspace_id
            AND sgwu.subscription_group_id = sg.id
            AND sgwu.active = true
          WHERE wu.workspace_id = $1 AND wu.id = $3 AND wu.active = true
       )
     )
), member_anchor AS MATERIALIZED (
  -- Exact target-client membership is proven before the job graph is read.
  SELECT g.id AS subscription_group_id, g.historical, m.subscription_id,
         c.id AS client_id,
         COALESCE(NULLIF(c.name, ''), btrim(concat_ws(' ', c.first_name, c.last_name)), c.id) AS client_name,
         COALESCE(c.first_name, '') AS client_first_name,
         COALESCE(c.last_name, '') AS client_last_name
    FROM group_context g
    JOIN {{subscription_group_member}} m
      ON m.subscription_group_id = g.id AND m.workspace_id = $1
     AND m.client_id = $4
    JOIN {{subscription}} s
      ON s.id = m.subscription_id AND s.workspace_id = $1 AND s.client_id = m.client_id
    JOIN {{client}} c
      ON c.id = m.client_id AND c.workspace_id = $1
     AND (g.historical OR c.active = true)
   WHERE g.historical OR (m.active = true AND s.active = true)
), job_rows AS MATERIALIZED (
  -- Match the section matrix's deterministic one-job-per-client/template grain.
  SELECT DISTINCT ON (a.client_id, j.job_template_id)
         j.id, j.workspace_id, j.client_id, j.origin_id, j.origin_type,
         j.job_template_id, j.job_category_id, j.output_product_id, j.active,
         COALESCE(local_jt.job_category_id, j.job_category_id) AS effective_category_id,
         a.historical, a.client_id AS anchored_client_id
    FROM member_anchor a
    JOIN {{job}} j
      ON j.origin_type = 'ORIGIN_TYPE_SUBSCRIPTION'
     AND j.origin_id = a.subscription_id
     AND j.client_id = a.client_id
     AND j.workspace_id = $1
    LEFT JOIN {{job_template}} local_jt
      ON local_jt.id = j.job_template_id AND local_jt.workspace_id = $1
   WHERE (
       (a.historical = false AND j.active = true AND local_jt.id IS NOT NULL AND local_jt.active = true)
       OR (
         a.historical = true
         AND (local_jt.id IS NOT NULL OR NOT EXISTS (
           SELECT 1 FROM {{job_template}} foreign_jt WHERE foreign_jt.id = j.job_template_id
         ))
       )
     )
   ORDER BY a.client_id, j.job_template_id, j.id ASC
), local_templates AS MATERIALIZED (
  SELECT DISTINCT jt.id, jt.workspace_id, jt.name, jt.template_code, jt.job_category_id, jt.active
    FROM job_rows j
    JOIN {{job_template}} jt
      ON jt.id = j.job_template_id AND jt.workspace_id = $1
   WHERE j.historical OR jt.active = true
), local_categories AS MATERIALIZED (
  SELECT DISTINCT jc.id, jc.workspace_id, jc.code, jc.name, jc.sort_order, jc.active
    FROM job_rows j
    LEFT JOIN local_templates jt ON jt.id = j.job_template_id
    JOIN {{job_category}} jc
      ON jc.id = j.effective_category_id
     AND jc.workspace_id = $1
   WHERE j.historical OR jc.active = true
), job_phases AS MATERIALIZED (
  SELECT jp.id, jp.workspace_id, jp.job_id, jp.template_phase_id, jp.phase_order, jp.active,
         jp.approval_status, jp.submitted_by, jp.submitted_at, jp.verified_by, jp.verified_at,
         jp.published_by, jp.published_at, jp.return_reason, jp.returned_by, jp.returned_at,
         j.historical
    FROM job_rows j
    JOIN {{job_phase}} jp
      ON jp.job_id = j.id AND jp.workspace_id = $1
   WHERE j.historical OR jp.active = true
), template_phases AS MATERIALIZED (
  SELECT DISTINCT jtp.id, jtp.workspace_id, jtp.job_template_id, jtp.code, jtp.name, jtp.phase_order, jtp.active,
         jtp.output_product_variant_id, j.historical
    FROM job_phases jp
    JOIN job_rows j ON j.id = jp.job_id
    JOIN {{job_template_phase}} jtp
      ON jtp.id = jp.template_phase_id AND jtp.workspace_id = $1
   WHERE j.historical OR jtp.active = true
), template_tasks AS MATERIALIZED (
  SELECT DISTINCT jtt.id, jtt.workspace_id, jtt.job_template_phase_id,
         jtt.name, jtt.code, jtt.step_order, jtt.active, tp.historical
    FROM template_phases tp
    JOIN {{job_template_task}} jtt
      ON jtt.job_template_phase_id = tp.id AND jtt.workspace_id = $1
   WHERE tp.historical OR jtt.active = true
), job_tasks AS MATERIALIZED (
  SELECT jt.id, jt.workspace_id, jt.job_phase_id, jt.template_task_id, jt.assigned_to, jt.active,
         jp.historical
    FROM job_phases jp
    JOIN {{job_task}} jt
      ON jt.job_phase_id = jp.id AND jt.workspace_id = $1
   WHERE jp.historical OR jt.active = true
), template_task_criteria AS MATERIALIZED (
  SELECT DISTINCT ttc.id, ttc.workspace_id, ttc.job_template_task_id,
         ttc.outcome_criteria_id, ttc.sequence_order, ttc.required_override,
         ttc.weight_override, ttc.active, tt.historical
    FROM template_tasks tt
    JOIN {{template_task_criteria}} ttc
      ON ttc.job_template_task_id = tt.id AND ttc.workspace_id = $1
   WHERE tt.historical OR ttc.active = true
), outcome_criteria AS MATERIALIZED (
  SELECT DISTINCT oc.id, oc.workspace_id, oc.name, oc.code, oc.unit,
         oc.decimal_places, oc.min_score, oc.max_score, oc.score_increment,
         oc.pass_label, oc.fail_label, oc.required, oc.active, ttc.historical
    FROM template_task_criteria ttc
    JOIN {{outcome_criteria}} oc
      ON oc.id = ttc.outcome_criteria_id AND oc.workspace_id = $1
   WHERE ttc.historical OR oc.active = true
), latest_task_outcomes AS MATERIALIZED (
  SELECT DISTINCT ON (jt.id, ttc.id)
         jt.id AS job_task_id, ttc.id AS template_task_criteria_id,
         o.numeric_value, o.categorical_value AS scaled_label,
         o.determination_note,
         floor(extract(epoch FROM o.recorded_date) * 1000)::bigint AS recorded_date
    FROM job_tasks jt
    JOIN template_tasks tt ON tt.id = jt.template_task_id
    JOIN template_task_criteria ttc ON ttc.job_template_task_id = tt.id
    JOIN {{task_outcome}} o
      ON o.job_task_id = jt.id AND o.criteria_version_id = ttc.outcome_criteria_id
     AND o.workspace_id = $1 AND (jt.historical OR o.active = true)
   ORDER BY jt.id, ttc.id, o.recorded_date DESC NULLS LAST, o.id DESC
), latest_phase_summaries AS MATERIALIZED (
  SELECT DISTINCT ON (pos.job_phase_id)
         pos.id, pos.workspace_id, pos.job_id, pos.job_phase_id,
         pos.scaled_label, pos.scaled_score, pos.summary_score,
         pos.total_criteria_count, pos.pass_count, pos.fail_count,
         pos.conditional_count, pos.deferred_count, pos.na_count, pos.narrative,
         pos.active,
         floor(extract(epoch FROM pos.date_created) * 1000)::bigint AS date_created
    FROM job_phases jp
    JOIN {{phase_outcome_summary}} pos
      ON pos.job_phase_id = jp.id AND pos.job_id = jp.job_id
     AND pos.workspace_id = $1 AND pos.active = true
   ORDER BY pos.job_phase_id, pos.date_created DESC NULLS LAST, pos.id DESC
), latest_job_summaries AS MATERIALIZED (
  SELECT DISTINCT ON (jos.job_id)
         jos.id, jos.workspace_id, jos.job_id, jos.client_id, jos.scaled_label, jos.scaled_score,
         jos.summary_score, jos.total_criteria_count, jos.pass_count, jos.fail_count,
         jos.conditional_count, jos.deferred_count, jos.na_count, jos.narrative,
         jos.active, floor(extract(epoch FROM jos.date_created) * 1000)::bigint AS date_created
    FROM job_rows j
    JOIN {{job_outcome_summary}} jos
      ON jos.job_id = j.id AND jos.workspace_id = $1 AND jos.active = true
     AND (jos.client_id IS NULL OR jos.client_id = j.client_id)
   ORDER BY jos.job_id, jos.date_created DESC NULLS LAST, jos.id DESC
), outcome_lines AS MATERIALIZED (
  SELECT jol.id, jol.workspace_id, jol.client_id, jol.job_outcome_summary_id,
         jol.label, jol.weight_or_credits, jol.output_value, jol.output_label, jol.active
    FROM latest_job_summaries jos
    JOIN job_rows j ON j.id = jos.job_id
    JOIN {{job_outcome_line}} jol
      ON jol.job_outcome_summary_id = jos.id AND jol.workspace_id = $1 AND jol.active = true
     AND (jol.client_id IS NULL OR jol.client_id = j.client_id)
), rating_description_rows AS MATERIALIZED (
  SELECT DISTINCT td.id, td.workspace_id, td.template_task_criteria_id,
         td.description, td.sequence_order, td.active
    FROM template_task_criteria ttc
    JOIN {{template_task_criteria_rating_description}} td
      ON td.template_task_criteria_id = ttc.id
     AND td.workspace_id = $1 AND (ttc.historical OR td.active = true)
), render_gate_template_sheets AS MATERIALIZED (
  SELECT jp.template_phase_id,
         COUNT(*)::int AS target_count,
         BOOL_OR(
           ` + gateAnyWorkflowEnteredSQLExpr + `
         ) AS any_workflow_entered,
         BOOL_AND(jp.approval_status = 'PHASE_APPROVAL_STATUS_PUBLISHED') AS all_published,
         BOOL_OR(EXISTS (
           SELECT 1
             FROM {{job_task}} gate_task
             JOIN {{task_outcome}} gate_outcome
               ON gate_outcome.job_task_id = gate_task.id
              AND gate_outcome.workspace_id = $1 AND gate_outcome.active = true
            WHERE gate_task.job_phase_id = jp.id
              AND gate_task.workspace_id = $1 AND gate_task.active = true
         )) AS has_data
    FROM {{job_phase}} jp
    JOIN {{job}} j ON j.id = jp.job_id
   WHERE jp.template_phase_id IN (
           SELECT DISTINCT projected.template_phase_id
             FROM job_phases projected
            WHERE projected.active = true AND projected.template_phase_id IS NOT NULL
         )
     AND j.workspace_id = $1 AND jp.workspace_id = $1 AND jp.active = true
     AND jp.template_phase_id IS NOT NULL{{render_gate_group_narrow}}
   GROUP BY jp.template_phase_id
), render_gate_singletons AS MATERIALIZED (
  SELECT jp.id AS job_phase_id,
         (` + gateAnyWorkflowEnteredSQLExpr + `) AS any_workflow_entered,
         (jp.approval_status = 'PHASE_APPROVAL_STATUS_PUBLISHED') AS all_published,
         EXISTS (
           SELECT 1
             FROM {{job_task}} gate_task
             JOIN {{task_outcome}} gate_outcome
               ON gate_outcome.job_task_id = gate_task.id
              AND gate_outcome.workspace_id = $1 AND gate_outcome.active = true
            WHERE gate_task.job_phase_id = jp.id
              AND gate_task.workspace_id = $1 AND gate_task.active = true
         ) AS has_data
    FROM job_phases projected
    JOIN {{job_phase}} jp ON jp.id = projected.id AND jp.workspace_id = $1
   WHERE projected.active = true AND projected.template_phase_id IS NULL AND jp.active = true
), attribute_rows AS MATERIALIZED (
  SELECT DISTINCT attr.code, ca.value
    FROM member_anchor a
    JOIN {{client_attribute}} ca ON ca.client_id = a.client_id AND ca.active = true
    JOIN {{attribute}} attr ON attr.id = ca.attribute_id AND attr.active = true
   WHERE attr.code IN (SELECT jsonb_array_elements_text($5::jsonb))
), plan_attribute_rows AS MATERIALIZED (
  -- Configured attributes of the group's plan (the section's grade/level).
  -- plan_attribute has no workspace_id; it is reached only through the
  -- workspace-scoped group_context plan.
  SELECT DISTINCT attr.code, pa.value
    FROM group_context g
    JOIN {{plan_attribute}} pa ON pa.plan_id = g.plan_id AND pa.active = true
    JOIN {{attribute}} attr ON attr.id = pa.attribute_id AND attr.active = true
   WHERE attr.code IN (SELECT jsonb_array_elements_text({{plan_attribute_codes_param}}::jsonb))
), staff_candidates AS MATERIALIZED (
  -- Direct task assignees are the override. A class-edge fallback is used only
  -- for phases with no valid active direct assignee. SGPP is scoped by workspace
  -- and exact group; product_plan has no workspace_id in the current schema and
  -- is therefore reached only through that edge plus this job's output product.
  -- The fallback considers only PRIMARY edges (DP-10/DP-11; mirrors the courses
  -- list's classEdgeEligibilityLivePredicate in job_template_summary_query.go)
  -- and returns EVERY qualifying primary edge, not just the newest one: a
  -- demoted-to-secondary staff member must never outrank a still-active
  -- primary staff just because their edge sorts newer.
  SELECT DISTINCT j.id AS job_id, jp.id AS job_phase_id, s.id AS staff_id,
         COALESCE(NULLIF(btrim(concat_ws(' ', u.first_name, u.last_name)), ''), s.id) AS display_name,
         0 AS source_order, ''::text AS edge_sort
    FROM job_rows j
    JOIN job_phases jp ON jp.job_id = j.id
    JOIN job_tasks jt ON jt.job_phase_id = jp.id AND jt.assigned_to IS NOT NULL AND btrim(jt.assigned_to) <> ''
    JOIN {{staff}} s ON s.id = jt.assigned_to AND s.workspace_id = $1 AND s.active = true
    LEFT JOIN "{{user}}" u ON u.id = s.user_id AND u.active = true
  UNION ALL
  SELECT j.id, jp.id, s.id,
         COALESCE(NULLIF(btrim(concat_ws(' ', u.first_name, u.last_name)), ''), s.id),
         1, picked.edge_sort
    FROM job_rows j
    JOIN job_phases jp ON jp.job_id = j.id
    JOIN LATERAL (
      SELECT sgpps.staff_id, sgpps.id AS edge_sort
        FROM {{subscription_group_product_plan_staff}} sgpps
        JOIN {{product_plan}} pp ON pp.id = sgpps.product_plan_id
        LEFT JOIN {{product_plan_staff}} pps ON pps.id = sgpps.product_plan_staff_id
       WHERE sgpps.subscription_group_id = $2
         AND sgpps.workspace_id = $1 AND sgpps.active = true
         AND sgpps.role = 'primary'
         AND pp.product_id = j.output_product_id
         AND (sgpps.job_template_phase_id IS NULL OR sgpps.job_template_phase_id = jp.template_phase_id)
         AND (sgpps.product_plan_staff_id IS NULL OR pps.active)
       ORDER BY sgpps.date_created DESC NULLS LAST, sgpps.id DESC
    ) picked ON true
    JOIN {{staff}} s ON s.id = picked.staff_id AND s.workspace_id = $1 AND s.active = true
    LEFT JOIN "{{user}}" u ON u.id = s.user_id AND u.active = true
   WHERE NOT EXISTS (
     SELECT 1 FROM job_tasks direct_task
       JOIN {{staff}} direct_staff
         ON direct_staff.id = direct_task.assigned_to
        AND direct_staff.workspace_id = $1 AND direct_staff.active = true
      WHERE direct_task.job_phase_id = jp.id
   )
), staff_assignments AS MATERIALIZED (
  SELECT DISTINCT ON (job_id, job_phase_id, staff_id)
         job_id, job_phase_id, staff_id, display_name
    FROM staff_candidates
   ORDER BY job_id, job_phase_id, staff_id, source_order, edge_sort
), staff_rows AS MATERIALIZED (
  SELECT DISTINCT staff_id, display_name FROM staff_assignments
)
`

func legacyOutcomeExportCTEs(staffClause string) string {
	return `
WITH group_context AS (
  SELECT sg.id, sg.name, sg.active, NOT sg.active AS historical,
         sg.price_schedule_id, COALESCE(ps.name, '') AS price_schedule_name,
         sg.plan_id, COALESCE(pl.name, '') AS plan_name
    FROM {{subscription_group}} sg
    LEFT JOIN {{price_schedule}} ps
      ON ps.id = sg.price_schedule_id AND ps.workspace_id = sg.workspace_id
    LEFT JOIN {{plan}} pl
      ON pl.id = sg.plan_id AND pl.workspace_id = sg.workspace_id
   WHERE sg.id = $2
     AND sg.workspace_id = $1
     AND (sg.price_schedule_id IS NULL OR ps.id IS NOT NULL)
     AND (sg.plan_id IS NULL OR pl.id IS NOT NULL)
     AND (
       $4::boolean
       OR EXISTS (
         SELECT 1
           FROM {{workspace_user}} wu
           JOIN {{subscription_group_workspace_user}} sgwu
             ON sgwu.workspace_user_id = wu.id
            AND sgwu.workspace_id = wu.workspace_id
            AND sgwu.subscription_group_id = sg.id
            AND sgwu.active = true
          WHERE wu.workspace_id = $1
            AND wu.id = $3
            AND wu.active = true
       )
     )
), member_rows AS (
  SELECT g.historical, m.subscription_id, m.client_id,
         COALESCE(NULLIF(c.name, ''), btrim(concat_ws(' ', c.first_name, c.last_name)), c.id) AS client_name,
         COALESCE(c.first_name, '') AS client_first_name,
         COALESCE(c.last_name, '') AS client_last_name
    FROM group_context g
    JOIN {{subscription_group_member}} m
      ON m.subscription_group_id = g.id AND m.workspace_id = $1
    JOIN {{subscription}} s
      ON s.id = m.subscription_id AND s.workspace_id = $1 AND s.client_id = m.client_id
    JOIN {{client}} c
      ON c.id = m.client_id AND c.workspace_id = $1 AND c.active = true
   WHERE g.historical OR (m.active = true AND s.active = true)
), ranked_jobs AS (
  SELECT m.client_id, m.client_name, m.client_first_name, m.client_last_name,
         j.id AS job_id, j.job_template_id,
         CASE WHEN jt.id IS NOT NULL THEN jt.job_category_id ELSE j.job_category_id END AS effective_category_id,
         COALESCE(NULLIF(btrim(jt.name), ''), j.job_template_id) AS job_template_name,
         row_number() OVER (PARTITION BY m.client_id, j.job_template_id ORDER BY j.id ASC) AS job_rank
    FROM member_rows m
    JOIN {{job}} j
      ON j.origin_type = 'ORIGIN_TYPE_SUBSCRIPTION'
     AND j.origin_id = m.subscription_id
     AND j.client_id = m.client_id
     AND j.workspace_id = $1
    LEFT JOIN {{job_template}} jt
      ON jt.id = j.job_template_id AND jt.workspace_id = $1
   WHERE (
       (m.historical = false AND j.active = true AND jt.id IS NOT NULL AND jt.active = true)
       OR (
         m.historical = true
         AND (jt.id IS NOT NULL OR NOT EXISTS (
           SELECT 1 FROM {{job_template}} foreign_jt WHERE foreign_jt.id = j.job_template_id
         ))
       )
     )` + staffClause + `
), chosen_jobs AS (
  SELECT ranked_jobs.*,
         COALESCE(local_category.id, 'uncategorized') AS category_bucket_id,
         COALESCE(local_category.code, '') AS category_code,
         COALESCE(local_category.name, '') AS category_name,
         COALESCE(local_category.sort_order, 2147483647) AS category_sort_order
    FROM ranked_jobs
    LEFT JOIN {{job_category}} local_category
      ON local_category.id = ranked_jobs.effective_category_id
     AND local_category.workspace_id = $1
     AND local_category.active = true
   WHERE ranked_jobs.job_rank = 1
), outcome_capable_jobs AS (
  SELECT cj.*
    FROM chosen_jobs cj
   WHERE EXISTS (
          SELECT 1
            FROM {{job_template_phase}} jtp
           WHERE jtp.job_template_id = cj.job_template_id
             AND jtp.workspace_id = $1
             AND jtp.active = true
       )
      OR EXISTS (
          SELECT 1
            FROM {{job_outcome_summary}} jos
           WHERE jos.job_id = cj.job_id
             AND jos.workspace_id = $1
             AND jos.active = true
       )
), category_options AS (
  SELECT cj.category_bucket_id AS job_category_id,
         min(cj.category_code) AS code,
         min(cj.category_name) AS name,
         min(cj.category_sort_order) AS sort_order,
         EXISTS (
           SELECT 1
             FROM outcome_capable_jobs final_job
             JOIN {{job_outcome_summary}} final_summary
               ON final_summary.job_id = final_job.job_id
              AND final_summary.workspace_id = $1
              AND final_summary.active = true
            WHERE final_job.category_bucket_id = cj.category_bucket_id
         ) AS final_outcome_available
    FROM outcome_capable_jobs cj
   GROUP BY cj.category_bucket_id
), phase_facts AS (
  SELECT cj.category_bucket_id AS job_category_id, cj.job_template_id, jtp.code,
         COALESCE(NULLIF(btrim(jtp.name), ''), jtp.code) AS name,
         COALESCE(jtp.phase_order, 0) AS sequence_order
    FROM outcome_capable_jobs cj
    JOIN category_options co ON co.job_category_id = cj.category_bucket_id
    JOIN {{job_template_phase}} jtp
      ON jtp.job_template_id = cj.job_template_id
     AND jtp.workspace_id = $1
     AND jtp.active = true
   WHERE jtp.code IS NOT NULL AND btrim(jtp.code) <> ''
), phase_options AS (
  SELECT job_category_id, code, min(name) AS name, min(sequence_order) AS sequence_order,
         count(DISTINCT (name, sequence_order)) > 1 AS ambiguous
    FROM phase_facts
   GROUP BY job_category_id, code
), selector_valid AS (
  SELECT 1
    FROM category_options co
   WHERE co.job_category_id = NULLIF($5, '')
     AND (
       ($6 = 'final' AND co.final_outcome_available)
       OR ($6 = 'phase' AND EXISTS (
         SELECT 1 FROM phase_options po
          WHERE po.job_category_id = co.job_category_id
            AND po.code = $7 AND po.ambiguous = false
       ))
     )
), selected_jobs AS (
  SELECT oj.*
    FROM outcome_capable_jobs oj
   WHERE oj.category_bucket_id = NULLIF($5, '')
     AND EXISTS (SELECT 1 FROM selector_valid)
     AND (
       $6 = 'options'
       OR $6 = 'final'
       OR (
         $6 = 'phase'
         AND EXISTS (
           SELECT 1
             FROM phase_facts pf
            WHERE pf.job_category_id = oj.category_bucket_id
              AND pf.job_template_id = oj.job_template_id
              AND pf.code = $7
         )
       )
     )
), matrix_columns AS (
  SELECT job_template_id, min(job_template_name) AS display_name
    FROM selected_jobs
   GROUP BY job_template_id
), matrix_members AS (
  SELECT DISTINCT m.client_id, m.client_name, m.client_first_name, m.client_last_name
    FROM member_rows m
   WHERE $8::boolean = false
      OR EXISTS (SELECT 1 FROM selected_jobs sj WHERE sj.client_id = m.client_id)
), matrix_cells AS (
  SELECT m.client_id, m.client_name, m.client_first_name, m.client_last_name,
         col.job_template_id, sj.job_id IS NOT NULL AS job_present,
         CASE WHEN $6 = 'phase' THEN phase_summary.scaled_label ELSE final_summary.scaled_label END AS scaled_label,
         CASE WHEN $6 = 'phase' THEN phase_summary.scaled_score ELSE final_summary.scaled_score END AS scaled_score,
         CASE WHEN $6 = 'phase' THEN phase_summary.summary_score ELSE final_summary.summary_score END AS summary_score,
         COALESCE(evidence.has_task_outcome, false) AS has_task_outcome,
         COALESCE(evidence.has_positive_task_outcome, false) AS has_positive_task_outcome
    FROM matrix_members m
    CROSS JOIN matrix_columns col
    LEFT JOIN selected_jobs sj
      ON sj.client_id = m.client_id AND sj.job_template_id = col.job_template_id
    LEFT JOIN LATERAL (
      SELECT jp.id
        FROM {{job_phase}} jp
        JOIN {{job_template_phase}} jtp
          ON jtp.id = jp.template_phase_id
         AND jtp.workspace_id = $1
         AND jtp.job_template_id = sj.job_template_id
         AND jtp.active = true
         AND jtp.code = $7
       WHERE $6 = 'phase'
         AND jp.job_id = sj.job_id
         AND jp.workspace_id = $1
         AND jp.active = true
       ORDER BY jp.id ASC
       LIMIT 1
    ) selected_phase ON true
    LEFT JOIN LATERAL (
      SELECT pos.scaled_label, pos.scaled_score, pos.summary_score
        FROM {{phase_outcome_summary}} pos
       WHERE $6 = 'phase'
         AND pos.job_phase_id = selected_phase.id
         AND pos.job_id = sj.job_id
         AND pos.workspace_id = $1
         AND pos.active = true
       ORDER BY pos.date_created DESC NULLS LAST, pos.id DESC
       LIMIT 1
    ) phase_summary ON true
    LEFT JOIN LATERAL (
      -- An imported (is_authoritative) final never carried a real composite: its
      -- stored 0 is a placeholder, so it reads as "no composite" (cells fall
      -- back to the scaled grade). Computed finals keep a real 0.
      SELECT jos.scaled_label, jos.scaled_score,
             CASE WHEN jos.is_authoritative AND jos.summary_score = 0 THEN NULL ELSE jos.summary_score END AS summary_score
        FROM {{job_outcome_summary}} jos
       WHERE $6 = 'final'
         AND jos.job_id = sj.job_id
         AND jos.workspace_id = $1
         AND jos.active = true
       ORDER BY jos.date_created DESC NULLS LAST, jos.id DESC
       LIMIT 1
    ) final_summary ON true
    LEFT JOIN LATERAL (
      SELECT count(*) FILTER (WHERE outcome.numeric_value IS NOT NULL) > 0 AS has_task_outcome,
             count(*) FILTER (WHERE outcome.numeric_value > 0) > 0 AS has_positive_task_outcome
        FROM {{job_phase}} evidence_phase
        JOIN {{job_task}} task
          ON task.job_phase_id = evidence_phase.id
         AND task.workspace_id = $1
         AND task.active = true
        JOIN {{task_outcome}} outcome
          ON outcome.job_task_id = task.id
         AND outcome.workspace_id = $1
         AND outcome.active = true
       WHERE evidence_phase.job_id = sj.job_id
         AND evidence_phase.workspace_id = $1
         AND evidence_phase.active = true
         AND ($6 = 'final' OR ($6 = 'phase' AND evidence_phase.id = selected_phase.id))
    ) evidence ON true
)
`
}
