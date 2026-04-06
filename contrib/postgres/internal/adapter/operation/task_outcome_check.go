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
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome_check"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.TaskOutcomeCheck, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres task_outcome_check repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresTaskOutcomeCheckRepository(dbOps, tableName), nil
	})
}

// PostgresTaskOutcomeCheckRepository implements task_outcome_check CRUD operations using PostgreSQL
type PostgresTaskOutcomeCheckRepository struct {
	pb.UnimplementedTaskOutcomeCheckDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresTaskOutcomeCheckRepository creates a new PostgreSQL task_outcome_check repository
func NewPostgresTaskOutcomeCheckRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.TaskOutcomeCheckDomainServiceServer {
	if tableName == "" {
		tableName = "task_outcome_check"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresTaskOutcomeCheckRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateTaskOutcomeCheck creates a new task_outcome_check record
func (r *PostgresTaskOutcomeCheckRepository) CreateTaskOutcomeCheck(ctx context.Context, req *pb.CreateTaskOutcomeCheckRequest) (*pb.CreateTaskOutcomeCheckResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("task outcome check data is required")
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

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create task outcome check: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	check := &pb.TaskOutcomeCheck{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, check); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateTaskOutcomeCheckResponse{
		Success: true,
		Data:    []*pb.TaskOutcomeCheck{check},
	}, nil
}

// ReadTaskOutcomeCheck retrieves a task_outcome_check record by ID
func (r *PostgresTaskOutcomeCheckRepository) ReadTaskOutcomeCheck(ctx context.Context, req *pb.ReadTaskOutcomeCheckRequest) (*pb.ReadTaskOutcomeCheckResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("task outcome check ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read task outcome check: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	check := &pb.TaskOutcomeCheck{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, check); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadTaskOutcomeCheckResponse{
		Success: true,
		Data:    []*pb.TaskOutcomeCheck{check},
	}, nil
}

// UpdateTaskOutcomeCheck updates a task_outcome_check record
func (r *PostgresTaskOutcomeCheckRepository) UpdateTaskOutcomeCheck(ctx context.Context, req *pb.UpdateTaskOutcomeCheckRequest) (*pb.UpdateTaskOutcomeCheckResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("task outcome check ID is required")
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

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update task outcome check: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	check := &pb.TaskOutcomeCheck{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, check); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateTaskOutcomeCheckResponse{
		Success: true,
		Data:    []*pb.TaskOutcomeCheck{check},
	}, nil
}

// DeleteTaskOutcomeCheck deletes a task_outcome_check record (soft delete)
func (r *PostgresTaskOutcomeCheckRepository) DeleteTaskOutcomeCheck(ctx context.Context, req *pb.DeleteTaskOutcomeCheckRequest) (*pb.DeleteTaskOutcomeCheckResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("task outcome check ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete task outcome check: %w", err)
	}

	return &pb.DeleteTaskOutcomeCheckResponse{
		Success: true,
	}, nil
}

// ListTaskOutcomeChecks lists task_outcome_check records with optional filters
func (r *PostgresTaskOutcomeCheckRepository) ListTaskOutcomeChecks(ctx context.Context, req *pb.ListTaskOutcomeChecksRequest) (*pb.ListTaskOutcomeChecksResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list task outcome checks: %w", err)
	}

	var checks []*pb.TaskOutcomeCheck
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal task_outcome_check row: %v", err)
			continue
		}

		check := &pb.TaskOutcomeCheck{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, check); err != nil {
			log.Printf("WARN: protojson unmarshal task_outcome_check: %v", err)
			continue
		}
		checks = append(checks, check)
	}

	return &pb.ListTaskOutcomeChecksResponse{
		Success: true,
		Data:    checks,
	}, nil
}

