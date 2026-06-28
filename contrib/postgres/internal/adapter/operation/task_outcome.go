//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.TaskOutcome, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres task_outcome repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresTaskOutcomeRepository(dbOps, tableName), nil
	})
}

// PostgresTaskOutcomeRepository implements task_outcome CRUD operations using PostgreSQL
type PostgresTaskOutcomeRepository struct {
	pb.UnimplementedTaskOutcomeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresTaskOutcomeRepository creates a new PostgreSQL task_outcome repository
func NewPostgresTaskOutcomeRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.TaskOutcomeDomainServiceServer {
	if tableName == "" {
		tableName = "task_outcome"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresTaskOutcomeRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// executor returns the transaction-aware SQL executor: the active *sql.Tx when
// one is present on ctx (so raw-SQL reads participate in the use-case
// transaction and see its uncommitted writes), else the pooled *sql.DB. The
// workspace-aware dbOps exposes GetExecutor(ctx); we type-assert for it so this
// works regardless of the concrete dbOps wrapping. Falls back to the stored
// *sql.DB handle only if neither capability is available.
func (r *PostgresTaskOutcomeRepository) executor(ctx context.Context) sqlexec.DBExecutor {
	if ep, ok := r.dbOps.(interface {
		GetExecutor(ctx context.Context) sqlexec.DBExecutor
	}); ok {
		if e := ep.GetExecutor(ctx); e != nil {
			return e
		}
	}
	if r.db != nil {
		return r.db
	}
	return nil
}

// CreateTaskOutcome creates a new task_outcome record
func (r *PostgresTaskOutcomeRepository) CreateTaskOutcome(ctx context.Context, req *pb.CreateTaskOutcomeRequest) (*pb.CreateTaskOutcomeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("task outcome data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")
	convertMillisToTime(data, "recordedDate")
	convertMillisToTime(data, "reviewedDate")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create task outcome: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	outcome := &pb.TaskOutcome{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, outcome); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateTaskOutcomeResponse{
		Success: true,
		Data:    []*pb.TaskOutcome{outcome},
	}, nil
}

// ReadTaskOutcome retrieves a task_outcome record by ID
func (r *PostgresTaskOutcomeRepository) ReadTaskOutcome(ctx context.Context, req *pb.ReadTaskOutcomeRequest) (*pb.ReadTaskOutcomeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("task outcome ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read task outcome: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	outcome := &pb.TaskOutcome{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, outcome); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	// Staff row-scope (Phase 4): the generic dbOps.Read by id has no WHERE seam,
	// so guard post-read — a STAFF principal may only read an outcome it
	// recorded_by OR reviewed_by. Fail-closed → not-found on an empty session
	// staff.id or a row it neither recorded nor reviewed. Non-staff unaffected.
	if staffID, ok := staffRowScope(ctx); ok {
		owns := staffID != "" && (outcome.RecordedBy == staffID ||
			(outcome.ReviewedBy != nil && *outcome.ReviewedBy == staffID))
		if !owns {
			return nil, fmt.Errorf("task outcome with ID '%s' not found", req.Data.Id)
		}
	}

	return &pb.ReadTaskOutcomeResponse{
		Success: true,
		Data:    []*pb.TaskOutcome{outcome},
	}, nil
}

// UpdateTaskOutcome updates a task_outcome record
func (r *PostgresTaskOutcomeRepository) UpdateTaskOutcome(ctx context.Context, req *pb.UpdateTaskOutcomeRequest) (*pb.UpdateTaskOutcomeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("task outcome ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")
	convertMillisToTime(data, "recordedDate")
	convertMillisToTime(data, "reviewedDate")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update task outcome: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	outcome := &pb.TaskOutcome{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, outcome); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateTaskOutcomeResponse{
		Success: true,
		Data:    []*pb.TaskOutcome{outcome},
	}, nil
}

// DeleteTaskOutcome deletes a task_outcome record (soft delete)
func (r *PostgresTaskOutcomeRepository) DeleteTaskOutcome(ctx context.Context, req *pb.DeleteTaskOutcomeRequest) (*pb.DeleteTaskOutcomeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("task outcome ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete task outcome: %w", err)
	}

	return &pb.DeleteTaskOutcomeResponse{
		Success: true,
	}, nil
}

// ListTaskOutcomes lists task_outcome records with optional filters
func (r *PostgresTaskOutcomeRepository) ListTaskOutcomes(ctx context.Context, req *pb.ListTaskOutcomesRequest) (*pb.ListTaskOutcomesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list task outcomes: %w", err)
	}

	var outcomes []*pb.TaskOutcome
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal task_outcome row: %v", err)
			continue
		}

		outcome := &pb.TaskOutcome{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, outcome); err != nil {
			log.Printf("WARN: protojson unmarshal task_outcome: %v", err)
			continue
		}
		outcomes = append(outcomes, outcome)
	}

	return &pb.ListTaskOutcomesResponse{
		Success: true,
		Data:    outcomes,
	}, nil
}

