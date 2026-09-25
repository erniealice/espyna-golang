//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"sort"
	"strings"
	"time"

	adaptercore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	"github.com/erniealice/espyna-golang/internal/application/ports"
	internalregistry "github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	bindingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/subscription_group_document_template"
	exportpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/subscription_group_outcome_export"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const subscriptionGroupOutcomeDocumentPurpose = "subscription_group_outcome_summary"

func init() {
	internalregistry.RegisterSubscriptionGroupOutcomeExportFactory(func(db any) any {
		sqlDB, ok := db.(*sql.DB)
		if !ok {
			return nil
		}
		return NewPostgresSubscriptionGroupOutcomeExportQuery(sqlDB)
	})
	internalregistry.RegisterSubscriptionGroupOutcomeLandingFactory(func(input internalregistry.SubscriptionGroupOutcomeLandingFactoryInput) any {
		sqlDB, ok := input.Connection.(*sql.DB)
		if !ok {
			return nil
		}
		return NewPostgresSubscriptionGroupOutcomeExportQuery(sqlDB)
	})
}

// PostgresSubscriptionGroupOutcomeExportQuery is the single-statement,
// group-scoped read behind the section export drawer and its report-only
// document resolver. The generated RPC method deliberately uses the narrow
// scope; only the application use case can pass an authorized workspace-wide
// scope through the in-process port.
type PostgresSubscriptionGroupOutcomeExportQuery struct {
	exportpb.UnimplementedSubscriptionGroupOutcomeExportServiceServer
	db *sql.DB
}

func NewPostgresSubscriptionGroupOutcomeExportQuery(db *sql.DB) *PostgresSubscriptionGroupOutcomeExportQuery {
	return &PostgresSubscriptionGroupOutcomeExportQuery{db: db}
}

func (q *PostgresSubscriptionGroupOutcomeExportQuery) GetSubscriptionGroupOutcomeExport(
	ctx context.Context,
	req *exportpb.GetSubscriptionGroupOutcomeExportRequest,
) (*exportpb.GetSubscriptionGroupOutcomeExportResponse, error) {
	return q.GetSubscriptionGroupOutcomeExportScoped(ctx, req, ports.SubscriptionGroupOutcomeExportScope{})
}

