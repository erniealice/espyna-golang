//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/principalscope"
	portsdomain "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// Compile-time proof the postgres adapter satisfies the hand-written Q26 port
// (schema-proposal.md §9.1, interfaces.md §2b).
var _ portsdomain.TaskOutcomeConditionalWriter = (*PostgresTaskOutcomeRepository)(nil)

// taskOutcomeReturningCols is the RETURNING/SELECT projection scanTOFields
// expects (same order and COALESCEs as GetTaskOutcomeItemPageData).
const taskOutcomeReturningCols = `
	id, COALESCE(job_task_id, '') AS job_task_id,
	COALESCE(criteria_version_id, '') AS criteria_version_id,
	COALESCE(criteria_type, '') AS criteria_type,
	COALESCE(is_ad_hoc, false) AS is_ad_hoc, numeric_value, text_value, categorical_value,
	pass_fail_value, COALESCE(determination, '') AS determination,
	COALESCE(determination_source, '') AS determination_source,
	determination_note, auto_proposed_determination,
	COALESCE(recorded_by, '') AS recorded_by, recorded_date, reviewed_by, reviewed_date,
	COALESCE(attachment_ids, '') AS attachment_ids, revision_of_id, revision_number,
	active, date_created, date_modified`

// conditionalTx resolves the trusted workspace and the ambient *sql.Tx; both
// Q26 writes fail closed without either.
func (r *PostgresTaskOutcomeRepository) conditionalTx(ctx context.Context, op string) (string, *sql.Tx, error) {
	idn, ok := identity.FromContext(ctx)
	if !ok || idn == nil || idn.WorkspaceID == "" {
		return "", nil, fmt.Errorf("task_outcome %s: no trusted workspace in context (fail closed)", op)
	}
	exec := r.executor(ctx)
	tx, isTx := exec.(*sql.Tx)
	if !isTx || tx == nil {
		return "", nil, fmt.Errorf("task_outcome %s: requires an ambient transaction (fail closed)", op)
	}
	return idn.WorkspaceID, tx, nil
}

// UpdateTaskOutcomeIfUnchanged is the Q26 conditional cell write: ONE UPDATE
// that only lands when the row still carries the caller's snapshot
// (numeric_value, determination_note, date_modified — each compared with IS
// NOT DISTINCT FROM so NULL == NULL). 0 rows → ErrTaskOutcomeConflict.
//
// fix2-backend (codex impl2 #10): EVERY typed value column the row carries
// (numeric_value, text_value, categorical_value, pass_fail_value) plus
// determination_note is compared against the snapshot. date_modified alone is
// NOT a reliable change detector — it is compared at millisecond grain, so two
// nonnumeric edits inside one millisecond would otherwise both pass. A version
// token (xmin) was considered and rejected: the snapshot comes from the
// generic dbOps.Read (map → protojson), which carries no xmin, and no other
// adapter in this repo uses xmin; full-value comparison needs no read-side
// change. Residual ABA (a concurrent edit that restores the exact same values
// within the same millisecond) is benign: the stored cell equals the snapshot
// the caller decided against.
//
// Only the cell-value columns are written: criteria_type (when specified),
// numeric_value / text_value / categorical_value / pass_fail_value /
// determination_note (each only when present on data), plus date_modified.
// Membership anchors (job_task_id, criteria_version_id), workspace_id and
// ownership are never written. date_modified is compared at millisecond
// grain because the read side (generic dbOps.Read → int64 unix-ms) truncates
// the stored microsecond timestamptz.
func (r *PostgresTaskOutcomeRepository) UpdateTaskOutcomeIfUnchanged(ctx context.Context, data *pb.TaskOutcome, expected *pb.TaskOutcome) (*pb.TaskOutcome, error) {
	if data == nil || data.GetId() == "" {
		return nil, fmt.Errorf("task outcome ID is required")
	}
	if expected == nil {
		return nil, fmt.Errorf("task_outcome conditional update: expected snapshot is required")
	}
	wsID, tx, err := r.conditionalTx(ctx, "conditional update")
	if err != nil {
		return nil, err
	}

	sets := make([]string, 0, 7)
	args := make([]any, 0, 12)
	add := func(col string, v any) {
		args = append(args, v)
		sets = append(sets, col+" = $"+strconv.Itoa(len(args)))
	}
	if ct := data.GetCriteriaType(); ct != enumspb.CriteriaType_CRITERIA_TYPE_UNSPECIFIED {
		add("criteria_type", ct.String())
	}
	if data.NumericValue != nil {
		add("numeric_value", *data.NumericValue)
	}
	if data.TextValue != nil {
		add("text_value", *data.TextValue)
	}
	if data.CategoricalValue != nil {
		add("categorical_value", *data.CategoricalValue)
	}
	if data.PassFailValue != nil {
		add("pass_fail_value", *data.PassFailValue)
	}
	if data.DeterminationNote != nil {
		add("determination_note", *data.DeterminationNote)
	}
	add("date_modified", time.Now().UTC())

	n := len(args)
	args = append(args,
		data.GetId(), // $n+1
		wsID,         // $n+2
		condNullFloat(expected.NumericValue),
		condNullString(expected.DeterminationNote),
		condNullInt64(expected.DateModified),
		condNullString(expected.TextValue),        // $n+6
		condNullString(expected.CategoricalValue), // $n+7
		condNullBool(expected.PassFailValue),      // $n+8
	)
	p := func(i int) string { return "$" + strconv.Itoa(n+i) }
	staffClause, staffArgs := principalscope.StaffScopeClauseAny(ctx, []string{"recorded_by", "reviewed_by"}, n+9)
	args = append(args, staffArgs...)

	query := `UPDATE ` + entityid.TaskOutcome + ` SET ` + strings.Join(sets, ", ") + `
		WHERE id = ` + p(1) + ` AND workspace_id = ` + p(2) + ` AND active = true
		  AND numeric_value IS NOT DISTINCT FROM ` + p(3) + `::double precision
		  AND determination_note IS NOT DISTINCT FROM ` + p(4) + `::text
		  AND text_value IS NOT DISTINCT FROM ` + p(6) + `::text
		  AND categorical_value IS NOT DISTINCT FROM ` + p(7) + `::text
		  AND pass_fail_value IS NOT DISTINCT FROM ` + p(8) + `::boolean
		  AND floor(extract(epoch FROM date_modified) * 1000)::bigint IS NOT DISTINCT FROM ` + p(5) + `::bigint` +
		staffClause + `
		RETURNING ` + taskOutcomeReturningCols

	out, err := scanTaskOutcomeSingleRow(tx.QueryRowContext(ctx, query, args...))
	if err == sql.ErrNoRows {
		return nil, portsdomain.ErrTaskOutcomeConflict
	}
	if err != nil {
		return nil, fmt.Errorf("task_outcome conditional update: %w", err)
	}
	return out, nil
}

