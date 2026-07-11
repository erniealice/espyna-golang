//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	criteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
	matrixpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/outcome_matrix"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	internalregistry "github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
)

// init self-registers the postgres outcome-matrix query with the composition-
// root factory registry (mirrors rbac/permission_query.go). The registry file
// is tag-free; only THIS file (build-tagged postgresql) calls Register, so
// non-postgres builds never wire it and the composition initializer degrades to
// a nil port (fail-closed empty response).
func init() {
	internalregistry.RegisterOutcomeMatrixFactory(func(db any) any {
		sqlDB, ok := db.(*sql.DB)
		if !ok {
			return nil
		}
		return NewPostgresOutcomeMatrixQuery(sqlDB)
	})
}

// PostgresOutcomeMatrixQuery implements the GENERATED
// operationv1.OutcomeMatrixServiceServer interface (Q-PROTO-MODE: service{rpc}).
// Embedding UnimplementedOutcomeMatrixServiceServer is MANDATORY — the
// interface carries an unexported marker method that only the Unimplemented
// struct can satisfy.
type PostgresOutcomeMatrixQuery struct {
	matrixpb.UnimplementedOutcomeMatrixServiceServer
	db *sql.DB
}

// NewPostgresOutcomeMatrixQuery constructs the PG-backed outcome-matrix reader.
func NewPostgresOutcomeMatrixQuery(db *sql.DB) matrixpb.OutcomeMatrixServiceServer {
	return &PostgresOutcomeMatrixQuery{db: db}
}

// GetOutcomeMatrix returns the column tree (from the template) + the cell rows
// (each scoped client's resolved task_outcomes) for one job_template.
//
// Scoping (every query is workspace_id-bound from the SESSION identity, never a
// request param):
//   - scope=ALL: every client under the template in the workspace (no
//     principalscope predicate). The widen is authorized upstream by the use
//     case's workspace:list authcheck (decisions.md Q-SCOPE-ALL; query-brief §4
//     — there is no ready-made operator-visibility resolver JOIN, the locked
//     design is a permission-gated widen over the same query, still
//     workspace-bounded).
//   - scope=MINE / UNSPECIFIED: principalscope.StaffReachableClientClause
//     narrows rows to the acting staff's reachable clients (fail-closed: a
//     non-staff principal here yields zero editable rows).
func (a *PostgresOutcomeMatrixQuery) GetOutcomeMatrix(
	ctx context.Context,
	req *matrixpb.GetOutcomeMatrixRequest,
) (*matrixpb.GetOutcomeMatrixResponse, error) {
	if req == nil || req.GetJobTemplateId() == "" {
		return &matrixpb.GetOutcomeMatrixResponse{Success: true}, nil
	}

	// workspace_id from the session identity — required for multi-tenancy.
	// FromContext (not identity.Must) so a missing identity fails closed to an
	// empty response rather than panicking.
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil || id.WorkspaceID == "" {
		return &matrixpb.GetOutcomeMatrixResponse{JobTemplateId: req.GetJobTemplateId(), Success: true}, nil
	}
	workspaceID := id.WorkspaceID
	jobTemplateID := req.GetJobTemplateId()

	templateName, err := a.loadTemplateName(ctx, jobTemplateID, workspaceID)
	if err != nil {
		return nil, err
	}

	phases, err := a.loadColumnTree(ctx, jobTemplateID, workspaceID)
	if err != nil {
		return nil, err
	}

	rows, err := a.loadRows(ctx, req, workspaceID)
	if err != nil {
		return nil, err
	}

	return &matrixpb.GetOutcomeMatrixResponse{
		JobTemplateId:   jobTemplateID,
		JobTemplateName: templateName,
		Phases:          phases,
		Rows:            rows,
		Success:         true,
	}, nil
}

// loadTemplateName resolves the template display name (tolerates a NULL/shared
// workspace_id but rejects a template owned by a DIFFERENT workspace).
func (a *PostgresOutcomeMatrixQuery) loadTemplateName(ctx context.Context, jobTemplateID, workspaceID string) (string, error) {
	q := `SELECT name FROM ` + entityid.JobTemplate + `
	WHERE id = $1 AND (workspace_id = $2 OR workspace_id IS NULL) AND active`
	var name string
	err := a.db.QueryRowContext(ctx, q, jobTemplateID, workspaceID).Scan(&name)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("outcome_matrix: load template name: %w", err)
	}
	return name, nil
}