var taskOutcomeSortableSQLCols = []string{
	"id", "job_task_id", "criteria_version_id", "criteria_type", "is_ad_hoc",
	"numeric_value", "text_value", "categorical_value", "pass_fail_value",
	"determination", "determination_source", "determination_note",
	"auto_proposed_determination", "recorded_by", "recorded_date",
	"reviewed_by", "reviewed_date", "attachment_ids", "revision_of_id",
	"revision_number", "active", "date_created", "date_modified",
}

// GetTaskOutcomeListPageData retrieves task outcomes with pagination, filtering, sorting, and search
func (r *PostgresTaskOutcomeRepository) GetTaskOutcomeListPageData(
	ctx context.Context,
	req *pb.GetTaskOutcomeListPageDataRequest,
) (*pb.GetTaskOutcomeListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get task outcome list page data request is required")
	}

	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
				offset = (page - 1) * limit
			}
		}
	}

	// Sort — fail-closed against the per-entity whitelist (A2 guard). The outer
	// SELECT projects the enriched columns unprefixed (e.*), so the ORDER BY
	// references unprefixed whitelist columns. An unknown column errors instead
	// of being interpolated verbatim into ORDER BY.
	orderByClause, err := postgresCore.BuildOrderBy(taskOutcomeSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}

	toColumns := `
		to_.id, to_.job_task_id, to_.criteria_version_id, to_.criteria_type,
		to_.is_ad_hoc, to_.numeric_value, to_.text_value, to_.categorical_value,
		to_.pass_fail_value, to_.determination, to_.determination_source,
		to_.determination_note, to_.auto_proposed_determination,
		to_.recorded_by, to_.recorded_date, to_.reviewed_by, to_.reviewed_date,
		to_.attachment_ids, to_.revision_of_id, to_.revision_number,
		to_.active, to_.date_created, to_.date_modified
	`

	// Staff row-scope (Phase 4): a STAFF principal sees only outcomes it
	// recorded_by OR reviewed_by ($4, session-derived). Predicate lives inside
	// the enriched CTE so the counted total matches the scoped set. Non-staff →
	// empty clause (unchanged).
	staffClause, staffArgs := staffScopeClauseAny(ctx, []string{"to_.recorded_by", "to_.reviewed_by"}, 4)

	query := `
		WITH enriched AS (
			SELECT ` + toColumns + `
			FROM task_outcome to_
			WHERE to_.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       to_.determination_note ILIKE $1)` + staffClause + `
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*, c.total
		FROM enriched e, counted c
		` + orderByClause + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, append([]any{searchPattern, limit, offset}, staffArgs...)...)
	if err != nil {
		return nil, fmt.Errorf("failed to query task outcome list page data: %w", err)
	}
	defer rows.Close()

	var outcomes []*pb.TaskOutcome
	var totalCount int64

	for rows.Next() {
		outcome, cnt, err := scanTaskOutcomeRowWithTotal(rows)
		if err != nil {
			return nil, err
		}
		totalCount = cnt
		outcomes = append(outcomes, outcome)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task outcome rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetTaskOutcomeListPageDataResponse{
		TaskOutcomeList: outcomes,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
		Success: true,
	}, nil
}

