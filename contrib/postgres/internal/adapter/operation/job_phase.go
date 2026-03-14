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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.JobPhase, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres job_phase repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresJobPhaseRepository(dbOps, tableName), nil
	})
}

// PostgresJobPhaseRepository implements job_phase CRUD operations using PostgreSQL
type PostgresJobPhaseRepository struct {
	pb.UnimplementedJobPhaseDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresJobPhaseRepository creates a new PostgreSQL job_phase repository
func NewPostgresJobPhaseRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobPhaseDomainServiceServer {
	if tableName == "" {
		tableName = "job_phase"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresJobPhaseRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateJobPhase creates a new job phase record
func (r *PostgresJobPhaseRepository) CreateJobPhase(ctx context.Context, req *pb.CreateJobPhaseRequest) (*pb.CreateJobPhaseResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("job phase data is required")
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
		return nil, fmt.Errorf("failed to create job phase: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	phase := &pb.JobPhase{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, phase); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateJobPhaseResponse{
		Success: true,
		Data:    []*pb.JobPhase{phase},
	}, nil
}

// ReadJobPhase retrieves a job phase record by ID
func (r *PostgresJobPhaseRepository) ReadJobPhase(ctx context.Context, req *pb.ReadJobPhaseRequest) (*pb.ReadJobPhaseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job phase ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job phase: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	phase := &pb.JobPhase{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, phase); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadJobPhaseResponse{
		Success: true,
		Data:    []*pb.JobPhase{phase},
	}, nil
}

// UpdateJobPhase updates a job phase record
func (r *PostgresJobPhaseRepository) UpdateJobPhase(ctx context.Context, req *pb.UpdateJobPhaseRequest) (*pb.UpdateJobPhaseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job phase ID is required")
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
		return nil, fmt.Errorf("failed to update job phase: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	phase := &pb.JobPhase{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, phase); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateJobPhaseResponse{
		Success: true,
		Data:    []*pb.JobPhase{phase},
	}, nil
}

// DeleteJobPhase deletes a job phase record (soft delete)
func (r *PostgresJobPhaseRepository) DeleteJobPhase(ctx context.Context, req *pb.DeleteJobPhaseRequest) (*pb.DeleteJobPhaseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job phase ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete job phase: %w", err)
	}

	return &pb.DeleteJobPhaseResponse{
		Success: true,
	}, nil
}

// ListJobPhases lists job phase records with optional filters
func (r *PostgresJobPhaseRepository) ListJobPhases(ctx context.Context, req *pb.ListJobPhasesRequest) (*pb.ListJobPhasesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list job phases: %w", err)
	}

	var phases []*pb.JobPhase
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal job_phase row: %v", err)
			continue
		}

		phase := &pb.JobPhase{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, phase); err != nil {
			log.Printf("WARN: protojson unmarshal job_phase: %v", err)
			continue
		}
		phases = append(phases, phase)
	}

	return &pb.ListJobPhasesResponse{
		Success: true,
		Data:    phases,
	}, nil
}

