//go:build sqlserver

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/shared/identity"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_task"
)

// jobTaskSortableSQLCols lists the SQL column names safe to sort by.
var jobTaskSortableSQLCols = []string{
	"jt.step_order",
	"jt.date_created",
	"jt.date_modified",
	"jt.name",
	"jt.status",
}

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.JobTask, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver job_task repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerJobTaskRepository(dbOps, tableName), nil
	})
}

// SQLServerJobTaskRepository implements job_task CRUD operations using SQL Server.
type SQLServerJobTaskRepository struct {
	pb.UnimplementedJobTaskDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerJobTaskRepository creates a new SQL Server job_task repository.
func NewSQLServerJobTaskRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobTaskDomainServiceServer {
	if tableName == "" {
		tableName = "job_task"
	}
	return &SQLServerJobTaskRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateJobTask creates a new job task record.
func (r *SQLServerJobTaskRepository) CreateJobTask(ctx context.Context, req *pb.CreateJobTaskRequest) (*pb.CreateJobTaskResponse, error) {
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

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create job task: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
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

// ReadJobTask retrieves a job task record by ID.
func (r *SQLServerJobTaskRepository) ReadJobTask(ctx context.Context, req *pb.ReadJobTaskRequest) (*pb.ReadJobTaskResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job task ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job task: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	task := &pb.JobTask{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadJobTaskResponse{
		Success: true,
		Data:    []*pb.JobTask{task},
	}, nil
}

// UpdateJobTask updates a job task record.
func (r *SQLServerJobTaskRepository) UpdateJobTask(ctx context.Context, req *pb.UpdateJobTaskRequest) (*pb.UpdateJobTaskResponse, error) {
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

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update job task: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
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

// DeleteJobTask soft-deletes a job task record.
func (r *SQLServerJobTaskRepository) DeleteJobTask(ctx context.Context, req *pb.DeleteJobTaskRequest) (*pb.DeleteJobTaskResponse, error) {
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

// ListJobTasks lists job task records with optional filters.
func (r *SQLServerJobTaskRepository) ListJobTasks(ctx context.Context, req *pb.ListJobTasksRequest) (*pb.ListJobTasksResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list job tasks: %w", err)
	}

	var tasks []*pb.JobTask
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal job_task row: %v", err)
			continue
		}

		task := &pb.JobTask{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, task); err != nil {
			log.Printf("WARN: protojson unmarshal job_task: %v", err)
			continue
		}
		tasks = append(tasks, task)
	}

	return &pb.ListJobTasksResponse{
		Success: true,
		Data:    tasks,
	}, nil
}

// GetJobTaskListPageData retrieves job tasks with pagination, filtering, sorting, and search.
//
// SQL Server differences vs postgres gold standard:
//   - ILIKE → LIKE; $N → @pN; active = true → active = 1.
//   - Pagination: OFFSET @pM ROWS FETCH NEXT @pN ROWS ONLY (ORDER BY required).
//   - workspace_id filter added for multi-tenancy.
//   - COUNT(*) OVER () retained — SQL Server 2017+ supports it.
func (r *SQLServerJobTaskRepository) GetJobTaskListPageData(
	ctx context.Context,
	req *pb.GetJobTaskListPageDataRequest,
) (*pb.GetJobTaskListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job task list page data request is required")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

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

	orderByClause, err := sqlserverCore.BuildOrderBy(jobTaskSortableSQLCols, req.GetSort(), "jt.step_order ASC")
	if err != nil {
		return nil, err
	}

	whereSQL := "WHERE jt.workspace_id = @p1 AND jt.active = 1"
	queryArgs := []any{workspaceID}
	nextIdx := 2

	if searchPattern != "" {
		whereSQL += fmt.Sprintf(" AND (jt.name LIKE @p%d)", nextIdx)
		queryArgs = append(queryArgs, searchPattern)
		nextIdx++
	}

	offsetIdx := nextIdx
	limitIdx := nextIdx + 1
	queryArgs = append(queryArgs, offset, limit)

	query := fmt.Sprintf(`
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
			%s
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		%s OFFSET @p%d ROWS FETCH NEXT @p%d ROWS ONLY;
	`, whereSQL, orderByClause, offsetIdx, limitIdx)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, queryArgs...)
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

// GetJobTaskItemPageData retrieves a single job task with enriched data.
func (r *SQLServerJobTaskRepository) GetJobTaskItemPageData(
	ctx context.Context,
	req *pb.GetJobTaskItemPageDataRequest,
) (*pb.GetJobTaskItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job task item page data request is required")
	}
	if req.JobTaskId == "" {
		return nil, fmt.Errorf("job task ID is required")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

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
		WHERE jt.id = @p1 AND jt.workspace_id = @p2 AND jt.active = 1
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.JobTaskId, workspaceID)

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

// ListByPhase retrieves all tasks for a given job phase, ordered by step_order.
func (r *SQLServerJobTaskRepository) ListByPhase(
	ctx context.Context,
	req *pb.ListJobTasksByPhaseRequest,
) (*pb.ListJobTasksByPhaseResponse, error) {
	if req == nil || req.JobPhaseId == "" {
		return nil, fmt.Errorf("job phase ID is required")
	}

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
		WHERE jt.job_phase_id = @p1 AND jt.active = 1
		ORDER BY jt.step_order ASC
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, req.JobPhaseId)
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

// ListByAssignee retrieves all tasks assigned to a given user, ordered by date_created desc.
func (r *SQLServerJobTaskRepository) ListByAssignee(
	ctx context.Context,
	req *pb.ListJobTasksByAssigneeRequest,
) (*pb.ListJobTasksByAssigneeResponse, error) {
	if req == nil || req.AssignedTo == "" {
		return nil, fmt.Errorf("assignee ID is required")
	}

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
		WHERE jt.assigned_to = @p1 AND jt.active = 1
		ORDER BY jt.date_created DESC
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, req.AssignedTo)
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