// GetTaskOutcomeItemPageData retrieves a single task outcome with enriched data
func (r *PostgresTaskOutcomeRepository) GetTaskOutcomeItemPageData(
	ctx context.Context,
	req *pb.GetTaskOutcomeItemPageDataRequest,
) (*pb.GetTaskOutcomeItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get task outcome item page data request is required")
	}
	if req.TaskOutcomeId == "" {
		return nil, fmt.Errorf("task outcome ID is required")
	}

	// Staff row-scope (Phase 4): a STAFF principal may only read an outcome it
	// recorded_by OR reviewed_by ($2). Fail-closed → not-found otherwise.
	staffClause, staffArgs := staffScopeClauseAny(ctx, []string{"to_.recorded_by", "to_.reviewed_by"}, 2)

	query := `
		SELECT
			to_.id, to_.job_task_id, to_.criteria_version_id, to_.criteria_type,
			to_.is_ad_hoc, to_.numeric_value, to_.text_value, to_.categorical_value,
			to_.pass_fail_value, to_.determination, to_.determination_source,
			to_.determination_note, to_.auto_proposed_determination,
			to_.recorded_by, to_.recorded_date, to_.reviewed_by, to_.reviewed_date,
			to_.attachment_ids, to_.revision_of_id, to_.revision_number,
			to_.active, to_.date_created, to_.date_modified
		FROM task_outcome to_
		WHERE to_.id = $1 AND to_.active = true` + staffClause + `
	`

	row := r.db.QueryRowContext(ctx, query, append([]any{req.TaskOutcomeId}, staffArgs...)...)

	outcome, err := scanTaskOutcomeSingleRow(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task outcome with ID '%s' not found", req.TaskOutcomeId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query task outcome item page data: %w", err)
	}

	return &pb.GetTaskOutcomeItemPageDataResponse{
		TaskOutcome: outcome,
		Success:     true,
	}, nil
}

// ListByJobTask retrieves all task outcomes for a given job task
func (r *PostgresTaskOutcomeRepository) ListByJobTask(
	ctx context.Context,
	req *pb.ListTaskOutcomesByJobTaskRequest,
) (*pb.ListTaskOutcomesByJobTaskResponse, error) {
	if req == nil || req.JobTaskId == "" {
		return nil, fmt.Errorf("job task ID is required")
	}

	// Staff row-scope (Phase 4): within a task a STAFF principal sees only the
	// outcomes it recorded_by OR reviewed_by ($2). Non-staff → empty clause.
	staffClause, staffArgs := staffScopeClauseAny(ctx, []string{"to_.recorded_by", "to_.reviewed_by"}, 2)

	query := `
		SELECT
			to_.id, to_.job_task_id, to_.criteria_version_id, to_.criteria_type,
			to_.is_ad_hoc, to_.numeric_value, to_.text_value, to_.categorical_value,
			to_.pass_fail_value, to_.determination, to_.determination_source,
			to_.determination_note, to_.auto_proposed_determination,
			to_.recorded_by, to_.recorded_date, to_.reviewed_by, to_.reviewed_date,
			to_.attachment_ids, to_.revision_of_id, to_.revision_number,
			to_.active, to_.date_created, to_.date_modified
		FROM task_outcome to_
		WHERE to_.job_task_id = $1 AND to_.active = true` + staffClause + `
		ORDER BY to_.date_created DESC
	`

	rows, err := r.db.QueryContext(ctx, query, append([]any{req.JobTaskId}, staffArgs...)...)
	if err != nil {
		return nil, fmt.Errorf("failed to list task outcomes by job task: %w", err)
	}
	defer rows.Close()

	outcomes, err := scanTaskOutcomeRows(rows)
	if err != nil {
		return nil, err
	}

	return &pb.ListTaskOutcomesByJobTaskResponse{
		TaskOutcomes: outcomes,
		Success:      true,
	}, nil
}