// CreateTaskOutcomeIfAbsent is the Q26 locked absence-check create
// (task_outcome has no unique key on (job_task_id, criteria_version_id)):
// inside the ambient transaction it locks the owning job_task row FOR UPDATE
// (workspace-scoped — a foreign or missing task fails closed), then inserts
// through the generic workspace-aware Create only when no ACTIVE outcome
// exists for the pair; otherwise ErrTaskOutcomeConflict. Two concurrent
// creates for one cell serialize on the job_task lock and the second sees
// the first's committed row.
func (r *PostgresTaskOutcomeRepository) CreateTaskOutcomeIfAbsent(ctx context.Context, data *pb.TaskOutcome) (*pb.TaskOutcome, error) {
	if data == nil || data.GetJobTaskId() == "" || data.GetCriteriaVersionId() == "" {
		return nil, fmt.Errorf("task_outcome create-if-absent: job_task_id and criteria_version_id are required")
	}
	wsID, tx, err := r.conditionalTx(ctx, "create-if-absent")
	if err != nil {
		return nil, err
	}

	var lockedID string
	err = tx.QueryRowContext(ctx,
		`SELECT id FROM `+entityid.JobTask+` WHERE id = $1 AND workspace_id = $2 FOR UPDATE`,
		data.GetJobTaskId(), wsID).Scan(&lockedID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task_outcome create-if-absent: job task not found in workspace (fail closed)")
	}
	if err != nil {
		return nil, fmt.Errorf("task_outcome create-if-absent: lock job task: %w", err)
	}

	var exists bool
	if err := tx.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM `+entityid.TaskOutcome+`
		  WHERE job_task_id = $1 AND criteria_version_id = $2 AND active = true)`,
		data.GetJobTaskId(), data.GetCriteriaVersionId()).Scan(&exists); err != nil {
		return nil, fmt.Errorf("task_outcome create-if-absent: absence check: %w", err)
	}
	if exists {
		return nil, portsdomain.ErrTaskOutcomeConflict
	}

	resp, err := r.CreateTaskOutcome(ctx, &pb.CreateTaskOutcomeRequest{Data: data})
	if err != nil {
		return nil, err
	}
	if len(resp.GetData()) == 0 || resp.GetData()[0] == nil {
		return nil, fmt.Errorf("task_outcome create-if-absent: create returned no row")
	}
	return resp.GetData()[0], nil
}

func condNullFloat(v *float64) sql.NullFloat64 {
	if v == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *v, Valid: true}
}

func condNullString(v *string) sql.NullString {
	if v == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *v, Valid: true}
}

func condNullBool(v *bool) sql.NullBool {
	if v == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *v, Valid: true}
}

func condNullInt64(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}
