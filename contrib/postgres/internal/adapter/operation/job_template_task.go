//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
)

// job_template_task has NO workspace_id column of its own, so the generic
// workspace-aware decorator cannot scope it and the raw-SQL paths below would
// otherwise return or mutate any tenant's rows. Every read/write is confined to
// the caller's workspace by deriving ownership up the parent chain
// task -> job_template_phase -> job_template (jt.workspace_id). The predicate
// spelling ($N::text = '' OR jt.workspace_id = $N::text) mirrors the sibling
// job/job_template adapters: a bound workspace scopes the rows; an empty
// workspace (a service context with no tenant bound) is left unscoped.

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.JobTemplateTask, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres job_template_task repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresJobTemplateTaskRepository(dbOps, tableName), nil
	})
}

// PostgresJobTemplateTaskRepository implements job_template_task CRUD operations using PostgreSQL
type PostgresJobTemplateTaskRepository struct {
	pb.UnimplementedJobTemplateTaskDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresJobTemplateTaskRepository creates a new PostgreSQL job_template_task repository
func NewPostgresJobTemplateTaskRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobTemplateTaskDomainServiceServer {
	if tableName == "" {
		tableName = "job_template_task"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresJobTemplateTaskRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateJobTemplateTask creates a new job template task record
func (r *PostgresJobTemplateTaskRepository) CreateJobTemplateTask(ctx context.Context, req *pb.CreateJobTemplateTaskRequest) (*pb.CreateJobTemplateTaskResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("job template task data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Convert millis timestamps to time.Time for postgres timestamp columns
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	// Cross-tenant guard: the parent job_template_phase must resolve to a
	// job_template owned by the caller's workspace, otherwise a task could be
	// attached under another tenant's phase.
	if err := r.ensurePhaseInWorkspace(ctx, req.Data.GetJobTemplatePhaseId()); err != nil {
		return nil, err
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create job template task: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	task := &pb.JobTemplateTask{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateJobTemplateTaskResponse{
		Success: true,
		Data:    []*pb.JobTemplateTask{task},
	}, nil
}

// ReadJobTemplateTask retrieves a job template task record by ID
func (r *PostgresJobTemplateTaskRepository) ReadJobTemplateTask(ctx context.Context, req *pb.ReadJobTemplateTaskRequest) (*pb.ReadJobTemplateTaskResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job template task ID is required")
	}

	// Cross-tenant guard: reject a by-id read of a task owned by another tenant.
	if err := r.ensureTaskInWorkspace(ctx, req.Data.Id); err != nil {
		return nil, err
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job template task: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	task := &pb.JobTemplateTask{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadJobTemplateTaskResponse{
		Success: true,
		Data:    []*pb.JobTemplateTask{task},
	}, nil
}

// UpdateJobTemplateTask updates a job template task record
func (r *PostgresJobTemplateTaskRepository) UpdateJobTemplateTask(ctx context.Context, req *pb.UpdateJobTemplateTaskRequest) (*pb.UpdateJobTemplateTaskResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job template task ID is required")
	}

	// Cross-tenant guard: reject an update to a task owned by another tenant.
	if err := r.ensureTaskInWorkspace(ctx, req.Data.Id); err != nil {
		return nil, err
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Convert millis timestamps to time.Time for postgres timestamp columns
	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update job template task: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	task := &pb.JobTemplateTask{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateJobTemplateTaskResponse{
		Success: true,
		Data:    []*pb.JobTemplateTask{task},
	}, nil
}

// DeleteJobTemplateTask deletes a job template task record (soft delete)
func (r *PostgresJobTemplateTaskRepository) DeleteJobTemplateTask(ctx context.Context, req *pb.DeleteJobTemplateTaskRequest) (*pb.DeleteJobTemplateTaskResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job template task ID is required")
	}

	// Cross-tenant guard: reject a delete of a task owned by another tenant.
	if err := r.ensureTaskInWorkspace(ctx, req.Data.Id); err != nil {
		return nil, err
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete job template task: %w", err)
	}

	return &pb.DeleteJobTemplateTaskResponse{
		Success: true,
	}, nil
}

// ListJobTemplateTasks lists job template task records with optional filters
func (r *PostgresJobTemplateTaskRepository) ListJobTemplateTasks(ctx context.Context, req *pb.ListJobTemplateTasksRequest) (*pb.ListJobTemplateTasksResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list job template tasks: %w", err)
	}

	var tasks []*pb.JobTemplateTask
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal job_template_task row: %v", err)
			continue
		}

		task := &pb.JobTemplateTask{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, task); err != nil {
			log.Printf("WARN: protojson unmarshal job_template_task: %v", err)
			continue
		}
		tasks = append(tasks, task)
	}

	// Cross-tenant scope: the generic list flows through the workspace-aware
	// decorator, which cannot scope this column-less child table, so it returns
	// rows across every tenant. Confine them to the caller's workspace via the
	// parent chain.
	tasks, err = r.filterTasksByWorkspace(ctx, tasks)
	if err != nil {
		return nil, err
	}

	return &pb.ListJobTemplateTasksResponse{
		Success: true,
		Data:    tasks,
	}, nil
}

// jobTemplateTaskSortableSQLCols is the fail-closed sort whitelist for
// GetJobTemplateTaskListPageData. Only columns projected by the CTE SELECT are
// included so ORDER BY can never reference an unprojected/injected identifier.
var jobTemplateTaskSortableSQLCols = []string{
	"id", "date_created", "date_modified", "active", "job_template_phase_id",
	"name", "step_order", "estimated_duration_minutes",
}

// GetJobTemplateTaskListPageData retrieves tasks with pagination, filtering, sorting, and search
func (r *PostgresJobTemplateTaskRepository) GetJobTemplateTaskListPageData(
	ctx context.Context,
	req *pb.GetJobTemplateTaskListPageDataRequest,
) (*pb.GetJobTemplateTaskListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job template task list page data request is required")
	}

	searchPattern, err := postgresCore.BoundedContainsSearchPattern(req.GetSearch())
	if err != nil {
		return nil, fmt.Errorf("invalid list search: %w", err)
	}

	limit, offset, page, err := postgresCore.BoundedOffsetPagination(req.GetPagination(), 50)
	if err != nil {
		return nil, fmt.Errorf("invalid list pagination: %w", err)
	}

	// Sort — fail-closed against the per-entity whitelist (A2 guard). The default
	// references the outer enriched projection (step_order) since the page rows
	// are selected via "SELECT e.* FROM enriched e". An unknown sort column now
	// errors instead of being interpolated verbatim into ORDER BY.
	orderByClause, err := postgresCore.BuildOrderBy(jobTemplateTaskSortableSQLCols, req.GetSort(), "step_order ASC")
	if err != nil {
		return nil, err
	}

	// Cross-tenant scope: the drawer list is reached only through the session, so
	// a missing identity is a middleware fault and fails closed (Must). $4 carries
	// the workspace_id, derived up the parent chain.
	wsID := identity.Must(ctx).WorkspaceID
	query := jobTemplateTaskListPageDataSQL(orderByClause)

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset, wsID)
	if err != nil {
		return nil, fmt.Errorf("failed to query job template task list page data: %w", err)
	}
	defer rows.Close()

	var tasks []*pb.JobTemplateTask
	var totalCount int64

	for rows.Next() {
		var (
			id                       string
			dateCreated              time.Time
			dateModified             time.Time
			active                   bool
			jobTemplatePhaseID       string
			name                     string
			stepOrder                sql.NullInt32
			estimatedDurationMinutes *int32
			code                     sql.NullString
			total                    int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&jobTemplatePhaseID,
			&name,
			&stepOrder,
			&estimatedDurationMinutes,
			&code,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job template task row: %w", err)
		}

		totalCount = total

		task := &pb.JobTemplateTask{
			Id:                 id,
			Active:             active,
			JobTemplatePhaseId: jobTemplatePhaseID,
			Name:               name,
			StepOrder:          stepOrder.Int32,
		}

		if estimatedDurationMinutes != nil {
			task.EstimatedDurationMinutes = estimatedDurationMinutes
		}
		if code.Valid {
			v := code.String
			task.Code = &v
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			task.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			task.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			task.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			task.DateModifiedString = &dmStr
		}

		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job template task rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetJobTemplateTaskListPageDataResponse{
		JobTemplateTaskList: tasks,
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

// GetJobTemplateTaskItemPageData retrieves a single task with enriched data
func (r *PostgresJobTemplateTaskRepository) GetJobTemplateTaskItemPageData(
	ctx context.Context,
	req *pb.GetJobTemplateTaskItemPageDataRequest,
) (*pb.GetJobTemplateTaskItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job template task item page data request is required")
	}
	if req.JobTemplateTaskId == "" {
		return nil, fmt.Errorf("job template task ID is required")
	}

	// Cross-tenant scope: drawer item reached only through the session; a missing
	// identity fails closed (Must). $2 carries the workspace_id, derived up the
	// parent chain.
	wsID := identity.Must(ctx).WorkspaceID
	query := jobTemplateTaskItemPageDataSQL()

	row := r.db.QueryRowContext(ctx, query, req.JobTemplateTaskId, wsID)

	var (
		id                       string
		dateCreated              time.Time
		dateModified             time.Time
		active                   bool
		jobTemplatePhaseID       string
		name                     string
		stepOrder                sql.NullInt32
		estimatedDurationMinutes *int32
		code                     sql.NullString
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&jobTemplatePhaseID,
		&name,
		&stepOrder,
		&estimatedDurationMinutes,
		&code,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job template task with ID '%s' not found", req.JobTemplateTaskId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query job template task item page data: %w", err)
	}

	task := &pb.JobTemplateTask{
		Id:                 id,
		Active:             active,
		JobTemplatePhaseId: jobTemplatePhaseID,
		Name:               name,
		StepOrder:          stepOrder.Int32,
	}

	if estimatedDurationMinutes != nil {
		task.EstimatedDurationMinutes = estimatedDurationMinutes
	}
	if code.Valid {
		v := code.String
		task.Code = &v
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		task.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		task.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		task.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		task.DateModifiedString = &dmStr
	}

	return &pb.GetJobTemplateTaskItemPageDataResponse{
		JobTemplateTask: task,
		Success:         true,
	}, nil
}

// jobTemplateTaskListByPhaseSQL returns the SELECT that ListByPhase runs. It is
// a package-level builder (not an inline literal) purely so the SQL projection
// is unit-testable via a shape test — mirroring jobTemplatePhaseListByTemplateSQL()
// in this package. Every column named here MUST have a matching positional
// rows.Scan(...) destination in ListByPhase, in the SAME order: a symmetric
// column drop returns nil forever without a database/sql column-count error.
func jobTemplateTaskListByPhaseSQL() string {
	return `
		SELECT
			jtt.id,
			jtt.date_created,
			jtt.date_modified,
			jtt.active,
			jtt.job_template_phase_id,
			jtt.name,
			jtt.step_order,
			jtt.estimated_duration_minutes,
			jtt.code
		FROM ` + entityid.JobTemplateTask + ` jtt
		JOIN ` + entityid.JobTemplatePhase + ` jtp ON jtp.id = jtt.job_template_phase_id
		JOIN ` + entityid.JobTemplate + ` jt ON jt.id = jtp.job_template_id
		WHERE jtt.job_template_phase_id = $1 AND jtt.active = true
		  AND ($2::text = '' OR jt.workspace_id = $2::text)
		ORDER BY jtt.step_order ASC
	`
}

// jobTemplateTaskItemPageDataSQL returns the enriched single-task read, scoped to
// the caller's workspace up the parent chain ($2 = workspace_id).
func jobTemplateTaskItemPageDataSQL() string {
	return `
		SELECT
			jtt.id,
			jtt.date_created,
			jtt.date_modified,
			jtt.active,
			jtt.job_template_phase_id,
			jtt.name,
			jtt.step_order,
			jtt.estimated_duration_minutes,
			jtt.code
		FROM ` + entityid.JobTemplateTask + ` jtt
		JOIN ` + entityid.JobTemplatePhase + ` jtp ON jtp.id = jtt.job_template_phase_id
		JOIN ` + entityid.JobTemplate + ` jt ON jt.id = jtp.job_template_id
		WHERE jtt.id = $1 AND jtt.active = true
		  AND ($2::text = '' OR jt.workspace_id = $2::text)
	`
}

// jobTemplateTaskListPageDataSQL returns the paginated task list, scoped to the
// caller's workspace up the parent chain ($4 = workspace_id).
func jobTemplateTaskListPageDataSQL(orderByClause string) string {
	return `
		WITH enriched AS (
			SELECT
				jtt.id,
				jtt.date_created,
				jtt.date_modified,
				jtt.active,
				jtt.job_template_phase_id,
				jtt.name,
				jtt.step_order,
				jtt.estimated_duration_minutes,
				jtt.code
			FROM ` + entityid.JobTemplateTask + ` jtt
			JOIN ` + entityid.JobTemplatePhase + ` jtp ON jtp.id = jtt.job_template_phase_id
			JOIN ` + entityid.JobTemplate + ` jt ON jt.id = jtp.job_template_id
			WHERE jtt.active = true
			  AND ($4::text = '' OR jt.workspace_id = $4::text)
			  AND ($1::text IS NULL OR $1::text = '' OR
			       jtt.name ILIKE $1)
		)
		-- A3 (Q-PAGE-COUNT default tier): COUNT(*) OVER () computes the total in the
		-- same scan as the page rows (the prior counted CTE forced a second scan).
		SELECT
			e.*,
			COUNT(*) OVER () AS total
		FROM enriched e
		` + orderByClause + `
		LIMIT $2 OFFSET $3;
	`
}

// jobTemplateTaskWorkspaceOwnedSQL reports whether a task (by id, $1) resolves up
// the parent chain to a job_template owned by the workspace ($2). Backs the
// generic read/update/delete tenant guards.
func jobTemplateTaskWorkspaceOwnedSQL() string {
	return `SELECT EXISTS (
		SELECT 1 FROM ` + entityid.JobTemplateTask + ` jtt
		JOIN ` + entityid.JobTemplatePhase + ` jtp ON jtp.id = jtt.job_template_phase_id
		JOIN ` + entityid.JobTemplate + ` jt ON jt.id = jtp.job_template_id
		WHERE jtt.id = $1 AND ($2::text = '' OR jt.workspace_id = $2::text))`
}

// jobTemplatePhaseInWorkspaceSQL reports whether a job_template_phase (by id, $1)
// resolves to a job_template owned by the workspace ($2). Backs the task-create
// parent-FK validation.
func jobTemplatePhaseInWorkspaceSQL() string {
	return `SELECT EXISTS (
		SELECT 1 FROM ` + entityid.JobTemplatePhase + ` jtp
		JOIN ` + entityid.JobTemplate + ` jt ON jt.id = jtp.job_template_id
		WHERE jtp.id = $1 AND ($2::text = '' OR jt.workspace_id = $2::text))`
}

// jobTemplateTaskOwnedPhaseIDsSQL returns, from a candidate phase-id set ($1),
// those resolving to a job_template owned by the workspace ($2). Backs the
// generic-list tenant scope.
func jobTemplateTaskOwnedPhaseIDsSQL() string {
	return `SELECT jtp.id FROM ` + entityid.JobTemplatePhase + ` jtp
		JOIN ` + entityid.JobTemplate + ` jt ON jt.id = jtp.job_template_id
		WHERE jtp.id = ANY($1) AND ($2::text = '' OR jt.workspace_id = $2::text)`
}

// ensureTaskInWorkspace fails a by-id generic op closed when the task's parent
// chain is not owned by the caller's workspace. An empty workspace (a service
// context with no tenant bound) skips the check.
func (r *PostgresJobTemplateTaskRepository) ensureTaskInWorkspace(ctx context.Context, taskID string) error {
	wsID := identity.Must(ctx).WorkspaceID
	if wsID == "" {
		return nil
	}
	var owned bool
	if err := r.db.QueryRowContext(ctx, jobTemplateTaskWorkspaceOwnedSQL(), taskID, wsID).Scan(&owned); err != nil {
		return fmt.Errorf("failed to verify job template task workspace: %w", err)
	}
	if !owned {
		return fmt.Errorf("job template task with ID '%s' not found", taskID)
	}
	return nil
}

// ensurePhaseInWorkspace rejects a task create whose parent job_template_phase FK
// resolves to another tenant's template.
func (r *PostgresJobTemplateTaskRepository) ensurePhaseInWorkspace(ctx context.Context, phaseID string) error {
	wsID := identity.Must(ctx).WorkspaceID
	if wsID == "" {
		return nil
	}
	var owned bool
	if err := r.db.QueryRowContext(ctx, jobTemplatePhaseInWorkspaceSQL(), phaseID, wsID).Scan(&owned); err != nil {
		return fmt.Errorf("failed to verify job template phase workspace: %w", err)
	}
	if !owned {
		return fmt.Errorf("job template phase with ID '%s' not found", phaseID)
	}
	return nil
}

// filterTasksByWorkspace drops tasks whose parent chain is not owned by the
// caller's workspace. An empty workspace leaves the set unscoped.
func (r *PostgresJobTemplateTaskRepository) filterTasksByWorkspace(ctx context.Context, tasks []*pb.JobTemplateTask) ([]*pb.JobTemplateTask, error) {
	wsID := identity.Must(ctx).WorkspaceID
	if wsID == "" || len(tasks) == 0 {
		return tasks, nil
	}

	seen := make(map[string]struct{}, len(tasks))
	ids := make([]string, 0, len(tasks))
	for _, t := range tasks {
		pid := t.GetJobTemplatePhaseId()
		if pid == "" {
			continue
		}
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		ids = append(ids, pid)
	}

	owned := make(map[string]struct{}, len(ids))
	if len(ids) > 0 {
		rows, err := r.db.QueryContext(ctx, jobTemplateTaskOwnedPhaseIDsSQL(), sqlStringArray(ids), wsID)
		if err != nil {
			return nil, fmt.Errorf("failed to scope job template tasks by workspace: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return nil, fmt.Errorf("failed to scan owned job template phase id: %w", err)
			}
			owned[id] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating owned job template phase ids: %w", err)
		}
	}

	out := make([]*pb.JobTemplateTask, 0, len(tasks))
	for _, t := range tasks {
		if _, ok := owned[t.GetJobTemplatePhaseId()]; ok {
			out = append(out, t)
		}
	}
	return out, nil
}

// ListByPhase retrieves all tasks for a given phase, ordered by step_order
func (r *PostgresJobTemplateTaskRepository) ListByPhase(
	ctx context.Context,
	req *pb.ListJobTemplateTasksByPhaseRequest,
) (*pb.ListJobTemplateTasksByPhaseResponse, error) {
	if req == nil || req.JobTemplatePhaseId == "" {
		return nil, fmt.Errorf("job template phase ID is required")
	}

	query := jobTemplateTaskListByPhaseSQL()

	// Cross-tenant scope up the parent chain ($2). This method is also consumed by
	// template-materialization, which may run in a service context with no tenant
	// bound; a soft identity read leaves that path unscoped (empty workspace)
	// while an authenticated caller is confined to its own workspace.
	wsID := ""
	if id, ok := identity.FromContext(ctx); ok {
		wsID = id.WorkspaceID
	}

	rows, err := r.db.QueryContext(ctx, query, req.JobTemplatePhaseId, wsID)
	if err != nil {
		return nil, fmt.Errorf("failed to list job template tasks by phase: %w", err)
	}
	defer rows.Close()

	var tasks []*pb.JobTemplateTask
	for rows.Next() {
		var (
			id                       string
			dateCreated              time.Time
			dateModified             time.Time
			active                   bool
			jobTemplatePhaseID       string
			name                     string
			stepOrder                sql.NullInt32
			estimatedDurationMinutes *int32
			code                     sql.NullString
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&jobTemplatePhaseID,
			&name,
			&stepOrder,
			&estimatedDurationMinutes,
			&code,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job template task row: %w", err)
		}

		task := &pb.JobTemplateTask{
			Id:                 id,
			Active:             active,
			JobTemplatePhaseId: jobTemplatePhaseID,
			Name:               name,
			StepOrder:          stepOrder.Int32,
		}

		if estimatedDurationMinutes != nil {
			task.EstimatedDurationMinutes = estimatedDurationMinutes
		}
		if code.Valid {
			v := code.String
			task.Code = &v
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			task.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			task.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			task.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			task.DateModifiedString = &dmStr
		}

		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job template task rows: %w", err)
	}

	return &pb.ListJobTemplateTasksByPhaseResponse{
		JobTemplateTasks: tasks,
		Success:          true,
	}, nil
}

// NewJobTemplateTaskRepository creates a new PostgreSQL job_template_task repository (old-style constructor)
func NewJobTemplateTaskRepository(db *sql.DB, tableName string) pb.JobTemplateTaskDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresJobTemplateTaskRepository(dbOps, tableName)
}