// GetTaskOutcomeCheckListPageData retrieves task outcome checks with pagination
func (r *PostgresTaskOutcomeCheckRepository) GetTaskOutcomeCheckListPageData(
	ctx context.Context,
	req *pb.GetTaskOutcomeCheckListPageDataRequest,
) (*pb.GetTaskOutcomeCheckListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get task outcome check list page data request is required")
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

	sortField := "toc.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	query := `
		WITH enriched AS (
			SELECT
				toc.id,
				toc.date_created,
				toc.task_outcome_id,
				toc.criteria_option_id,
				toc.checked,
				toc.note
			FROM task_outcome_check toc
			WHERE ($1::text IS NULL OR $1::text = '' OR
			       toc.note ILIKE $1)
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
		return nil, fmt.Errorf("failed to query task outcome check list page data: %w", err)
	}
	defer rows.Close()

	var checks []*pb.TaskOutcomeCheck
	var totalCount int64

	for rows.Next() {
		var (
			id               string
			dateCreated      time.Time
			taskOutcomeID    string
			criteriaOptionID string
			checked          bool
			note             *string
			total            int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&taskOutcomeID,
			&criteriaOptionID,
			&checked,
			&note,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task outcome check row: %w", err)
		}

		totalCount = total

		check := &pb.TaskOutcomeCheck{
			Id:               id,
			TaskOutcomeId:    taskOutcomeID,
			CriteriaOptionId: criteriaOptionID,
			Checked:          checked,
		}

		if note != nil {
			check.Note = note
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			check.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			check.DateCreatedString = &dcStr
		}

		checks = append(checks, check)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task outcome check rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetTaskOutcomeCheckListPageDataResponse{
		TaskOutcomeCheckList: checks,
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

// GetTaskOutcomeCheckItemPageData retrieves a single task outcome check with enriched data
func (r *PostgresTaskOutcomeCheckRepository) GetTaskOutcomeCheckItemPageData(
	ctx context.Context,
	req *pb.GetTaskOutcomeCheckItemPageDataRequest,
) (*pb.GetTaskOutcomeCheckItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get task outcome check item page data request is required")
	}
	if req.TaskOutcomeCheckId == "" {
		return nil, fmt.Errorf("task outcome check ID is required")
	}

	query := `
		SELECT
			toc.id,
			toc.date_created,
			toc.task_outcome_id,
			toc.criteria_option_id,
			toc.checked,
			toc.note
		FROM task_outcome_check toc
		WHERE toc.id = $1
	`

	row := r.db.QueryRowContext(ctx, query, req.TaskOutcomeCheckId)

	var (
		id               string
		dateCreated      time.Time
		taskOutcomeID    string
		criteriaOptionID string
		checked          bool
		note             *string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&taskOutcomeID,
		&criteriaOptionID,
		&checked,
		&note,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task outcome check with ID '%s' not found", req.TaskOutcomeCheckId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query task outcome check item page data: %w", err)
	}

	check := &pb.TaskOutcomeCheck{
		Id:               id,
		TaskOutcomeId:    taskOutcomeID,
		CriteriaOptionId: criteriaOptionID,
		Checked:          checked,
	}

	if note != nil {
		check.Note = note
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		check.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		check.DateCreatedString = &dcStr
	}

	return &pb.GetTaskOutcomeCheckItemPageDataResponse{
		TaskOutcomeCheck: check,
		Success:          true,
	}, nil
}

// ListByTaskOutcome retrieves all task outcome checks for a given task outcome
func (r *PostgresTaskOutcomeCheckRepository) ListByTaskOutcome(
	ctx context.Context,
	req *pb.ListTaskOutcomeChecksByTaskOutcomeRequest,
) (*pb.ListTaskOutcomeChecksByTaskOutcomeResponse, error) {
	if req == nil || req.TaskOutcomeId == "" {
		return nil, fmt.Errorf("task outcome ID is required")
	}

	query := `
		SELECT
			toc.id,
			toc.date_created,
			toc.task_outcome_id,
			toc.criteria_option_id,
			toc.checked,
			toc.note
		FROM task_outcome_check toc
		WHERE toc.task_outcome_id = $1
		ORDER BY toc.date_created ASC
	`

	rows, err := r.db.QueryContext(ctx, query, req.TaskOutcomeId)
	if err != nil {
		return nil, fmt.Errorf("failed to list task outcome checks by task outcome: %w", err)
	}
	defer rows.Close()

	var checks []*pb.TaskOutcomeCheck
	for rows.Next() {
		var (
			id               string
			dateCreated      time.Time
			taskOutcomeID    string
			criteriaOptionID string
			checked          bool
			note             *string
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&taskOutcomeID,
			&criteriaOptionID,
			&checked,
			&note,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task outcome check row: %w", err)
		}

		check := &pb.TaskOutcomeCheck{
			Id:               id,
			TaskOutcomeId:    taskOutcomeID,
			CriteriaOptionId: criteriaOptionID,
			Checked:          checked,
		}

		if note != nil {
			check.Note = note
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			check.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			check.DateCreatedString = &dcStr
		}

		checks = append(checks, check)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task outcome check rows: %w", err)
	}

	return &pb.ListTaskOutcomeChecksByTaskOutcomeResponse{
		TaskOutcomeChecks: checks,
		Success:           true,
	}, nil
}