// GetSubscriptionGroupClientReportCardScoped is an in-process projection read
// for one client enrollment. It intentionally is not an RPC method: its input
// attribute allowlist is supplied by trusted application composition, while
// the workspace and acting workspace-user always come from request identity.
func (q *PostgresSubscriptionGroupOutcomeExportQuery) GetSubscriptionGroupClientReportCardScoped(
	ctx context.Context,
	req *exportpb.GetSubscriptionGroupClientReportCardRequest,
	scope ports.SubscriptionGroupOutcomeExportScope,
) (*exportpb.GetSubscriptionGroupClientReportCardResponse, error) {
	if q == nil || q.db == nil {
		return nil, fmt.Errorf("subscription group client report card requires PostgreSQL")
	}
	response := &exportpb.GetSubscriptionGroupClientReportCardResponse{Success: true}
	if req == nil || strings.TrimSpace(req.GetSubscriptionGroupId()) == "" || strings.TrimSpace(req.GetClientId()) == "" {
		return response, nil
	}
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil || strings.TrimSpace(id.WorkspaceID) == "" {
		return response, nil
	}

	codes := req.GetClientAttributeCodes()
	if codes == nil {
		codes = []string{}
	}
	codesJSON, err := json.Marshal(codes)
	if err != nil {
		return nil, fmt.Errorf("subscription group client report card attribute codes: %w", err)
	}
	planCodes := req.GetPlanAttributeCodes()
	if planCodes == nil {
		planCodes = []string{}
	}
	planCodesJSON, err := json.Marshal(planCodes)
	if err != nil {
		return nil, fmt.Errorf("subscription group client report card plan attribute codes: %w", err)
	}
	built := buildSubscriptionGroupClientReportCardSQL(id, req, scope, string(codesJSON), string(planCodesJSON))
	rows, err := adaptercore.ExecutorFromContext(ctx, q.db).QueryContext(ctx, built.statement, built.args...)
	if err != nil {
		return nil, fmt.Errorf("subscription group client report card query: %w", err)
	}
	defer rows.Close()

	projection := &exportpb.ClientReportCardProjection{}
	for rows.Next() {
		var kind string
		var payload []byte
		if err := rows.Scan(&kind, &payload); err != nil {
			return nil, fmt.Errorf("subscription group client report card scan: %w", err)
		}
		if err := appendClientReportCardPayload(projection, kind, payload); err != nil {
			return nil, fmt.Errorf("subscription group client report card %s payload: %w", kind, err)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("subscription group client report card rows: %w", err)
	}
	if projection.GetContext() != nil && projection.GetClient() != nil && len(projection.GetClientSubscriptionIds()) > 0 {
		response.ReportCard = projection
	}
	return response, nil
}

// appendClientReportCardPayload decodes one explicit SQL JSON object into the
// matching typed projection field. protojson accepts the snake_case SQL keys
// and validates each row against the generated DTO, rejecting accidental
// expansion of this report-only boundary.
func appendClientReportCardPayload(projection *exportpb.ClientReportCardProjection, kind string, payload []byte) error {
	field, repeated := "", true
	switch kind {
	case "context":
		field, repeated = "context", false
	case "client":
		field, repeated = "client", false
	case "client_subscription":
		field = "client_subscription_ids"
	case "attribute":
		field = "attributes"
	case "plan_attribute":
		field = "plan_attributes"
	case "job":
		field = "jobs"
	case "job_template":
		field = "job_templates"
	case "job_category":
		field = "job_categories"
	case "job_phase":
		field = "job_phases"
	case "job_template_phase":
		field = "job_template_phases"
	case "job_template_task":
		field = "job_template_tasks"
	case "job_task":
		field = "job_tasks"
	case "task_outcome":
		field = "task_outcomes"
	case "outcome_criteria":
		field = "outcome_criteria"
	case "template_task_criteria":
		field = "template_task_criteria"
	case "rating_description":
		field = "rating_descriptions"
	case "phase_outcome_summary":
		field = "phase_outcome_summaries"
	case "job_outcome_summary":
		field = "job_outcome_summaries"
	case "job_outcome_line":
		field = "job_outcome_lines"
	case "staff":
		field = "staff"
	case "teacher_assignment":
		field = "teacher_assignments"
	case "render_gate_job_id":
		field = "render_gate_job_ids"
	case "render_gate_group_id":
		field, repeated = "render_gate_applied_subscription_group_id", false
	case "render_gate_sheet":
		field = "render_gate_sheets"
	default:
		return fmt.Errorf("unknown projection row kind %q", kind)
	}

	wrapped := make([]byte, 0, len(payload)+len(field)+12)
	wrapped = append(wrapped, '{', '"')
	wrapped = append(wrapped, field...)
	wrapped = append(wrapped, '"', ':')
	if repeated {
		wrapped = append(wrapped, '[')
	}
	wrapped = append(wrapped, payload...)
	if repeated {
		wrapped = append(wrapped, ']')
	}
	wrapped = append(wrapped, '}')
	row := &exportpb.ClientReportCardProjection{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(wrapped, row); err != nil {
		return err
	}
	proto.Merge(projection, row)
	return nil
}

type exportContextJSON struct {
	SubscriptionGroupID   string  `json:"subscription_group_id"`
	SubscriptionGroupName string  `json:"subscription_group_name"`
	PriceScheduleID       *string `json:"price_schedule_id"`
	PriceScheduleName     string  `json:"price_schedule_name"`
	PlanID                *string `json:"plan_id"`
	PlanName              string  `json:"plan_name"`
	Historical            bool    `json:"historical"`
}

type exportCategoryJSON struct {
	JobCategoryID         string `json:"job_category_id"`
	Code                  string `json:"code"`
	Name                  string `json:"name"`
	SortOrder             int32  `json:"sort_order"`
	FinalOutcomeAvailable bool   `json:"final_outcome_available"`
}

type exportPhaseJSON struct {
	JobCategoryID string `json:"job_category_id"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	SequenceOrder int32  `json:"sequence_order"`
	Ambiguous     bool   `json:"ambiguous"`
}

type exportColumnJSON struct {
	JobTemplateID string `json:"job_template_id"`
	DisplayName   string `json:"display_name"`
}

type exportCellJSON struct {
	ClientID        string   `json:"client_id"`
	ClientName      string   `json:"client_name"`
	ClientFirstName string   `json:"client_first_name"`
	ClientLastName  string   `json:"client_last_name"`
	JobTemplateID   string   `json:"job_template_id"`
	JobPresent      bool     `json:"job_present"`
	ScaledLabel     *string  `json:"scaled_label"`
	ScaledScore     *float64 `json:"scaled_score"`
	SummaryScore    *float64 `json:"summary_score"`
	HasMarks        bool     `json:"has_marks"`
	HasPositiveMark bool     `json:"has_positive_mark"`
}

type exportScopeSQL struct {
	statement   string
	args        []any
	staffScoped bool
}

func buildSubscriptionGroupClientReportCardSQL(
	id *identity.RequestIdentity,
	req *exportpb.GetSubscriptionGroupClientReportCardRequest,
	scope ports.SubscriptionGroupOutcomeExportScope,
	codesJSON string,
	planCodesJSON string,
) exportScopeSQL {
	args := []any{id.WorkspaceID, req.GetSubscriptionGroupId(), id.WorkspaceUserID, req.GetClientId(), codesJSON, scope.WorkspaceWide}
	gateNarrow, gateArgs := groupNarrowPredicate(req.GetSubscriptionGroupId(), 7, 1)
	args = append(args, gateArgs...)
	args = append(args, planCodesJSON)
	planCodesParam := fmt.Sprintf("$%d", len(args))
	statement := strings.ReplaceAll(clientReportCardCTEs, "{{render_gate_group_narrow}}", gateNarrow) + clientReportCardRowsSQL
	statement = strings.ReplaceAll(statement, "{{plan_attribute_codes_param}}", planCodesParam)
	return exportScopeSQL{
		statement: renderOutcomeExportTables(statement),
		args:      args,
	}
}

// The source tables below deliberately use explicit JSON field lists. In
// particular, client attributes are keyed by the server-configured request
// codes, and task outcomes/staff/client data cross this boundary only through
// their narrow report-card DTOs. Tables without workspace_id (client_attribute,
// attribute, product_plan, and user) are constrained through a workspace-owned
// client, SGPP/job chain, or staff row respectively.
const clientReportCardCTEs = `
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
         j.historical
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
), teacher_candidates AS MATERIALIZED (
  -- Direct task assignees are the override. A class-edge fallback is used only
  -- for phases with no valid active direct assignee. SGPP is scoped by workspace
  -- and exact group; product_plan has no workspace_id in the current schema and
  -- is therefore reached only through that edge plus this job's output product.
  -- The fallback considers only PRIMARY edges (DP-10/DP-11; mirrors the courses
  -- list's classEdgeEligibilityLivePredicate in job_template_summary_query.go)
  -- and returns EVERY qualifying primary edge, not just the newest one: a
  -- demoted-to-secondary staff member must never outrank a still-active
  -- primary teacher just because their edge sorts newer.
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
), teacher_assignments AS MATERIALIZED (
  SELECT DISTINCT ON (job_id, job_phase_id, staff_id)
         job_id, job_phase_id, staff_id, display_name
    FROM teacher_candidates
   ORDER BY job_id, job_phase_id, staff_id, source_order, edge_sort
), staff_rows AS MATERIALIZED (
  SELECT DISTINCT staff_id, display_name FROM teacher_assignments
)
`

const clientReportCardRowsSQL = `
SELECT kind, payload
  FROM (
    SELECT 0 AS kind_order, 'context'::text AS kind, ''::text AS sort_key_1, ''::text AS sort_key_2,
           jsonb_build_object('subscription_group_id', g.id, 'subscription_group_name', g.name,
             'price_schedule_id', g.price_schedule_id, 'price_schedule_name', g.price_schedule_name,
             'plan_id', g.plan_id, 'plan_name', g.plan_name, 'historical', g.historical) AS payload
      FROM group_context g
     WHERE EXISTS (SELECT 1 FROM member_anchor a WHERE a.subscription_group_id = g.id)
    UNION ALL
    SELECT DISTINCT 1, 'client', a.client_id, '', jsonb_build_object('client_id', a.client_id, 'name', a.client_name,
             'first_name', a.client_first_name, 'last_name', a.client_last_name)
      FROM member_anchor a
    UNION ALL
    SELECT DISTINCT 2, 'client_subscription', a.subscription_id, '', to_jsonb(a.subscription_id)
      FROM member_anchor a
    UNION ALL
    SELECT 3, 'attribute', a.code, a.value, jsonb_build_object('code', a.code, 'value', a.value)
      FROM attribute_rows a
    UNION ALL
    SELECT 3, 'plan_attribute', a.code, a.value, jsonb_build_object('code', a.code, 'value', a.value)
      FROM plan_attribute_rows a
    UNION ALL
    SELECT 4, 'job', j.id, '', jsonb_build_object('id', j.id, 'workspace_id', j.workspace_id,
             'client_id', j.client_id, 'origin_id', j.origin_id, 'origin_type', j.origin_type,
             'job_template_id', j.job_template_id, 'job_category_id', j.effective_category_id,
             'output_product_id', j.output_product_id, 'active', j.active)
      FROM job_rows j
    UNION ALL
    SELECT 5, 'job_template', jt.id, '', jsonb_build_object('id', jt.id, 'workspace_id', jt.workspace_id,
             'name', jt.name, 'template_code', jt.template_code,
             'job_category_id', jt.job_category_id, 'active', jt.active)
      FROM local_templates jt
    UNION ALL
    SELECT 6, 'job_category', jc.id, '', jsonb_build_object('id', jc.id, 'workspace_id', jc.workspace_id,
             'code', jc.code, 'name', jc.name, 'sort_order', jc.sort_order, 'active', jc.active)
      FROM local_categories jc
    UNION ALL
    SELECT 7, 'job_phase', jp.id, '', jsonb_build_object('id', jp.id, 'workspace_id', jp.workspace_id,
             'job_id', jp.job_id, 'template_phase_id', jp.template_phase_id, 'phase_order', jp.phase_order,
             'active', jp.active, 'approval_status', jp.approval_status, 'submitted_by', jp.submitted_by,
             'submitted_at', jp.submitted_at, 'verified_by', jp.verified_by, 'verified_at', jp.verified_at,
             'published_by', jp.published_by, 'published_at', jp.published_at,
             'return_reason', jp.return_reason, 'returned_by', jp.returned_by, 'returned_at', jp.returned_at)
      FROM job_phases jp
    UNION ALL
    SELECT 8, 'job_template_phase', tp.id, '', jsonb_build_object('id', tp.id, 'workspace_id', tp.workspace_id,
             'job_template_id', tp.job_template_id, 'code', tp.code, 'name', tp.name,
             'phase_order', tp.phase_order, 'active', tp.active)
      FROM template_phases tp
    UNION ALL
    SELECT 9, 'job_template_task', tt.id, '', jsonb_build_object('id', tt.id, 'workspace_id', tt.workspace_id,
             'job_template_phase_id', tt.job_template_phase_id, 'name', tt.name, 'code', tt.code,
             'step_order', tt.step_order, 'active', tt.active)
      FROM template_tasks tt
    UNION ALL
    SELECT 10, 'job_task', jt.id, '', jsonb_build_object('id', jt.id, 'workspace_id', jt.workspace_id,
             'job_phase_id', jt.job_phase_id, 'template_task_id', jt.template_task_id,
             'assigned_to', jt.assigned_to, 'active', jt.active)
      FROM job_tasks jt
    UNION ALL
    SELECT 11, 'task_outcome', o.job_task_id, o.template_task_criteria_id,
           jsonb_build_object('job_task_id', o.job_task_id, 'template_task_criteria_id', o.template_task_criteria_id,
             'numeric_value', o.numeric_value, 'scaled_label', o.scaled_label,
             'determination_note', o.determination_note, 'recorded_date', o.recorded_date)
      FROM latest_task_outcomes o
    UNION ALL
    SELECT 12, 'outcome_criteria', oc.id, '', jsonb_build_object('id', oc.id, 'workspace_id', oc.workspace_id,
             'name', oc.name, 'code', oc.code, 'unit', oc.unit, 'decimal_places', oc.decimal_places,
             'min_score', oc.min_score, 'max_score', oc.max_score, 'score_increment', oc.score_increment,
             'pass_label', oc.pass_label, 'fail_label', oc.fail_label, 'required', oc.required, 'active', oc.active)
      FROM outcome_criteria oc
    UNION ALL
    SELECT 13, 'template_task_criteria', ttc.id, '', jsonb_build_object('id', ttc.id, 'workspace_id', ttc.workspace_id,
             'job_template_task_id', ttc.job_template_task_id, 'outcome_criteria_id', ttc.outcome_criteria_id,
             'sequence_order', ttc.sequence_order, 'required_override', ttc.required_override,
             'weight_override', ttc.weight_override, 'active', ttc.active)
      FROM template_task_criteria ttc
    UNION ALL
    SELECT 14, 'rating_description', td.id, '', jsonb_build_object('id', td.id,
             'workspace_id', td.workspace_id, 'template_task_criteria_id', td.template_task_criteria_id,
             'description', td.description, 'sequence_order', td.sequence_order, 'active', td.active)
      FROM rating_description_rows td
    UNION ALL
    SELECT 15, 'phase_outcome_summary', pos.id, '', jsonb_build_object('id', pos.id, 'workspace_id', pos.workspace_id,
             'job_id', pos.job_id, 'job_phase_id', pos.job_phase_id, 'scaled_label', pos.scaled_label,
             'scaled_score', pos.scaled_score, 'summary_score', pos.summary_score,
             'total_criteria_count', pos.total_criteria_count, 'pass_count', pos.pass_count,
             'fail_count', pos.fail_count, 'conditional_count', pos.conditional_count,
             'deferred_count', pos.deferred_count, 'na_count', pos.na_count, 'narrative', pos.narrative,
             'active', pos.active, 'date_created', pos.date_created)
      FROM latest_phase_summaries pos
    UNION ALL
    SELECT 16, 'job_outcome_summary', jos.id, '', jsonb_build_object('id', jos.id, 'workspace_id', jos.workspace_id,
             'client_id', jos.client_id,
             'job_id', jos.job_id, 'scaled_label', jos.scaled_label, 'scaled_score', jos.scaled_score,
             'summary_score', jos.summary_score, 'total_criteria_count', jos.total_criteria_count,
             'pass_count', jos.pass_count, 'fail_count', jos.fail_count,
             'conditional_count', jos.conditional_count, 'deferred_count', jos.deferred_count,
             'na_count', jos.na_count, 'narrative', jos.narrative,
             'active', jos.active, 'date_created', jos.date_created)
      FROM latest_job_summaries jos
    UNION ALL
    SELECT 17, 'job_outcome_line', jol.id, '', jsonb_build_object('id', jol.id, 'workspace_id', jol.workspace_id,
             'client_id', jol.client_id,
             'job_outcome_summary_id', jol.job_outcome_summary_id, 'label', jol.label,
             'weight_or_credits', jol.weight_or_credits, 'output_value', jol.output_value,
             'output_label', jol.output_label, 'active', jol.active)
      FROM outcome_lines jol
    UNION ALL
    SELECT 18, 'staff', s.staff_id, '', jsonb_build_object('staff_id', s.staff_id, 'display_name', s.display_name)
      FROM staff_rows s
    UNION ALL
    SELECT 19, 'teacher_assignment', ta.job_id, ta.job_phase_id || ':' || ta.staff_id,
           jsonb_build_object('job_id', ta.job_id, 'job_phase_id', ta.job_phase_id,
             'staff_id', ta.staff_id, 'display_name', ta.display_name)
      FROM teacher_assignments ta
    UNION ALL
    SELECT DISTINCT 20, 'render_gate_job_id', j.id, '', to_jsonb(j.id)
      FROM job_rows j
    UNION ALL
    SELECT 21, 'render_gate_group_id', '', '', to_jsonb($7::text)
    UNION ALL
    SELECT 22, 'render_gate_sheet', 'template:' || s.template_phase_id, '', jsonb_build_object(
             'job_template_phase_id', s.template_phase_id,
             'applied_subscription_group_id', $7::text,
             'target_count', s.target_count,
             'any_workflow_entered', s.any_workflow_entered,
             'all_published', s.all_published,
             'has_data', s.has_data)
      FROM render_gate_template_sheets s
    UNION ALL
    SELECT 22, 'render_gate_sheet', 'singleton:' || s.job_phase_id, '', jsonb_build_object(
             'job_phase_id', s.job_phase_id,
             'applied_subscription_group_id', $7::text,
             'target_count', 1,
             'any_workflow_entered', s.any_workflow_entered,
             'all_published', s.all_published,
             'has_data', s.has_data)
      FROM render_gate_singletons s
  ) projection_rows
 ORDER BY kind_order, sort_key_1, sort_key_2
`

func exportSelector(req *exportpb.GetSubscriptionGroupOutcomeExportRequest) (kind, phaseCode string) {
	if req == nil {
		return "options", ""
	}
	switch selector := req.GetOutcomeSelector().(type) {
	case *exportpb.GetSubscriptionGroupOutcomeExportRequest_JobTemplatePhaseCode:
		return "phase", selector.JobTemplatePhaseCode
	case *exportpb.GetSubscriptionGroupOutcomeExportRequest_FinalOutcome:
		if selector.FinalOutcome {
			return "final", ""
		}
	}
	return "options", ""
}

func (q *PostgresSubscriptionGroupOutcomeExportQuery) GetSubscriptionGroupOutcomeExportScoped(
	ctx context.Context,
	req *exportpb.GetSubscriptionGroupOutcomeExportRequest,
	scope ports.SubscriptionGroupOutcomeExportScope,
) (*exportpb.GetSubscriptionGroupOutcomeExportResponse, error) {
	if q == nil || q.db == nil {
		return nil, fmt.Errorf("subscription group outcome export requires PostgreSQL")
	}
	if req == nil || strings.TrimSpace(req.GetSubscriptionGroupId()) == "" {
		return nil, fmt.Errorf("subscription_group_id is required")
	}
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil || strings.TrimSpace(id.WorkspaceID) == "" {
		return nil, fmt.Errorf("subscription group outcome export is unavailable")
	}

	kind, phaseCode := exportSelector(req)
	built := buildSubscriptionGroupOutcomeExportSQL(ctx, id, req, scope, kind, phaseCode)
	rows, err := adaptercore.ExecutorFromContext(ctx, q.db).QueryContext(ctx, built.statement, built.args...)
	if err != nil {
		return nil, fmt.Errorf("subscription group outcome export query: %w", err)
	}
	defer rows.Close()

	response := &exportpb.GetSubscriptionGroupOutcomeExportResponse{Success: true}
	categories := make(map[string]*exportpb.JobCategoryOption)
	categoryOrder := make([]string, 0)
	columns := make(map[string]*exportpb.JobTemplateColumn)
	columnOrder := make([]string, 0)
	clients := make(map[string]*exportpb.SubscriptionGroupOutcomeClientRow)
	clientOrder := make([]string, 0)
	cellIDs := make(map[string]map[string]struct{})

	for rows.Next() {
		var kind string
		var payload []byte
		if err := rows.Scan(&kind, &payload); err != nil {
			return nil, fmt.Errorf("subscription group outcome export scan: %w", err)
		}
		switch kind {
		case "context":
			if response.Context != nil {
				return nil, fmt.Errorf("subscription group outcome export returned duplicate context")
			}
			var value exportContextJSON
			if err := json.Unmarshal(payload, &value); err != nil {
				return nil, fmt.Errorf("subscription group outcome export context: %w", err)
			}
			response.Context = &exportpb.SubscriptionGroupOutcomeExportContext{
				SubscriptionGroupId: value.SubscriptionGroupID, SubscriptionGroupName: value.SubscriptionGroupName,
				PriceScheduleId: value.PriceScheduleID, PriceScheduleName: value.PriceScheduleName,
				PlanId: value.PlanID, PlanName: value.PlanName, Historical: value.Historical,
			}
		case "category":
			var value exportCategoryJSON
			if err := json.Unmarshal(payload, &value); err != nil {
				return nil, fmt.Errorf("subscription group outcome export category: %w", err)
			}
			if _, exists := categories[value.JobCategoryID]; exists {
				return nil, fmt.Errorf("subscription group outcome export returned duplicate category %q", value.JobCategoryID)
			}
			categories[value.JobCategoryID] = &exportpb.JobCategoryOption{
				JobCategoryId: value.JobCategoryID, Code: value.Code, Name: value.Name,
				SortOrder: value.SortOrder, FinalOutcomeAvailable: value.FinalOutcomeAvailable,
			}
			categoryOrder = append(categoryOrder, value.JobCategoryID)
		case "phase":
			var value exportPhaseJSON
			if err := json.Unmarshal(payload, &value); err != nil {
				return nil, fmt.Errorf("subscription group outcome export phase: %w", err)
			}
			category := categories[value.JobCategoryID]
			if category == nil {
				return nil, fmt.Errorf("subscription group outcome export phase references unknown category %q", value.JobCategoryID)
			}
			for _, existing := range category.JobTemplatePhases {
				if existing.GetCode() == value.Code {
					return nil, fmt.Errorf("subscription group outcome export returned duplicate phase %q", value.Code)
				}
			}
			category.JobTemplatePhases = append(category.JobTemplatePhases, &exportpb.JobTemplatePhaseOption{
				Code: value.Code, Name: value.Name, SequenceOrder: value.SequenceOrder, Ambiguous: value.Ambiguous,
			})
		case "column":
			var value exportColumnJSON
			if err := json.Unmarshal(payload, &value); err != nil {
				return nil, fmt.Errorf("subscription group outcome export column: %w", err)
			}
			if _, exists := columns[value.JobTemplateID]; exists {
				return nil, fmt.Errorf("subscription group outcome export returned duplicate column %q", value.JobTemplateID)
			}
			columns[value.JobTemplateID] = &exportpb.JobTemplateColumn{JobTemplateId: value.JobTemplateID, DisplayName: value.DisplayName}
			columnOrder = append(columnOrder, value.JobTemplateID)
		case "cell":
			var value exportCellJSON
			if err := json.Unmarshal(payload, &value); err != nil {
				return nil, fmt.Errorf("subscription group outcome export cell: %w", err)
			}
			row := clients[value.ClientID]
			if row == nil {
				row = &exportpb.SubscriptionGroupOutcomeClientRow{
					ClientId: value.ClientID, ClientName: value.ClientName,
					ClientFirstName: value.ClientFirstName, ClientLastName: value.ClientLastName,
				}
				clients[value.ClientID] = row
				clientOrder = append(clientOrder, value.ClientID)
				cellIDs[value.ClientID] = make(map[string]struct{})
			}
			if row.GetClientName() != value.ClientName || row.GetClientFirstName() != value.ClientFirstName || row.GetClientLastName() != value.ClientLastName {
				return nil, fmt.Errorf("subscription group outcome export returned inconsistent client metadata for %q", value.ClientID)
			}
			if _, duplicate := cellIDs[value.ClientID][value.JobTemplateID]; duplicate {
				return nil, fmt.Errorf("subscription group outcome export returned duplicate cell %q/%q", value.ClientID, value.JobTemplateID)
			}
			cellIDs[value.ClientID][value.JobTemplateID] = struct{}{}
			row.Cells = append(row.Cells, &exportpb.SubscriptionGroupOutcomeCell{
				JobTemplateId: value.JobTemplateID, JobPresent: value.JobPresent,
				ScaledLabel: value.ScaledLabel, ScaledScore: value.ScaledScore, SummaryScore: value.SummaryScore,
				EnrollmentEvidence: &exportpb.EnrollmentEvidence{HasMarks: value.HasMarks, HasPositiveMark: value.HasPositiveMark},
			})
		default:
			return nil, fmt.Errorf("subscription group outcome export returned unknown row kind %q", kind)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("subscription group outcome export rows: %w", err)
	}
	if response.Context == nil {
		// Preserve the non-enumerating empty shape for the application boundary.
		// The use case still returns a validation error, while in-process HTTP
		// composition can map missing/foreign/unauthorized groups uniformly to 404
		// without conflating them with a database execution failure.
		return response, nil
	}

	sort.SliceStable(categoryOrder, func(i, j int) bool {
		a, b := categories[categoryOrder[i]], categories[categoryOrder[j]]
		if a.GetSortOrder() != b.GetSortOrder() {
			return a.GetSortOrder() < b.GetSortOrder()
		}
		an, bn := strings.ToLower(strings.TrimSpace(a.GetName())), strings.ToLower(strings.TrimSpace(b.GetName()))
		if an != bn {
			return an < bn
		}
		return a.GetJobCategoryId() < b.GetJobCategoryId()
	})
	for _, categoryID := range categoryOrder {
		category := categories[categoryID]
		sort.SliceStable(category.JobTemplatePhases, func(i, j int) bool {
			return phaseOptionLess(category.JobTemplatePhases[i], category.JobTemplatePhases[j])
		})
		response.JobCategories = append(response.JobCategories, category)
	}

	sort.SliceStable(columnOrder, func(i, j int) bool {
		a, b := columns[columnOrder[i]], columns[columnOrder[j]]
		an, bn := strings.ToLower(columnName(a)), strings.ToLower(columnName(b))
		if an != bn {
			return an < bn
		}
		return a.GetJobTemplateId() < b.GetJobTemplateId()
	})
	for _, columnID := range columnOrder {
		response.JobTemplateColumns = append(response.JobTemplateColumns, columns[columnID])
	}
	for _, clientID := range clientOrder {
		row := clients[clientID]
		byTemplate := make(map[string]*exportpb.SubscriptionGroupOutcomeCell, len(row.Cells))
		for _, cell := range row.Cells {
			byTemplate[cell.GetJobTemplateId()] = cell
		}
		row.Cells = row.Cells[:0]
		for _, columnID := range columnOrder {
			if cell := byTemplate[columnID]; cell != nil {
				row.Cells = append(row.Cells, cell)
			}
		}
	}
	sort.SliceStable(clientOrder, func(i, j int) bool {
		a, b := clients[clientOrder[i]], clients[clientOrder[j]]
		al, bl := strings.ToLower(strings.TrimSpace(a.GetClientLastName())), strings.ToLower(strings.TrimSpace(b.GetClientLastName()))
		if al != bl {
			if al == "" || bl == "" {
				return al != ""
			}
			return al < bl
		}
		an, bn := strings.ToLower(strings.TrimSpace(a.GetClientName())), strings.ToLower(strings.TrimSpace(b.GetClientName()))
		if an != bn {
			return an < bn
		}
		return a.GetClientId() < b.GetClientId()
	})
	for _, clientID := range clientOrder {
		response.ClientRows = append(response.ClientRows, clients[clientID])
	}
	return response, nil
}

func columnName(column *exportpb.JobTemplateColumn) string {
	if name := strings.TrimSpace(column.GetDisplayName()); name != "" {
		return name
	}
	return column.GetJobTemplateId()
}

func phaseOptionLess(a, b *exportpb.JobTemplatePhaseOption) bool {
	if a.GetSequenceOrder() != b.GetSequenceOrder() {
		return a.GetSequenceOrder() < b.GetSequenceOrder()
	}
	if a.GetCode() != b.GetCode() {
		return a.GetCode() < b.GetCode()
	}
	return strings.ToLower(strings.TrimSpace(a.GetName())) < strings.ToLower(strings.TrimSpace(b.GetName()))
}

func buildSubscriptionGroupOutcomeExportSQL(
	ctx context.Context,
	id *identity.RequestIdentity,
	req *exportpb.GetSubscriptionGroupOutcomeExportRequest,
	scope ports.SubscriptionGroupOutcomeExportScope,
	selectorKind, phaseCode string,
) exportScopeSQL {
	staffScoped := false
	staffClause := ""
	var staffArgs []any
	if narrowStaffReportsToReachableJobs && !scope.WorkspaceWide {
		if _, applies := principalscope.StaffRowScope(ctx); applies {
			staffScoped = true
			staffClause, staffArgs = principalscope.StaffReachableJobClause(ctx, "j", 9)
		}
	}
	args := []any{id.WorkspaceID, req.GetSubscriptionGroupId(), id.WorkspaceUserID, scope.WorkspaceWide,
		req.GetJobCategoryId(), selectorKind, phaseCode, staffScoped}
	args = append(args, staffArgs...)
	return exportScopeSQL{
		statement: renderOutcomeExportTables(outcomeExportCTEs(staffClause) + outcomeExportRowsSQL),
		args:      args, staffScoped: staffScoped,
	}
}

// outcomeExportCTEs is shared verbatim with the report resolver. Every graph
// edge that carries workspace_id is compared with the trusted session
// workspace, including child rows whose foreign keys alone are not a tenant
// boundary. Historical groups retain inactive member/job/template rows, while
// active groups require all three to be active. A present local template owns
// category identity; only a truly missing historical template falls back to
// the frozen job category.
func outcomeExportCTEs(staffClause string) string {
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
         COALESCE(evidence.has_marks, false) AS has_marks,
         COALESCE(evidence.has_positive_mark, false) AS has_positive_mark
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
      SELECT count(*) FILTER (WHERE outcome.numeric_value IS NOT NULL) > 0 AS has_marks,
             count(*) FILTER (WHERE outcome.numeric_value > 0) > 0 AS has_positive_mark
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

const outcomeExportRowsSQL = `
SELECT kind, payload
  FROM (
    SELECT 0 AS kind_order, 'context'::text AS kind, ''::text AS sort_key_1, ''::text AS sort_key_2,
           jsonb_build_object(
             'subscription_group_id', id,
             'subscription_group_name', name,
             'price_schedule_id', price_schedule_id,
             'price_schedule_name', price_schedule_name,
             'plan_id', plan_id,
             'plan_name', plan_name,
             'historical', historical
           ) AS payload
      FROM group_context
    UNION ALL
    SELECT 1, 'category', job_category_id, '',
           jsonb_build_object(
             'job_category_id', job_category_id, 'code', code, 'name', name,
             'sort_order', sort_order, 'final_outcome_available', final_outcome_available
           )
      FROM category_options
    UNION ALL
    SELECT 2, 'phase', job_category_id, code,
           jsonb_build_object(
             'job_category_id', job_category_id, 'code', code, 'name', name,
             'sequence_order', sequence_order, 'ambiguous', ambiguous
           )
      FROM phase_options
    UNION ALL
    SELECT 3, 'column', job_template_id, '',
           jsonb_build_object('job_template_id', job_template_id, 'display_name', display_name)
      FROM matrix_columns
    UNION ALL
    SELECT 4, 'cell', client_id, job_template_id,
           jsonb_build_object(
             'client_id', client_id, 'client_name', client_name,
             'client_first_name', client_first_name, 'client_last_name', client_last_name,
             'job_template_id', job_template_id, 'job_present', job_present,
             'scaled_label', scaled_label, 'scaled_score', scaled_score,
             'summary_score', summary_score,
             'has_marks', has_marks, 'has_positive_mark', has_positive_mark
           )
      FROM matrix_cells
  ) rows
 ORDER BY kind_order, sort_key_1, sort_key_2
`

func renderOutcomeExportTables(statement string) string {
	return strings.NewReplacer(
		"{{subscription_group}}", entityid.SubscriptionGroup,
		"{{subscription_group_member}}", entityid.SubscriptionGroupMember,
		"{{subscription_group_workspace_user}}", entityid.SubscriptionGroupWorkspaceUser,
		"{{subscription}}", entityid.Subscription,
		"{{workspace_user}}", entityid.WorkspaceUser,
		"{{client}}", entityid.Client,
		"{{job}}", entityid.Job,
		"{{job_template}}", entityid.JobTemplate,
		"{{job_category}}", entityid.JobCategory,
		"{{job_template_phase}}", entityid.JobTemplatePhase,
		"{{job_phase}}", entityid.JobPhase,
		"{{job_task}}", entityid.JobTask,
		"{{task_outcome}}", entityid.TaskOutcome,
		"{{phase_outcome_summary}}", entityid.PhaseOutcomeSummary,
		"{{job_outcome_summary}}", entityid.JobOutcomeSummary,
		"{{price_schedule}}", entityid.PriceSchedule,
		"{{plan}}", entityid.Plan,
		"{{subscription_group_document_template}}", entityid.SubscriptionGroupDocumentTemplate,
		"{{document_template}}", entityid.DocumentTemplate,
		"{{client_attribute}}", entityid.ClientAttribute,
		"{{attribute}}", entityid.Attribute,
		"{{plan_attribute}}", entityid.PlanAttribute,
		"{{job_template_task}}", entityid.JobTemplateTask,
		"{{template_task_criteria}}", entityid.TemplateTaskCriteria,
		"{{template_task_criteria_rating_description}}", entityid.TemplateTaskCriteriaRatingDescription,
		"{{outcome_criteria}}", entityid.OutcomeCriteria,
		"{{job_outcome_line}}", entityid.JobOutcomeLine,
		"{{staff}}", entityid.Staff,
		"{{user}}", entityid.User,
		"{{subscription_group_product_plan_staff}}", entityid.SubscriptionGroupProductPlanStaff,
		"{{product_plan}}", entityid.ProductPlan,
		"{{product_plan_staff}}", entityid.ProductPlanStaff,
	).Replace(statement)
}

func (q *PostgresSubscriptionGroupOutcomeExportQuery) ResolveSubscriptionGroupOutcomeDocumentForRenderScoped(
	ctx context.Context,
	req *exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest,
	scope ports.SubscriptionGroupOutcomeExportScope,
) (*exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse, error) {
	if q == nil || q.db == nil {
		return nil, fmt.Errorf("subscription group outcome document resolver requires PostgreSQL")
	}
	if req == nil {
		return nil, fmt.Errorf("subscription group outcome document resolver received an unsupported request")
	}
	switch req.GetRenderProfile() {
	case bindingpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1:
		if strings.TrimSpace(req.GetJobCategoryId()) == "" {
			return nil, fmt.Errorf("matrix document resolver requires an exact job category")
		}
	case bindingpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_CLIENT_PHASE_OUTCOME_REPORT_V1:
		if strings.TrimSpace(req.GetJobCategoryId()) != "" {
			return nil, fmt.Errorf("client phase document resolver requires whole-report category scope")
		}
	default:
		return nil, fmt.Errorf("subscription group outcome document resolver received an unsupported request")
	}
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil || strings.TrimSpace(id.WorkspaceID) == "" {
		return renderDocumentMiss(), nil
	}

	built := buildSubscriptionGroupOutcomeDocumentResolverSQL(ctx, id, req, scope)
	rows, err := adaptercore.ExecutorFromContext(ctx, q.db).QueryContext(ctx, built.statement, built.args...)
	if err != nil {
		return nil, fmt.Errorf("subscription group outcome document resolver query: %w", err)
	}
	defer rows.Close()

	type candidate struct {
		profile, category, container, key string
		rank                              int
	}
	var candidates []candidate
	for rows.Next() {
		var value candidate
		if err := rows.Scan(&value.profile, &value.category, &value.container, &value.key, &value.rank); err != nil {
			return nil, fmt.Errorf("subscription group outcome document resolver scan: %w", err)
		}
		candidates = append(candidates, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("subscription group outcome document resolver rows: %w", err)
	}
	if len(candidates) == 0 {
		return renderDocumentMiss(), nil
	}
	if len(candidates) > 1 && candidates[0].rank == candidates[1].rank {
		return nil, fmt.Errorf("subscription group outcome document resolver found ambiguous candidates")
	}
	profileValue, ok := bindingpb.RenderProfile_value[candidates[0].profile]
	if !ok {
		return nil, fmt.Errorf("subscription group outcome document resolver returned an unknown render profile")
	}
	return &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse{
		Found: true, Success: true,
		Document: &exportpb.ResolvedSubscriptionGroupOutcomeDocument{
			StorageContainer: strings.TrimSpace(candidates[0].container),
			StorageKey:       strings.TrimSpace(candidates[0].key),
			RenderProfile:    bindingpb.RenderProfile(profileValue),
			JobCategoryId:    candidates[0].category,
		},
	}, nil
}

func buildSubscriptionGroupOutcomeDocumentResolverSQL(
	ctx context.Context,
	id *identity.RequestIdentity,
	req *exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest,
	scope ports.SubscriptionGroupOutcomeExportScope,
) exportScopeSQL {
	wholeReport := req.GetRenderProfile() == bindingpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_CLIENT_PHASE_OUTCOME_REPORT_V1
	categoryAvailabilityPredicate := `EXISTS (
       SELECT 1 FROM outcome_capable_jobs cj
        JOIN category_options co ON co.job_category_id = cj.category_bucket_id
       WHERE co.job_category_id = $5
     )`
	bindingCategoryPredicate := `b.job_category_id = $5`
	if wholeReport {
		categoryAvailabilityPredicate = `EXISTS (SELECT 1 FROM outcome_capable_jobs)`
		bindingCategoryPredicate = `b.job_category_id IS NULL AND NULLIF($5, '') IS NULL`
	}
	staffScoped := false
	staffClause := ""
	var staffArgs []any
	if narrowStaffReportsToReachableJobs && !scope.WorkspaceWide {
		if _, applies := principalscope.StaffRowScope(ctx); applies {
			staffScoped = true
			staffClause, staffArgs = principalscope.StaffReachableJobClause(ctx, "j", 9)
		}
	}
	args := []any{id.WorkspaceID, req.GetSubscriptionGroupId(), id.WorkspaceUserID, scope.WorkspaceWide,
		req.GetJobCategoryId(), "final", "", staffScoped}
	args = append(args, staffArgs...)
	next := len(args) + 1
	profileP, expectedPlanP, expectedScheduleP := next, next+1, next+2
	asOfP, purposeP, publishedP := next+3, next+4, next+5
	args = append(args, req.GetRenderProfile().String(), optionalStringValue(req.ExpectedPlanId),
		optionalStringValue(req.ExpectedPriceScheduleId), time.Now().UTC(), subscriptionGroupOutcomeDocumentPurpose, versionStatusPublished)

	resolverSQL := fmt.Sprintf(`
, resolved_group AS (
  SELECT g.*
    FROM group_context g
   WHERE g.plan_id IS NOT DISTINCT FROM $%d::text
     AND g.price_schedule_id IS NOT DISTINCT FROM $%d::text
     AND %s
), candidates AS (
  SELECT b.render_profile, b.job_category_id, dt.storage_container, dt.storage_key, b.version,
         CASE
           WHEN b.plan_id = g.plan_id AND b.price_schedule_id = g.price_schedule_id THEN 0
           WHEN b.plan_id = g.plan_id AND b.price_schedule_id IS NULL THEN 1
           WHEN b.plan_id IS NULL AND b.price_schedule_id = g.price_schedule_id THEN 2
           WHEN b.plan_id IS NULL AND b.price_schedule_id IS NULL THEN 3
         END AS match_rank
    FROM resolved_group g
    JOIN {{subscription_group_document_template}} b
      ON b.workspace_id = $1
     AND b.render_profile = $%d
     AND %s
     AND b.active = true
     AND b.version_status = $%d
     AND (b.validity_start IS NULL OR b.validity_start <= $%d)
     AND (b.validity_end IS NULL OR $%d < b.validity_end)
     AND (b.plan_id = g.plan_id OR b.plan_id IS NULL)
     AND (b.price_schedule_id = g.price_schedule_id OR b.price_schedule_id IS NULL)
    JOIN {{document_template}} dt
      ON dt.id = b.document_template_id
     AND dt.workspace_id = b.workspace_id
     AND dt.active = true
     AND dt.status = 'active'
     AND dt.template_type = 'docx'
     AND dt.document_purpose = $%d
     AND NULLIF(btrim(dt.storage_container), '') IS NOT NULL
     AND NULLIF(btrim(dt.storage_key), '') IS NOT NULL
   ORDER BY match_rank, b.version DESC
   LIMIT 2
)
SELECT render_profile, COALESCE(job_category_id, ''), storage_container, storage_key, match_rank
  FROM candidates
 ORDER BY match_rank, version DESC`,
		expectedPlanP, expectedScheduleP, categoryAvailabilityPredicate, profileP, bindingCategoryPredicate, publishedP, asOfP, asOfP, purposeP)
	return exportScopeSQL{
		statement:   renderOutcomeExportTables(outcomeExportCTEs(staffClause) + resolverSQL),
		args:        args,
		staffScoped: staffScoped,
	}
}

func optionalStringValue(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func renderDocumentMiss() *exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse {
	return &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse{Found: false, Success: true}
}