// ListByJobPhase retrieves all task outcomes for a given job phase via JOIN with job_task
func (r *PostgresTaskOutcomeRepository) ListByJobPhase(
	ctx context.Context,
	req *pb.ListTaskOutcomesByJobPhaseRequest,
) (*pb.ListTaskOutcomesByJobPhaseResponse, error) {
	if req == nil || req.JobPhaseId == "" {
		return nil, fmt.Errorf("job phase ID is required")
	}

	// Staff row-scope (Phase 4): within a phase a STAFF principal sees only the
	// outcomes it recorded_by OR reviewed_by ($2). Non-staff → empty clause.
	staffClause, staffArgs := staffScopeClauseAny(ctx, []string{"to_.recorded_by", "to_.reviewed_by"}, 2)

	query := `
		SELECT
			to_.id, to_.job_task_id, to_.criteria_version_id, to_.criteria_type,
			to_.is_ad_hoc, to_.numeric_value, to_.text_value, to_.categorical_value,
			to_.pass_fail_value, to_.determination, to_.determination_source,
			to_.determination_note, to_.auto_proposed_determination,
			to_.recorded_by, to_.recorded_date, to_.reviewed_by, to_.reviewed_date,
			to_.attachment_ids, to_.revision_of_id, to_.revision_number,
			to_.active, to_.date_created, to_.date_modified
		FROM task_outcome to_
		JOIN job_task jt ON to_.job_task_id = jt.id
		WHERE jt.job_phase_id = $1 AND to_.active = true` + staffClause + `
		ORDER BY to_.date_created DESC
	`

	// Use the transaction-aware executor so this JOIN read participates in an
	// active *sql.Tx (and never nil-derefs a missing raw *sql.DB handle). The
	// workspace-aware dbOps exposes GetExecutor(ctx); fall back to the stored
	// pool only if that capability is somehow absent.
	exec := r.executor(ctx)
	if exec == nil {
		return nil, fmt.Errorf("task_outcome ListByJobPhase: no SQL executor available")
	}
	rows, err := exec.QueryContext(ctx, query, append([]any{req.JobPhaseId}, staffArgs...)...)
	if err != nil {
		return nil, fmt.Errorf("failed to list task outcomes by job phase: %w", err)
	}
	defer rows.Close()

	outcomes, err := scanTaskOutcomeRows(rows)
	if err != nil {
		return nil, err
	}

	return &pb.ListTaskOutcomesByJobPhaseResponse{
		TaskOutcomes: outcomes,
		Success:      true,
	}, nil
}

// ListByJob retrieves all task outcomes for a given job via multi-JOIN with job_task
func (r *PostgresTaskOutcomeRepository) ListByJob(
	ctx context.Context,
	req *pb.ListTaskOutcomesByJobRequest,
) (*pb.ListTaskOutcomesByJobResponse, error) {
	if req == nil || req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	// Staff row-scope (Phase 4): within a job a STAFF principal sees only the
	// outcomes it recorded_by OR reviewed_by ($2). Non-staff → empty clause.
	staffClause, staffArgs := staffScopeClauseAny(ctx, []string{"to_.recorded_by", "to_.reviewed_by"}, 2)

	query := `
		SELECT
			to_.id, to_.job_task_id, to_.criteria_version_id, to_.criteria_type,
			to_.is_ad_hoc, to_.numeric_value, to_.text_value, to_.categorical_value,
			to_.pass_fail_value, to_.determination, to_.determination_source,
			to_.determination_note, to_.auto_proposed_determination,
			to_.recorded_by, to_.recorded_date, to_.reviewed_by, to_.reviewed_date,
			to_.attachment_ids, to_.revision_of_id, to_.revision_number,
			to_.active, to_.date_created, to_.date_modified
		FROM task_outcome to_
		JOIN job_task jt ON to_.job_task_id = jt.id
		WHERE jt.job_id = $1 AND to_.active = true` + staffClause + `
		ORDER BY to_.date_created DESC
	`

	rows, err := r.db.QueryContext(ctx, query, append([]any{req.JobId}, staffArgs...)...)
	if err != nil {
		return nil, fmt.Errorf("failed to list task outcomes by job: %w", err)
	}
	defer rows.Close()

	outcomes, err := scanTaskOutcomeRows(rows)
	if err != nil {
		return nil, err
	}

	return &pb.ListTaskOutcomesByJobResponse{
		TaskOutcomes: outcomes,
		Success:      true,
	}, nil
}