// GetJobPhaseListPageData retrieves job phases with pagination, filtering, sorting, and search
func (r *PostgresJobPhaseRepository) GetJobPhaseListPageData(
	ctx context.Context,
	req *pb.GetJobPhaseListPageDataRequest,
) (*pb.GetJobPhaseListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job phase list page data request is required")
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

	sortField := "jp.phase_order"
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
				jp.id,
				jp.date_created,
				jp.date_modified,
				jp.active,
				jp.job_id,
				jp.name,
				jp.phase_order,
				jp.status
			FROM job_phase jp
			WHERE jp.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       jp.name ILIKE $1)
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
		return nil, fmt.Errorf("failed to query job phase list page data: %w", err)
	}
	defer rows.Close()

	var phases []*pb.JobPhase
	var totalCount int64

	for rows.Next() {
		var (
			id           string
			dateCreated  time.Time
			dateModified time.Time
			active       bool
			jobID        string
			name         string
			phaseOrder   int32
			status       string
			total        int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&jobID,
			&name,
			&phaseOrder,
			&status,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job phase row: %w", err)
		}

		totalCount = total

		phase := &pb.JobPhase{
			Id:         id,
			Active:     active,
			JobId:      jobID,
			Name:       name,
			PhaseOrder: phaseOrder,
		}

		// Map enum string to proto enum
		if v, ok := pb.PhaseStatus_value[status]; ok {
			phase.Status = pb.PhaseStatus(v)
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			phase.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			phase.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			phase.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			phase.DateModifiedString = &dmStr
		}

		phases = append(phases, phase)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job phase rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetJobPhaseListPageDataResponse{
		JobPhaseList: phases,
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

// GetJobPhaseItemPageData retrieves a single job phase with enriched data
func (r *PostgresJobPhaseRepository) GetJobPhaseItemPageData(
	ctx context.Context,
	req *pb.GetJobPhaseItemPageDataRequest,
) (*pb.GetJobPhaseItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job phase item page data request is required")
	}
	if req.JobPhaseId == "" {
		return nil, fmt.Errorf("job phase ID is required")
	}

	query := `
		SELECT
			jp.id,
			jp.date_created,
			jp.date_modified,
			jp.active,
			jp.job_id,
			jp.name,
			jp.phase_order,
			jp.status
		FROM job_phase jp
		WHERE jp.id = $1 AND jp.active = true
	`

	row := r.db.QueryRowContext(ctx, query, req.JobPhaseId)

	var (
		id           string
		dateCreated  time.Time
		dateModified time.Time
		active       bool
		jobID        string
		name         string
		phaseOrder   int32
		status       string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&jobID,
		&name,
		&phaseOrder,
		&status,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job phase with ID '%s' not found", req.JobPhaseId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query job phase item page data: %w", err)
	}

	phase := &pb.JobPhase{
		Id:         id,
		Active:     active,
		JobId:      jobID,
		Name:       name,
		PhaseOrder: phaseOrder,
	}

	if v, ok := pb.PhaseStatus_value[status]; ok {
		phase.Status = pb.PhaseStatus(v)
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		phase.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		phase.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		phase.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		phase.DateModifiedString = &dmStr
	}

	return &pb.GetJobPhaseItemPageDataResponse{
		JobPhase: phase,
		Success:  true,
	}, nil
}

// ListByJob retrieves all phases for a given job, ordered by phase_order
func (r *PostgresJobPhaseRepository) ListByJob(
	ctx context.Context,
	req *pb.ListJobPhasesByJobRequest,
) (*pb.ListJobPhasesByJobResponse, error) {
	if req == nil || req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}

	query := `
		SELECT
			jp.id,
			jp.date_created,
			jp.date_modified,
			jp.active,
			jp.job_id,
			jp.name,
			jp.phase_order,
			jp.status
		FROM job_phase jp
		WHERE jp.job_id = $1 AND jp.active = true
		ORDER BY jp.phase_order ASC
	`

	rows, err := r.db.QueryContext(ctx, query, req.JobId)
	if err != nil {
		return nil, fmt.Errorf("failed to list job phases by job: %w", err)
	}
	defer rows.Close()

	var phases []*pb.JobPhase
	for rows.Next() {
		var (
			id           string
			dateCreated  time.Time
			dateModified time.Time
			active       bool
			jobID        string
			name         string
			phaseOrder   int32
			status       string
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&jobID,
			&name,
			&phaseOrder,
			&status,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job phase row: %w", err)
		}

		phase := &pb.JobPhase{
			Id:         id,
			Active:     active,
			JobId:      jobID,
			Name:       name,
			PhaseOrder: phaseOrder,
		}

		if v, ok := pb.PhaseStatus_value[status]; ok {
			phase.Status = pb.PhaseStatus(v)
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			phase.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			phase.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			phase.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			phase.DateModifiedString = &dmStr
		}

		phases = append(phases, phase)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job phase rows: %w", err)
	}

	return &pb.ListJobPhasesByJobResponse{
		JobPhases: phases,
		Success:   true,
	}, nil
}

// NewJobPhaseRepository creates a new PostgreSQL job_phase repository (old-style constructor)
func NewJobPhaseRepository(db *sql.DB, tableName string) pb.JobPhaseDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresJobPhaseRepository(dbOps, tableName)
}