// loadColumnTree builds the phase→task→criterion column tree from the TEMPLATE
// decomposition (job_template_phase → job_template_task → template_task_criteria
// → outcome_criteria), ordered by phase_order, step_order, sequence_order.
// Independent of which cells are recorded — every template leaf is a column.
// The phase-header label is composed here (composePhaseLabel): when the phase
// carries a sub-deliverable (output_product_variant_id → product_variant) its
// variant name is appended as a parenthetical ("Semester 1 (Visual Arts)").
func (a *PostgresOutcomeMatrixQuery) loadColumnTree(ctx context.Context, jobTemplateID, workspaceID string) ([]*matrixpb.PhaseColumn, error) {
	q := `
SELECT
    jtp.id                        AS phase_id,
    jtp.name                      AS phase_name,
    pv.sku                        AS variant_name,
    jtp.phase_order,
    jtt.id                        AS task_id,
    jtt.name                      AS task_name,
    jtt.step_order,
    ttc.sequence_order,
    oc.id                         AS criteria_id,
    oc.name                       AS criteria_name,
    oc.criteria_type,
    oc.unit,
    oc.decimal_places,
    oc.min_score,
    oc.max_score,
    oc.score_increment,
    oc.pass_label,
    oc.fail_label,
    oc.max_text_length,
    oc.text_prompt,
    oc.weight,
    oc.required
FROM ` + entityid.JobTemplatePhase + ` jtp
JOIN ` + entityid.JobTemplateTask + ` jtt
       ON jtt.job_template_phase_id = jtp.id AND jtt.active
JOIN ` + entityid.TemplateTaskCriteria + ` ttc
       ON ttc.job_template_task_id = jtt.id AND ttc.active
JOIN ` + entityid.OutcomeCriteria + ` oc
       ON oc.id = ttc.outcome_criteria_id AND oc.active
LEFT JOIN ` + entityid.ProductVariant + ` pv
       ON pv.id = jtp.output_product_variant_id AND pv.active
WHERE jtp.job_template_id = $1
  AND jtp.active
  AND EXISTS (
        SELECT 1 FROM ` + entityid.JobTemplate + ` jt
        WHERE jt.id = $1
          AND (jt.workspace_id = $2 OR jt.workspace_id IS NULL)
          AND jt.active
      )
ORDER BY jtp.phase_order, jtt.step_order, ttc.sequence_order`

	rows, err := a.db.QueryContext(ctx, q, jobTemplateID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("outcome_matrix: column tree query: %w", err)
	}
	defer rows.Close()

	var phases []*matrixpb.PhaseColumn
	phaseByID := map[string]*matrixpb.PhaseColumn{}
	taskByID := map[string]*matrixpb.TaskColumn{}

	for rows.Next() {
		var (
			phaseID, phaseName       string
			variantName              sql.NullString
			phaseOrder               int32
			taskID, taskName         string
			stepOrder                int32
			seqOrder                 int32
			criteriaID, criteriaName string
			criteriaType             sql.NullString
			unit                     sql.NullString
			decimalPlaces            sql.NullInt64
			minScore, maxScore       sql.NullInt64
			scoreIncrement           sql.NullFloat64
			passLabel, failLabel     sql.NullString
			maxTextLength            sql.NullInt64
			textPrompt               sql.NullString
			weight                   sql.NullFloat64
			required                 sql.NullBool
		)
		if err := rows.Scan(
			&phaseID, &phaseName, &variantName, &phaseOrder,
			&taskID, &taskName, &stepOrder,
			&seqOrder,
			&criteriaID, &criteriaName, &criteriaType,
			&unit, &decimalPlaces, &minScore, &maxScore, &scoreIncrement,
			&passLabel, &failLabel, &maxTextLength, &textPrompt, &weight, &required,
		); err != nil {
			return nil, fmt.Errorf("outcome_matrix: scan column tree: %w", err)
		}

		phase := phaseByID[phaseID]
		if phase == nil {
			phase = &matrixpb.PhaseColumn{
				JobTemplatePhaseId: phaseID,
				Label:              composePhaseLabel(phaseName, variantName),
				SequenceOrder:      phaseOrder,
			}
			phaseByID[phaseID] = phase
			phases = append(phases, phase)
		}

		task := taskByID[taskID]
		if task == nil {
			task = &matrixpb.TaskColumn{
				JobTemplateTaskId: taskID,
				Label:             taskName,
				SequenceOrder:     stepOrder,
			}
			taskByID[taskID] = task
			phase.Tasks = append(phase.Tasks, task)
		}

		crit := &matrixpb.CriterionColumn{
			ColumnKey:     taskID + ":" + criteriaID,
			SequenceOrder: seqOrder,
			Required:      required.Valid && required.Bool,
			Criteria: &criteriapb.OutcomeCriteria{
				Id:             criteriaID,
				Name:           criteriaName,
				CriteriaType:   parseCriteriaType(criteriaType),
				Unit:           nullStringPtr(unit),
				DecimalPlaces:  nullInt32Ptr(decimalPlaces),
				MinScore:       nullInt32Ptr(minScore),
				MaxScore:       nullInt32Ptr(maxScore),
				ScoreIncrement: nullFloat64Ptr(scoreIncrement),
				PassLabel:      nullStringPtr(passLabel),
				FailLabel:      nullStringPtr(failLabel),
				MaxTextLength:  nullInt32Ptr(maxTextLength),
				TextPrompt:     nullStringPtr(textPrompt),
				Weight:         nullFloat64Val(weight),
				Required:       required.Valid && required.Bool,
				Active:         true,
			},
		}
		task.Criteria = append(task.Criteria, crit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("outcome_matrix: column tree rows: %w", err)
	}
	return phases, nil
}

// loadRows runs the single batched cells query (DISTINCT ON most-recent active
// task_outcome), scoped to the workspace + template + optional section/product
// + principalscope (MINE only). Every template leaf under a client's job_task
// yields a cell; the cell is blank+editable when no outcome exists yet.
func (a *PostgresOutcomeMatrixQuery) loadRows(ctx context.Context, req *matrixpb.GetOutcomeMatrixRequest, workspaceID string) ([]*matrixpb.OutcomeRow, error) {
	args := []any{req.GetJobTemplateId(), workspaceID}
	where := "WHERE j.job_template_id = $1 AND j.workspace_id = $2 AND j.active"
	nextParam := 3

	if req.SectionId != nil && req.GetSectionId() != "" {
		// workspace_id bound INSIDE the subquery too — a foreign-workspace
		// section id must never influence the roster.
		where += fmt.Sprintf(
			" AND j.client_id IN (SELECT client_id FROM "+entityid.SubscriptionGroupMember+" WHERE subscription_group_id = $%d AND workspace_id = $2 AND active = true)",
			nextParam)
		args = append(args, req.GetSectionId())
		nextParam++
	}
	if req.ProductId != nil && req.GetProductId() != "" {
		where += fmt.Sprintf(" AND j.output_product_id = $%d", nextParam)
		args = append(args, req.GetProductId())
		nextParam++
	}

	// principalscope MINE narrowing — omitted entirely for scope=ALL (the widen
	// is authorized upstream by the use case's workspace:list gate). For any
	// non-staff principal, MINE is undefined (assigned_to/recorded_by reference
	// staff.id, which a non-staff identity can never hold) — fail closed to
	// ZERO rows rather than fall through unscoped; operators reach the roster
	// only via the authorized ALL widen.
	if req.GetScope() != matrixpb.OutcomeMatrixScope_OUTCOME_MATRIX_SCOPE_ALL {
		if _, applies := principalscope.StaffRowScope(ctx); !applies {
			return nil, nil
		}
		clause, scopeArgs := principalscope.StaffReachableJobClause(ctx, "j", nextParam)
		where += clause
		args = append(args, scopeArgs...)
		nextParam += len(scopeArgs)
	}

	q := `
SELECT DISTINCT ON (j.client_id, jt.id, ttc.id)
       j.client_id,
       jt.id                          AS job_task_id,
       ttc.job_template_task_id       AS job_template_task_id,
       ttc.outcome_criteria_id        AS criteria_id,
       t.id                           AS outcome_id,
       t.numeric_value,
       t.text_value,
       t.categorical_value,
       t.pass_fail_value,
       COALESCE(t.recorded_by, '')    AS recorded_by,
       COALESCE(jt.assigned_to, '')   AS assigned_to
FROM ` + entityid.Job + ` j
JOIN ` + entityid.JobPhase + ` jp
       ON jp.job_id = j.id AND jp.active
JOIN ` + entityid.JobTask + ` jt
       ON jt.job_phase_id = jp.id AND jt.active
JOIN ` + entityid.TemplateTaskCriteria + ` ttc
       ON ttc.job_template_task_id = jt.template_task_id AND ttc.active
LEFT JOIN ` + entityid.TaskOutcome + ` t
       ON t.job_task_id = jt.id
      AND t.criteria_version_id = ttc.outcome_criteria_id
      AND t.active
` + where + `
ORDER BY j.client_id, jt.id, ttc.id, t.recorded_date DESC NULLS LAST, t.id DESC`

	rows, err := a.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("outcome_matrix: cells query: %w", err)
	}
	defer rows.Close()

	// Acting staff for the editable computation — from the SESSION identity,
	// never a request param. staffOK is false for any non-staff principal, so
	// scope=ALL viewers (operators) get read-only cells across the board.
	actingStaff, isStaff := principalscope.StaffRowScope(ctx)
	staffOK := isStaff && actingStaff != ""

	var out []*matrixpb.OutcomeRow
	rowByClient := map[string]*matrixpb.OutcomeRow{}

	for rows.Next() {
		var (
			clientID       string
			jobTaskID      string
			jobTemplateTID string
			criteriaID     string
			outcomeID      sql.NullString
			numericValue   sql.NullFloat64
			textValue      sql.NullString
			categorical    sql.NullString
			passFail       sql.NullBool
			recordedBy     string
			assignedTo     string
		)
		if err := rows.Scan(
			&clientID, &jobTaskID, &jobTemplateTID, &criteriaID,
			&outcomeID, &numericValue, &textValue, &categorical, &passFail, &recordedBy, &assignedTo,
		); err != nil {
			return nil, fmt.Errorf("outcome_matrix: scan cells: %w", err)
		}

		row := rowByClient[clientID]
		if row == nil {
			row = &matrixpb.OutcomeRow{
				ClientId:    clientID,
				ClientLabel: clientID, // opaque id; the view resolves a display name
				Cells:       map[string]*matrixpb.OutcomeCell{},
			}
			rowByClient[clientID] = row
			out = append(out, row)
		}

		hasOutcome := outcomeID.Valid && outcomeID.String != ""
		editable := computeCellEditable(hasOutcome, staffOK, recordedBy, assignedTo, actingStaff, jobTaskID)

		cell := &matrixpb.OutcomeCell{
			OutcomeId:  nullStringVal(outcomeID),
			JobTaskId:  jobTaskID,
			RecordedBy: recordedBy,
			Editable:   editable,
		}
		if numericValue.Valid {
			v := numericValue.Float64
			cell.NumericValue = &v
		}
		if textValue.Valid {
			v := textValue.String
			cell.TextValue = &v
		}
		if categorical.Valid {
			v := categorical.String
			cell.CategoricalValue = &v
		}
		if passFail.Valid {
			v := passFail.Bool
			cell.PassFailValue = &v
		}

		row.Cells[jobTemplateTID+":"+criteriaID] = cell
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("outcome_matrix: cells rows: %w", err)
	}
	return out, nil
}

