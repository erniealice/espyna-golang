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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.JobTemplatePhase, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres job_template_phase repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresJobTemplatePhaseRepository(dbOps, tableName), nil
	})
}

// PostgresJobTemplatePhaseRepository implements job_template_phase CRUD operations using PostgreSQL
type PostgresJobTemplatePhaseRepository struct {
	pb.UnimplementedJobTemplatePhaseDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresJobTemplatePhaseRepository creates a new PostgreSQL job_template_phase repository
func NewPostgresJobTemplatePhaseRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobTemplatePhaseDomainServiceServer {
	if tableName == "" {
		tableName = "job_template_phase"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresJobTemplatePhaseRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateJobTemplatePhase creates a new job template phase record
func (r *PostgresJobTemplatePhaseRepository) CreateJobTemplatePhase(ctx context.Context, req *pb.CreateJobTemplatePhaseRequest) (*pb.CreateJobTemplatePhaseResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("job template phase data is required")
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
		return nil, fmt.Errorf("failed to create job template phase: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	phase := &pb.JobTemplatePhase{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, phase); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateJobTemplatePhaseResponse{
		Success: true,
		Data:    []*pb.JobTemplatePhase{phase},
	}, nil
}

// ReadJobTemplatePhase retrieves a job template phase record by ID
func (r *PostgresJobTemplatePhaseRepository) ReadJobTemplatePhase(ctx context.Context, req *pb.ReadJobTemplatePhaseRequest) (*pb.ReadJobTemplatePhaseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job template phase ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job template phase: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	phase := &pb.JobTemplatePhase{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, phase); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadJobTemplatePhaseResponse{
		Success: true,
		Data:    []*pb.JobTemplatePhase{phase},
	}, nil
}

// UpdateJobTemplatePhase updates a job template phase record
func (r *PostgresJobTemplatePhaseRepository) UpdateJobTemplatePhase(ctx context.Context, req *pb.UpdateJobTemplatePhaseRequest) (*pb.UpdateJobTemplatePhaseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job template phase ID is required")
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
		return nil, fmt.Errorf("failed to update job template phase: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	phase := &pb.JobTemplatePhase{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, phase); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateJobTemplatePhaseResponse{
		Success: true,
		Data:    []*pb.JobTemplatePhase{phase},
	}, nil
}

// DeleteJobTemplatePhase deletes a job template phase record (soft delete)
func (r *PostgresJobTemplatePhaseRepository) DeleteJobTemplatePhase(ctx context.Context, req *pb.DeleteJobTemplatePhaseRequest) (*pb.DeleteJobTemplatePhaseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job template phase ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete job template phase: %w", err)
	}

	return &pb.DeleteJobTemplatePhaseResponse{
		Success: true,
	}, nil
}

// ListJobTemplatePhases lists job template phase records with optional filters
func (r *PostgresJobTemplatePhaseRepository) ListJobTemplatePhases(ctx context.Context, req *pb.ListJobTemplatePhasesRequest) (*pb.ListJobTemplatePhasesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list job template phases: %w", err)
	}

	var phases []*pb.JobTemplatePhase
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal job_template_phase row: %v", err)
			continue
		}

		phase := &pb.JobTemplatePhase{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, phase); err != nil {
			log.Printf("WARN: protojson unmarshal job_template_phase: %v", err)
			continue
		}
		phases = append(phases, phase)
	}

	return &pb.ListJobTemplatePhasesResponse{
		Success: true,
		Data:    phases,
	}, nil
}

