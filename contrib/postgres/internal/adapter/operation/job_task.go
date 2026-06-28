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
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_task"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.JobTask, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres job_task repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresJobTaskRepository(dbOps, tableName), nil
	})
}

// PostgresJobTaskRepository implements job_task CRUD operations using PostgreSQL
type PostgresJobTaskRepository struct {
	pb.UnimplementedJobTaskDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresJobTaskRepository creates a new PostgreSQL job_task repository
func NewPostgresJobTaskRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobTaskDomainServiceServer {
	if tableName == "" {
		tableName = "job_task"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresJobTaskRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateJobTask creates a new job task record
func (r *PostgresJobTaskRepository) CreateJobTask(ctx context.Context, req *pb.CreateJobTaskRequest) (*pb.CreateJobTaskResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("job task data is required")
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

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create job task: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	task := &pb.JobTask{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateJobTaskResponse{
		Success: true,
		Data:    []*pb.JobTask{task},
	}, nil
}

// ReadJobTask retrieves a job task record by ID
func (r *PostgresJobTaskRepository) ReadJobTask(ctx context.Context, req *pb.ReadJobTaskRequest) (*pb.ReadJobTaskResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job task ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job task: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	task := &pb.JobTask{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	// Staff row-scope (Phase 4): the generic dbOps.Read by id has no WHERE seam,
	// so enforce the IDOR guard post-read — a STAFF principal may only read a
	// job_task assigned to itself. Fail-closed: empty session staff.id, or a row
	// with no/other assignee, is treated as not-found. Non-staff principals are
	// unaffected (workspace scoping already applied by dbOps).
	if staffID, ok := staffRowScope(ctx); ok {
		if staffID == "" || task.AssignedTo == nil || *task.AssignedTo != staffID {
			return nil, fmt.Errorf("job task with ID '%s' not found", req.Data.Id)
		}
	}

	return &pb.ReadJobTaskResponse{
		Success: true,
		Data:    []*pb.JobTask{task},
	}, nil
}

// UpdateJobTask updates a job task record
func (r *PostgresJobTaskRepository) UpdateJobTask(ctx context.Context, req *pb.UpdateJobTaskRequest) (*pb.UpdateJobTaskResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job task ID is required")
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

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update job task: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	task := &pb.JobTask{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateJobTaskResponse{
		Success: true,
		Data:    []*pb.JobTask{task},
	}, nil
}

// DeleteJobTask deletes a job task record (soft delete)
func (r *PostgresJobTaskRepository) DeleteJobTask(ctx context.Context, req *pb.DeleteJobTaskRequest) (*pb.DeleteJobTaskResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job task ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete job task: %w", err)
	}

	return &pb.DeleteJobTaskResponse{
		Success: true,
	}, nil
}

// ListJobTasks lists job task records with optional filters
func (r *PostgresJobTaskRepository) ListJobTasks(ctx context.Context, req *pb.ListJobTasksRequest) (*pb.ListJobTasksResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list job tasks: %w", err)
	}

	staffID, staffScoped := staffRowScope(ctx)
	var tasks []*pb.JobTask
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal job_task row: %v", err)
			continue
		}

		task := &pb.JobTask{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, task); err != nil {
			log.Printf("WARN: protojson unmarshal job_task: %v", err)
			continue
		}
		// Staff row-scope (Phase 4 IDOR): the generic dbOps.List applies workspace
		// scoping only — a STAFF principal must see only its OWN assigned tasks
		// (matches the ReadJobTask post-read guard + the live POST
		// /api/operation/job-task/list route). Fail-closed: empty session staff.id
		// ⇒ no rows. Non-staff / no-session (e.g. server orchestration) unaffected.
		if staffScoped && (staffID == "" || task.AssignedTo == nil || *task.AssignedTo != staffID) {
			continue
		}
		tasks = append(tasks, task)
	}

	return &pb.ListJobTasksResponse{
		Success: true,
		Data:    tasks,
	}, nil
}

var jobTaskSortableSQLCols = []string{
	"id", "date_created", "date_modified", "active", "job_phase_id",
	"name", "step_order", "status", "is_ad_hoc", "assigned_to",
}

// GetJobTaskListPageData retrieves job tasks with pagination, filtering, sorting, and search
func (r *PostgresJobTaskRepository) GetJobTaskListPageData(
	ctx context.Context,
	req *pb.GetJobTaskListPageDataRequest,
) (*pb.GetJobTaskListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job task list page data request is required")
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

	// Sort — fail-closed against the per-entity whitelist (A2 guard). The ORDER BY
	// runs against the outer `enriched e` projection (unprefixed cols), so the
	// whitelist + fallback are unprefixed. Default preserves step_order ASC.
	orderByClause, err := postgresCore.BuildOrderBy(jobTaskSortableSQLCols, req.GetSort(), "step_order ASC")
	if err != nil {
		return nil, err
	}

	// Staff row-scope (Phase 4): a STAFF principal sees only job_tasks assigned to
	// itself ($4, from the session — never a request param). Non-staff principals
	// get the empty clause (unchanged). The predicate lives inside the enriched
	// CTE so the counted total reflects the scoped set.
	staffClause, staffArgs := staffScopeClause(ctx, "jt.assigned_to", 4)

	query := `
		WITH enriched AS (
			SELECT
				jt.id,
				jt.date_created,
				jt.date_modified,
				jt.active,
				jt.job_phase_id,
				jt.name,
				jt.step_order,
				jt.status,
				jt.is_ad_hoc,
				jt.assigned_to
			FROM job_task jt
			WHERE jt.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       jt.name ILIKE $1)` + staffClause + `
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		` + orderByClause + `
		LIMIT $2 OFFSET $3;
	`

	args := append([]any{searchPattern, limit, offset}, staffArgs...)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query job task list page data: %w", err)
	}
	defer rows.Close()

	var tasks []*pb.JobTask
	var totalCount int64

	for rows.Next() {
		var (
			id           string
			dateCreated  time.Time
			dateModified time.Time
			active       bool
			jobPhaseID   string
			name         string
			stepOrder    int32
			status       string
			isAdHoc      bool
			assignedTo   sql.NullString
			total        int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&jobPhaseID,
			&name,
			&stepOrder,
			&status,
			&isAdHoc,
			&assignedTo,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job task row: %w", err)
		}

		totalCount = total

		task := &pb.JobTask{
			Id:         id,
			Active:     active,
			JobPhaseId: jobPhaseID,
			Name:       name,
			StepOrder:  stepOrder,
			IsAdHoc:    isAdHoc,
		}

		if v, ok := pb.TaskStatus_value[status]; ok {
			task.Status = pb.TaskStatus(v)
		}

		if assignedTo.Valid {
			task.AssignedTo = &assignedTo.String
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
		return nil, fmt.Errorf("error iterating job task rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetJobTaskListPageDataResponse{
		JobTaskList: tasks,
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

// GetJobTaskItemPageData retrieves a single job task with enriched data
func (r *PostgresJobTaskRepository) GetJobTaskItemPageData(
	ctx context.Context,
	req *pb.GetJobTaskItemPageDataRequest,
) (*pb.GetJobTaskItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job task item page data request is required")
	}
	if req.JobTaskId == "" {
		return nil, fmt.Errorf("job task ID is required")
	}

	// Staff row-scope (Phase 4): a STAFF principal may only read a job_task
	// assigned to itself ($2, from the session). Fail-closed → not-found on a
	// foreign or unassigned row. Non-staff principals get the empty clause.
	staffClause, staffArgs := staffScopeClause(ctx, "jt.assigned_to", 2)

	query := `
		SELECT
			jt.id,
			jt.date_created,
			jt.date_modified,
			jt.active,
			jt.job_phase_id,
			jt.name,
			jt.step_order,
			jt.status,
			jt.is_ad_hoc,
			jt.assigned_to
		FROM job_task jt
		WHERE jt.id = $1 AND jt.active = true` + staffClause + `
	`

	row := r.db.QueryRowContext(ctx, query, append([]any{req.JobTaskId}, staffArgs...)...)

	var (
		id           string
		dateCreated  time.Time
		dateModified time.Time
		active       bool
		jobPhaseID   string
		name         string
		stepOrder    int32
		status       string
		isAdHoc      bool
		assignedTo   sql.NullString
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&jobPhaseID,
		&name,
		&stepOrder,
		&status,
		&isAdHoc,
		&assignedTo,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job task with ID '%s' not found", req.JobTaskId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query job task item page data: %w", err)
	}

	task := &pb.JobTask{
		Id:         id,
		Active:     active,
		JobPhaseId: jobPhaseID,
		Name:       name,
		StepOrder:  stepOrder,
		IsAdHoc:    isAdHoc,
	}

	if v, ok := pb.TaskStatus_value[status]; ok {
		task.Status = pb.TaskStatus(v)
	}

	if assignedTo.Valid {
		task.AssignedTo = &assignedTo.String
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

	return &pb.GetJobTaskItemPageDataResponse{
		JobTask: task,
		Success: true,
	}, nil
}

// ListByPhase retrieves all tasks for a given job phase, ordered by step_order
func (r *PostgresJobTaskRepository) ListByPhase(
	ctx context.Context,
	req *pb.ListJobTasksByPhaseRequest,
) (*pb.ListJobTasksByPhaseResponse, error) {
	if req == nil || req.JobPhaseId == "" {
		return nil, fmt.Errorf("job phase ID is required")
	}

	// Staff row-scope (Phase 4): within a phase a STAFF principal sees only the
	// tasks assigned to itself ($2, session-derived). Non-staff → empty clause.
	staffClause, staffArgs := staffScopeClause(ctx, "jt.assigned_to", 2)

	query := `
		SELECT
			jt.id,
			jt.date_created,
			jt.date_modified,
			jt.active,
			jt.job_phase_id,
			jt.name,
			jt.step_order,
			jt.status,
			jt.is_ad_hoc,
			jt.assigned_to
		FROM job_task jt
		WHERE jt.job_phase_id = $1 AND jt.active = true` + staffClause + `
		ORDER BY jt.step_order ASC
	`

	rows, err := r.db.QueryContext(ctx, query, append([]any{req.JobPhaseId}, staffArgs...)...)
	if err != nil {
		return nil, fmt.Errorf("failed to list job tasks by phase: %w", err)
	}
	defer rows.Close()

	var tasks []*pb.JobTask
	for rows.Next() {
		var (
			id           string
			dateCreated  time.Time
			dateModified time.Time
			active       bool
			jobPhaseID   string
			name         string
			stepOrder    int32
			status       string
			isAdHoc      bool
			assignedTo   sql.NullString
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&jobPhaseID,
			&name,
			&stepOrder,
			&status,
			&isAdHoc,
			&assignedTo,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job task row: %w", err)
		}

		task := &pb.JobTask{
			Id:         id,
			Active:     active,
			JobPhaseId: jobPhaseID,
			Name:       name,
			StepOrder:  stepOrder,
			IsAdHoc:    isAdHoc,
		}

		if v, ok := pb.TaskStatus_value[status]; ok {
			task.Status = pb.TaskStatus(v)
		}

		if assignedTo.Valid {
			task.AssignedTo = &assignedTo.String
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
		return nil, fmt.Errorf("error iterating job task rows: %w", err)
	}

	return &pb.ListJobTasksByPhaseResponse{
		JobTasks: tasks,
		Success:  true,
	}, nil
}

// ListByAssignee retrieves all tasks assigned to a given user, ordered by date_created desc
func (r *PostgresJobTaskRepository) ListByAssignee(
	ctx context.Context,
	req *pb.ListJobTasksByAssigneeRequest,
) (*pb.ListJobTasksByAssigneeResponse, error) {
	if req == nil || req.AssignedTo == "" {
		return nil, fmt.Errorf("assignee ID is required")
	}

	// Staff row-scope (Phase 4) — IDOR-critical: req.AssignedTo is a REQUEST
	// param, so a STAFF principal must additionally be pinned to its own
	// session staff.id ($2). The two predicates intersect: a staff asking for
	// another staff's tasks gets zero rows. Non-staff principals are unchanged.
	staffClause, staffArgs := staffScopeClause(ctx, "jt.assigned_to", 2)

	query := `
		SELECT
			jt.id,
			jt.date_created,
			jt.date_modified,
			jt.active,
			jt.job_phase_id,
			jt.name,
			jt.step_order,
			jt.status,
			jt.is_ad_hoc,
			jt.assigned_to
		FROM job_task jt
		WHERE jt.assigned_to = $1 AND jt.active = true` + staffClause + `
		ORDER BY jt.date_created DESC
	`

	rows, err := r.db.QueryContext(ctx, query, append([]any{req.AssignedTo}, staffArgs...)...)
	if err != nil {
		return nil, fmt.Errorf("failed to list job tasks by assignee: %w", err)
	}
	defer rows.Close()

	var tasks []*pb.JobTask
	for rows.Next() {
		var (
			id           string
			dateCreated  time.Time
			dateModified time.Time
			active       bool
			jobPhaseID   string
			name         string
			stepOrder    int32
			status       string
			isAdHoc      bool
			assignedTo   sql.NullString
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&jobPhaseID,
			&name,
			&stepOrder,
			&status,
			&isAdHoc,
			&assignedTo,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job task row: %w", err)
		}

		task := &pb.JobTask{
			Id:         id,
			Active:     active,
			JobPhaseId: jobPhaseID,
			Name:       name,
			StepOrder:  stepOrder,
			IsAdHoc:    isAdHoc,
		}

		if v, ok := pb.TaskStatus_value[status]; ok {
			task.Status = pb.TaskStatus(v)
		}

		if assignedTo.Valid {
			task.AssignedTo = &assignedTo.String
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
		return nil, fmt.Errorf("error iterating job task rows: %w", err)
	}

	return &pb.ListJobTasksByAssigneeResponse{
		JobTasks: tasks,
		Success:  true,
	}, nil
}

// NewJobTaskRepository creates a new PostgreSQL job_task repository (old-style constructor)
func NewJobTaskRepository(db *sql.DB, tableName string) pb.JobTaskDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresJobTaskRepository(dbOps, tableName)
}