// composePhaseLabel renders one phase column-group header. When the phase carries
// a sub-deliverable (job_template_phase.output_product_variant_id → product_variant)
// the variant name is appended as a parenthetical ("Semester 1 (Visual Arts)") so a
// merged, multi-variant deliverable's otherwise same-named phase groups read as
// distinct subjects; a phase with no variant renders its bare name. This is the ONE
// label-composition site: the matrix proto's PhaseColumn.Label is a resolved display
// string, so composing here (adapter) matches the existing adapter-resolves-server-
// side-display-labels convention (staff_name / template name) and needs no new proto
// field on PhaseColumn. Generic: no vertical vocabulary — the strand name is DATA
// carried by the product_variant row.
func composePhaseLabel(name string, variantName sql.NullString) string {
	if variantName.Valid {
		if v := strings.TrimSpace(variantName.String); v != "" {
			return name + " (" + v + ")"
		}
	}
	return name
}

// computeCellEditable decides whether the acting STAFF principal may edit a matrix
// cell. A non-staff principal (staffOK=false) never edits (operators reach the
// roster read-only via the authorized ALL widen). RECORDED cells (an outcome
// exists) keep recorder-only semantics UNCHANGED — editable iff the acting staff is
// the recorder. EMPTY cells (no outcome yet) are editable only by the staff the
// cell's task is ASSIGNED to (job_task.assigned_to): on a merged, multi-deliverer
// class this stops one strand's teacher from entering grades into the other
// strand's unassessed cells (design §E hazard). An unassigned empty cell
// (assigned_to == "") is never editable (fail-closed).
func computeCellEditable(hasOutcome, staffOK bool, recordedBy, assignedTo, actingStaff, jobTaskID string) bool {
	if !staffOK {
		return false
	}
	if hasOutcome {
		return recordedBy == actingStaff
	}
	return jobTaskID != "" && assignedTo == actingStaff
}

// parseCriteriaType maps the stored enum-name string (e.g.
// "CRITERIA_TYPE_NUMERIC_SCORE", written by the grade-loader) to the enum value;
// unknown/NULL → UNSPECIFIED.
func parseCriteriaType(s sql.NullString) enumspb.CriteriaType {
	if !s.Valid {
		return enumspb.CriteriaType_CRITERIA_TYPE_UNSPECIFIED
	}
	if v, ok := enumspb.CriteriaType_value[s.String]; ok {
		return enumspb.CriteriaType(v)
	}
	return enumspb.CriteriaType_CRITERIA_TYPE_UNSPECIFIED
}

func nullStringPtr(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	v := s.String
	return &v
}

func nullStringVal(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return s.String
}

func nullInt32Ptr(n sql.NullInt64) *int32 {
	if !n.Valid {
		return nil
	}
	v := int32(n.Int64)
	return &v
}

func nullFloat64Ptr(f sql.NullFloat64) *float64 {
	if !f.Valid {
		return nil
	}
	v := f.Float64
	return &v
}

func nullFloat64Val(f sql.NullFloat64) float64 {
	if !f.Valid {
		return 0
	}
	return f.Float64
}