// scanTOFields is a shared scanner for all task_outcome SELECT columns
func scanTOFields(scanFn func(dest ...any) error) (
	id string, jobTaskID string, criteriaVersionID string,
	criteriaType string, isAdHoc bool,
	numericValue sql.NullFloat64, textValue sql.NullString,
	categoricalValue sql.NullString, passFailValue sql.NullBool,
	determination string, determinationSource string,
	determinationNote sql.NullString, autoProposedDetermination sql.NullString,
	recordedBy string, recordedDate sql.NullInt64,
	reviewedBy sql.NullString, reviewedDate sql.NullInt64,
	attachmentIdsStr string, revisionOfId sql.NullString,
	revisionNumber int32, active bool,
	dateCreated sql.NullInt64, dateModified sql.NullInt64, err error,
) {
	// These four columns are NULL-able in the live schema (a recorded outcome
	// may carry no determination yet, no determination_source, no recorded_by
	// attribution, and no attachment_ids). Scan them as NullString and coalesce
	// to the empty-string zero value the downstream builder already expects
	// (empty determination → enum UNSPECIFIED; empty attachment_ids → no ids).
	// Without this, a synthesized/back-filled task_outcome with NULL
	// determination fails scanning ("converting NULL to string is unsupported").
	var determinationNS, determinationSourceNS, recordedByNS, attachmentIdsNS sql.NullString
	// Date columns are scanned through epochMillis so a timestamptz live column
	// (e.g. a SQL-typed backfill) is accepted alongside the BIGINT unix-ms shape.
	var recordedDateEM, reviewedDateEM, dateCreatedEM, dateModifiedEM epochMillis
	err = scanFn(
		&id, &jobTaskID, &criteriaVersionID,
		&criteriaType, &isAdHoc,
		&numericValue, &textValue,
		&categoricalValue, &passFailValue,
		&determinationNS, &determinationSourceNS,
		&determinationNote, &autoProposedDetermination,
		&recordedByNS, &recordedDateEM,
		&reviewedBy, &reviewedDateEM,
		&attachmentIdsNS, &revisionOfId,
		&revisionNumber, &active,
		&dateCreatedEM, &dateModifiedEM,
	)
	determination = determinationNS.String
	determinationSource = determinationSourceNS.String
	recordedBy = recordedByNS.String
	attachmentIdsStr = attachmentIdsNS.String
	recordedDate = recordedDateEM.asNullInt64()
	reviewedDate = reviewedDateEM.asNullInt64()
	dateCreated = dateCreatedEM.asNullInt64()
	dateModified = dateModifiedEM.asNullInt64()
	return
}

// epochMillis scans a date column that may be stored EITHER as BIGINT unix-ms
// (the espyna timestamp convention buildTaskOutcome expects) OR as a
// timestamp/timestamptz (the shape a SQL-typed backfill/migration may write).
// It normalises both to int64 unix-ms, mirroring the generic operations
// reader's TIMESTAMP→ms normalisation, so a row whose date columns are
// timestamptz no longer fails scanning ("converting time.Time to int64").
type epochMillis struct {
	Valid bool
	Ms    int64
}