// GetJobTemplatePhaseListPageData retrieves phases with pagination, filtering, sorting, and search
func (r *PostgresJobTemplatePhaseRepository) GetJobTemplatePhaseListPageData(
	ctx context.Context,
	req *pb.GetJobTemplatePhaseListPageDataRequest,
) (*pb.GetJobTemplatePhaseListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job template phase list page data request is required")
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

	sortField := "jtp.phase_order"
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
				jtp.id,
				jtp.date_created,
				jtp.date_modified,
				jtp.active,
				jtp.job_template_id,
				jtp.name,
				jtp.phase_order
			FROM job_template_phase jtp
			WHERE jtp.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       jtp.name ILIKE $1)
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
		return nil, fmt.Errorf("failed to query job template phase list page data: %w", err)
	}
	defer rows.Close()

	var phases []*pb.JobTemplatePhase
	var totalCount int64

	for rows.Next() {
		var (
			id             string
			dateCreated    time.Time
			dateModified   time.Time
			active         bool
			jobTemplateID  string
			name           string
			phaseOrder     int32
			total          int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&jobTemplateID,
			&name,
			&phaseOrder,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job template phase row: %w", err)
		}

		totalCount = total

		phase := &pb.JobTemplatePhase{
			Id:            id,
			Active:        active,
			JobTemplateId: jobTemplateID,
			Name:          name,
			PhaseOrder:    phaseOrder,
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
		return nil, fmt.Errorf("error iterating job template phase rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetJobTemplatePhaseListPageDataResponse{
		JobTemplatePhaseList: phases,
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

// GetJobTemplatePhaseItemPageData retrieves a single phase with enriched data
func (r *PostgresJobTemplatePhaseRepository) GetJobTemplatePhaseItemPageData(
	ctx context.Context,
	req *pb.GetJobTemplatePhaseItemPageDataRequest,
) (*pb.GetJobTemplatePhaseItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job template phase item page data request is required")
	}
	if req.JobTemplatePhaseId == "" {
		return nil, fmt.Errorf("job template phase ID is required")
	}

	query := `
		SELECT
			jtp.id,
			jtp.date_created,
			jtp.date_modified,
			jtp.active,
			jtp.job_template_id,
			jtp.name,
			jtp.phase_order
		FROM job_template_phase jtp
		WHERE jtp.id = $1 AND jtp.active = true
	`

	row := r.db.QueryRowContext(ctx, query, req.JobTemplatePhaseId)

	var (
		id             string
		dateCreated    time.Time
		dateModified   time.Time
		active         bool
		jobTemplateID  string
		name           string
		phaseOrder     int32
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&jobTemplateID,
		&name,
		&phaseOrder,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job template phase with ID '%s' not found", req.JobTemplatePhaseId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query job template phase item page data: %w", err)
	}

	phase := &pb.JobTemplatePhase{
		Id:            id,
		Active:        active,
		JobTemplateId: jobTemplateID,
		Name:          name,
		PhaseOrder:    phaseOrder,
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

	return &pb.GetJobTemplatePhaseItemPageDataResponse{
		JobTemplatePhase: phase,
		Success:          true,
	}, nil
}

// ListByJobTemplate retrieves all phases for a given job template, ordered by phase_order
func (r *PostgresJobTemplatePhaseRepository) ListByJobTemplate(
	ctx context.Context,
	req *pb.ListByJobTemplateRequest,
) (*pb.ListByJobTemplateResponse, error) {
	if req == nil || req.JobTemplateId == "" {
		return nil, fmt.Errorf("job template ID is required")
	}

	query := `
		SELECT
			jtp.id,
			jtp.date_created,
			jtp.date_modified,
			jtp.active,
			jtp.job_template_id,
			jtp.name,
			jtp.phase_order
		FROM job_template_phase jtp
		WHERE jtp.job_template_id = $1 AND jtp.active = true
		ORDER BY jtp.phase_order ASC
	`

	rows, err := r.db.QueryContext(ctx, query, req.JobTemplateId)
	if err != nil {
		return nil, fmt.Errorf("failed to list job template phases by template: %w", err)
	}
	defer rows.Close()

	var phases []*pb.JobTemplatePhase
	for rows.Next() {
		var (
			id             string
			dateCreated    time.Time
			dateModified   time.Time
			active         bool
			jobTemplateID  string
			name           string
			phaseOrder     int32
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&jobTemplateID,
			&name,
			&phaseOrder,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job template phase row: %w", err)
		}

		phase := &pb.JobTemplatePhase{
			Id:            id,
			Active:        active,
			JobTemplateId: jobTemplateID,
			Name:          name,
			PhaseOrder:    phaseOrder,
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
		return nil, fmt.Errorf("error iterating job template phase rows: %w", err)
	}

	return &pb.ListByJobTemplateResponse{
		JobTemplatePhases: phases,
		Success:           true,
	}, nil
}

// NewJobTemplatePhaseRepository creates a new PostgreSQL job_template_phase repository (old-style constructor)
func NewJobTemplatePhaseRepository(db *sql.DB, tableName string) pb.JobTemplatePhaseDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresJobTemplatePhaseRepository(dbOps, tableName)
}

// convertMillisToTime converts a millis-epoch value in a JSON map to time.Time.
// Protobuf int64 fields serialize to JSON strings via protojson (e.g. "1771886746000").
// Postgres timestamp columns need time.Time, not raw millis.
func convertMillisToTime(data map[string]any, jsonKey string) {
	v, ok := data[jsonKey]
	if !ok {
		return
	}
	switch val := v.(type) {
	case string:
		// protojson serializes int64 as string
		var millis int64
		if _, err := fmt.Sscanf(val, "%d", &millis); err == nil && millis > 1e12 {
			data[jsonKey] = time.UnixMilli(millis)
		}
	case float64:
		if val > 1e12 {
			data[jsonKey] = time.UnixMilli(int64(val))
		}
	}
}
