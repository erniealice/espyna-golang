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
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.JobTemplateTask, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres job_template_task repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
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

	return &pb.ListJobTemplateTasksResponse{
		Success: true,
		Data:    tasks,
	}, nil
}

// GetJobTemplateTaskListPageData retrieves tasks with pagination, filtering, sorting, and search
func (r *PostgresJobTemplateTaskRepository) GetJobTemplateTaskListPageData(
	ctx context.Context,
	req *pb.GetJobTemplateTaskListPageDataRequest,
) (*pb.GetJobTemplateTaskListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job template task list page data request is required")
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

	sortField := "jtt.step_order"
	sortOrder := "ASC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_DESC {
			sortOrder = "DESC"
		}
	}

	query := `
		WITH enriched AS (
			SELECT
				jtt.id,
				jtt.date_created,
				jtt.date_modified,
				jtt.active,
				jtt.job_template_phase_id,
				jtt.name,
				jtt.step_order,
				jtt.estimated_duration_minutes
			FROM job_template_task jtt
			WHERE jtt.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       jtt.name ILIKE $1)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
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
			stepOrder                int32
			estimatedDurationMinutes *int32
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
			StepOrder:          stepOrder,
		}

		if estimatedDurationMinutes != nil {
			task.EstimatedDurationMinutes = estimatedDurationMinutes
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

	query := `
		SELECT
			jtt.id,
			jtt.date_created,
			jtt.date_modified,
			jtt.active,
			jtt.job_template_phase_id,
			jtt.name,
			jtt.step_order,
			jtt.estimated_duration_minutes
		FROM job_template_task jtt
		WHERE jtt.id = $1 AND jtt.active = true
	`

	row := r.db.QueryRowContext(ctx, query, req.JobTemplateTaskId)

	var (
		id                       string
		dateCreated              time.Time
		dateModified             time.Time
		active                   bool
		jobTemplatePhaseID       string
		name                     string
		stepOrder                int32
		estimatedDurationMinutes *int32
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
		StepOrder:          stepOrder,
	}

	if estimatedDurationMinutes != nil {
		task.EstimatedDurationMinutes = estimatedDurationMinutes
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

// ListByPhase retrieves all tasks for a given phase, ordered by step_order
func (r *PostgresJobTemplateTaskRepository) ListByPhase(
	ctx context.Context,
	req *pb.ListJobTemplateTasksByPhaseRequest,
) (*pb.ListJobTemplateTasksByPhaseResponse, error) {
	if req == nil || req.JobTemplatePhaseId == "" {
		return nil, fmt.Errorf("job template phase ID is required")
	}

	query := `
		SELECT
			jtt.id,
			jtt.date_created,
			jtt.date_modified,
			jtt.active,
			jtt.job_template_phase_id,
			jtt.name,
			jtt.step_order,
			jtt.estimated_duration_minutes
		FROM job_template_task jtt
		WHERE jtt.job_template_phase_id = $1 AND jtt.active = true
		ORDER BY jtt.step_order ASC
	`

	rows, err := r.db.QueryContext(ctx, query, req.JobTemplatePhaseId)
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
			stepOrder                int32
			estimatedDurationMinutes *int32
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
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job template task row: %w", err)
		}

		task := &pb.JobTemplateTask{
			Id:                 id,
			Active:             active,
			JobTemplatePhaseId: jobTemplatePhaseID,
			Name:               name,
			StepOrder:          stepOrder,
		}

		if estimatedDurationMinutes != nil {
			task.EstimatedDurationMinutes = estimatedDurationMinutes
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
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresJobTemplateTaskRepository(dbOps, tableName)
}