func (e *epochMillis) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		e.Valid = false
		e.Ms = 0
	case time.Time:
		e.Valid = true
		e.Ms = v.UnixMilli()
	case int64:
		e.Valid = true
		e.Ms = v
	case int:
		e.Valid = true
		e.Ms = int64(v)
	case []byte:
		var n int64
		if _, err := fmt.Sscan(string(v), &n); err != nil {
			return fmt.Errorf("epochMillis: cannot parse %q: %w", string(v), err)
		}
		e.Valid = true
		e.Ms = n
	default:
		return fmt.Errorf("epochMillis: unsupported source type %T", src)
	}
	return nil
}

func (e epochMillis) asNullInt64() sql.NullInt64 {
	return sql.NullInt64{Int64: e.Ms, Valid: e.Valid}
}

// scanTaskOutcomeRows scans multiple rows into TaskOutcome protos
func scanTaskOutcomeRows(rows *sql.Rows) ([]*pb.TaskOutcome, error) {
	var outcomes []*pb.TaskOutcome
	for rows.Next() {
		id, jobTaskID, criteriaVersionID, criteriaType, isAdHoc,
			numericValue, textValue, categoricalValue, passFailValue,
			determination, determinationSource, determinationNote, autoProposedDetermination,
			recordedBy, recordedDate, reviewedBy, reviewedDate,
			attachmentIdsStr, revisionOfId, revisionNumber, active,
			dateCreated, dateModified, err := scanTOFields(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task outcome row: %w", err)
		}
		outcomes = append(outcomes, buildTaskOutcome(
			id, jobTaskID, criteriaVersionID, criteriaType, isAdHoc,
			numericValue, textValue, categoricalValue, passFailValue,
			determination, determinationSource, determinationNote, autoProposedDetermination,
			recordedBy, recordedDate, reviewedBy, reviewedDate,
			attachmentIdsStr, revisionOfId, revisionNumber, active,
			dateCreated, dateModified))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task outcome rows: %w", err)
	}
	return outcomes, nil
}

// scanTaskOutcomeRowWithTotal scans a row with a total count column appended
func scanTaskOutcomeRowWithTotal(rows *sql.Rows) (*pb.TaskOutcome, int64, error) {
	var (
		id                        string
		jobTaskID                 string
		criteriaVersionID         string
		criteriaType              string
		isAdHoc                   bool
		numericValue              sql.NullFloat64
		textValue                 sql.NullString
		categoricalValue          sql.NullString
		passFailValue             sql.NullBool
		determination             string
		determinationSource       string
		determinationNote         sql.NullString
		autoProposedDetermination sql.NullString
		recordedBy                string
		recordedDate              sql.NullInt64
		reviewedBy                sql.NullString
		reviewedDate              sql.NullInt64
		attachmentIdsStr          string
		revisionOfId              sql.NullString
		revisionNumber            int32
		active                    bool
		dateCreated               sql.NullInt64
		dateModified              sql.NullInt64
		total                     int64
	)

	err := rows.Scan(
		&id, &jobTaskID, &criteriaVersionID,
		&criteriaType, &isAdHoc,
		&numericValue, &textValue,
		&categoricalValue, &passFailValue,
		&determination, &determinationSource,
		&determinationNote, &autoProposedDetermination,
		&recordedBy, &recordedDate,
		&reviewedBy, &reviewedDate,
		&attachmentIdsStr, &revisionOfId,
		&revisionNumber, &active,
		&dateCreated, &dateModified, &total,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to scan task outcome row: %w", err)
	}

	outcome := buildTaskOutcome(
		id, jobTaskID, criteriaVersionID, criteriaType, isAdHoc,
		numericValue, textValue, categoricalValue, passFailValue,
		determination, determinationSource, determinationNote, autoProposedDetermination,
		recordedBy, recordedDate, reviewedBy, reviewedDate,
		attachmentIdsStr, revisionOfId, revisionNumber, active,
		dateCreated, dateModified)
	return outcome, total, nil
}

