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
	HasMarks        bool     `json:"has_marks"`
	HasPositiveMark bool     `json:"has_positive_mark"`
}

type exportScopeSQL struct {
	statement   string
	args        []any
	staffScoped bool
}

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
				ScaledLabel: value.ScaledLabel, ScaledScore: value.ScaledScore,
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
	if !scope.WorkspaceWide {
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
      SELECT pos.scaled_label, pos.scaled_score
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
      SELECT jos.scaled_label, jos.scaled_score
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
	if req == nil || req.GetRenderProfile() != bindingpb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1 {
		return nil, fmt.Errorf("subscription group outcome document resolver received an unsupported request")
	}
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil || strings.TrimSpace(id.WorkspaceID) == "" {
		return renderDocumentMiss(), nil
	}

	staffScoped := false
	staffClause := ""
	var staffArgs []any
	if !scope.WorkspaceWide {
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
     AND EXISTS (
       SELECT 1 FROM outcome_capable_jobs cj
        JOIN category_options co ON co.job_category_id = cj.effective_category_id
       WHERE co.job_category_id = $5
     )
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
     AND b.job_category_id = $5
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
SELECT render_profile, job_category_id, storage_container, storage_key, match_rank
  FROM candidates
 ORDER BY match_rank, version DESC`,
		expectedPlanP, expectedScheduleP, profileP, publishedP, asOfP, asOfP, purposeP)
	statement := renderOutcomeExportTables(outcomeExportCTEs(staffClause) + resolverSQL)
	rows, err := adaptercore.ExecutorFromContext(ctx, q.db).QueryContext(ctx, statement, args...)
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

func optionalStringValue(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func renderDocumentMiss() *exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse {
	return &exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse{Found: false, Success: true}
}