// scanTaskOutcomeSingleRow scans a single sql.Row into a TaskOutcome proto
func scanTaskOutcomeSingleRow(row *sql.Row) (*pb.TaskOutcome, error) {
	id, jobTaskID, criteriaVersionID, criteriaType, isAdHoc,
		numericValue, textValue, categoricalValue, passFailValue,
		determination, determinationSource, determinationNote, autoProposedDetermination,
		recordedBy, recordedDate, reviewedBy, reviewedDate,
		attachmentIdsStr, revisionOfId, revisionNumber, active,
		dateCreated, dateModified, err := scanTOFields(row.Scan)
	if err != nil {
		return nil, err
	}
	return buildTaskOutcome(
		id, jobTaskID, criteriaVersionID, criteriaType, isAdHoc,
		numericValue, textValue, categoricalValue, passFailValue,
		determination, determinationSource, determinationNote, autoProposedDetermination,
		recordedBy, recordedDate, reviewedBy, reviewedDate,
		attachmentIdsStr, revisionOfId, revisionNumber, active,
		dateCreated, dateModified), nil
}

func buildTaskOutcome(
	id string, jobTaskID string, criteriaVersionID string,
	criteriaType string, isAdHoc bool,
	numericValue sql.NullFloat64, textValue sql.NullString,
	categoricalValue sql.NullString, passFailValue sql.NullBool,
	determination string, determinationSource string,
	determinationNote sql.NullString, autoProposedDetermination sql.NullString,
	recordedBy string, recordedDate sql.NullInt64,
	reviewedBy sql.NullString, reviewedDate sql.NullInt64,
	attachmentIdsStr string, revisionOfId sql.NullString,
	revisionNumber int32, active bool,
	dateCreated sql.NullInt64, dateModified sql.NullInt64,
) *pb.TaskOutcome {
	outcome := &pb.TaskOutcome{
		Id:                  id,
		Active:              active,
		JobTaskId:           jobTaskID,
		CriteriaVersionId:   criteriaVersionID,
		CriteriaType:        enumspb.CriteriaType(enumspb.CriteriaType_value[criteriaType]),
		IsAdHoc:             isAdHoc,
		Determination:       enumspb.Determination(enumspb.Determination_value[determination]),
		DeterminationSource: enumspb.DeterminationSource(enumspb.DeterminationSource_value[determinationSource]),
		RecordedBy:          recordedBy,
		RevisionNumber:      revisionNumber,
	}

	if numericValue.Valid {
		outcome.NumericValue = &numericValue.Float64
	}
	if textValue.Valid {
		outcome.TextValue = &textValue.String
	}
	if categoricalValue.Valid {
		outcome.CategoricalValue = &categoricalValue.String
	}
	if passFailValue.Valid {
		outcome.PassFailValue = &passFailValue.Bool
	}
	if determinationNote.Valid {
		outcome.DeterminationNote = &determinationNote.String
	}
	if autoProposedDetermination.Valid {
		v := enumspb.Determination(enumspb.Determination_value[autoProposedDetermination.String])
		outcome.AutoProposedDetermination = &v
	}
	if recordedDate.Valid {
		outcome.RecordedDate = &recordedDate.Int64
	}
	if reviewedBy.Valid {
		outcome.ReviewedBy = &reviewedBy.String
	}
	if reviewedDate.Valid {
		outcome.ReviewedDate = &reviewedDate.Int64
	}
	if attachmentIdsStr != "" {
		var ids []string
		if err := json.Unmarshal([]byte(attachmentIdsStr), &ids); err == nil {
			outcome.AttachmentIds = ids
		}
	}
	if revisionOfId.Valid {
		outcome.RevisionOfId = &revisionOfId.String
	}
	if dateCreated.Valid {
		outcome.DateCreated = &dateCreated.Int64
		dcStr := time.UnixMilli(dateCreated.Int64).Format(time.RFC3339)
		outcome.DateCreatedString = &dcStr
	}
	if dateModified.Valid {
		outcome.DateModified = &dateModified.Int64
		dmStr := time.UnixMilli(dateModified.Int64).Format(time.RFC3339)
		outcome.DateModifiedString = &dmStr
	}

	return outcome
}
