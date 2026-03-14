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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.JobTemplate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres job_template repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresJobTemplateRepository(dbOps, tableName), nil
	})
}

// PostgresJobTemplateRepository implements job_template CRUD operations using PostgreSQL
type PostgresJobTemplateRepository struct {
	pb.UnimplementedJobTemplateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresJobTemplateRepository creates a new PostgreSQL job_template repository
func NewPostgresJobTemplateRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobTemplateDomainServiceServer {
	if tableName == "" {
		tableName = "job_template"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresJobTemplateRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateJobTemplate creates a new job template record
func (r *PostgresJobTemplateRepository) CreateJobTemplate(ctx context.Context, req *pb.CreateJobTemplateRequest) (*pb.CreateJobTemplateResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("job template data is required")
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
		return nil, fmt.Errorf("failed to create job template: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	template := &pb.JobTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, template); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.CreateJobTemplateResponse{
		Success: true,
		Data:    []*pb.JobTemplate{template},
	}, nil
}

// ReadJobTemplate retrieves a job template record by ID
func (r *PostgresJobTemplateRepository) ReadJobTemplate(ctx context.Context, req *pb.ReadJobTemplateRequest) (*pb.ReadJobTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job template ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read job template: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	template := &pb.JobTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, template); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.ReadJobTemplateResponse{
		Success: true,
		Data:    []*pb.JobTemplate{template},
	}, nil
}

// UpdateJobTemplate updates a job template record
func (r *PostgresJobTemplateRepository) UpdateJobTemplate(ctx context.Context, req *pb.UpdateJobTemplateRequest) (*pb.UpdateJobTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job template ID is required")
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
		return nil, fmt.Errorf("failed to update job template: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	template := &pb.JobTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, template); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pb.UpdateJobTemplateResponse{
		Success: true,
		Data:    []*pb.JobTemplate{template},
	}, nil
}

// DeleteJobTemplate deletes a job template record (soft delete)
func (r *PostgresJobTemplateRepository) DeleteJobTemplate(ctx context.Context, req *pb.DeleteJobTemplateRequest) (*pb.DeleteJobTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job template ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete job template: %w", err)
	}

	return &pb.DeleteJobTemplateResponse{
		Success: true,
	}, nil
}

// ListJobTemplates lists job template records with optional filters
func (r *PostgresJobTemplateRepository) ListJobTemplates(ctx context.Context, req *pb.ListJobTemplatesRequest) (*pb.ListJobTemplatesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list job templates: %w", err)
	}

	var templates []*pb.JobTemplate
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal job_template row: %v", err)
			continue
		}

		template := &pb.JobTemplate{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, template); err != nil {
			log.Printf("WARN: protojson unmarshal job_template: %v", err)
			continue
		}
		templates = append(templates, template)
	}

	return &pb.ListJobTemplatesResponse{
		Success: true,
		Data:    templates,
	}, nil
}

// GetJobTemplateListPageData retrieves job templates with pagination, filtering, sorting, and search
func (r *PostgresJobTemplateRepository) GetJobTemplateListPageData(
	ctx context.Context,
	req *pb.GetJobTemplateListPageDataRequest,
) (*pb.GetJobTemplateListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job template list page data request is required")
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

	sortField := "jt.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_DESC {
			sortOrder = "DESC"
		}
	}

	query := `
		WITH enriched AS (
			SELECT
				jt.id,
				jt.date_created,
				jt.date_modified,
				jt.active,
				jt.name,
				jt.description,
				jt.default_fulfillment_type,
				jt.default_cost_flow_type,
				jt.default_billing_rule_type
			FROM job_template jt
			WHERE jt.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       jt.name ILIKE $1 OR
			       jt.description ILIKE $1)
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
		return nil, fmt.Errorf("failed to query job template list page data: %w", err)
	}
	defer rows.Close()

	var templates []*pb.JobTemplate
	var totalCount int64

	for rows.Next() {
		var (
			id                     string
			dateCreated            time.Time
			dateModified           time.Time
			active                 bool
			name                   string
			description            sql.NullString
			defaultFulfillmentType sql.NullString
			defaultCostFlowType    sql.NullString
			defaultBillingRuleType sql.NullString
			total                  int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&name,
			&description,
			&defaultFulfillmentType,
			&defaultCostFlowType,
			&defaultBillingRuleType,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job template row: %w", err)
		}

		totalCount = total

		template := &pb.JobTemplate{
			Id:     id,
			Active: active,
			Name:   name,
		}

		if description.Valid {
			template.Description = &description.String
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			template.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			template.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			template.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			template.DateModifiedString = &dmStr
		}

		templates = append(templates, template)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job template rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pb.GetJobTemplateListPageDataResponse{
		JobTemplateList: templates,
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

// GetJobTemplateItemPageData retrieves a single job template with enriched data
func (r *PostgresJobTemplateRepository) GetJobTemplateItemPageData(
	ctx context.Context,
	req *pb.GetJobTemplateItemPageDataRequest,
) (*pb.GetJobTemplateItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get job template item page data request is required")
	}
	if req.JobTemplateId == "" {
		return nil, fmt.Errorf("job template ID is required")
	}

	query := `
		SELECT
			jt.id,
			jt.date_created,
			jt.date_modified,
			jt.active,
			jt.name,
			jt.description,
			jt.default_fulfillment_type,
			jt.default_cost_flow_type,
			jt.default_billing_rule_type
		FROM job_template jt
		WHERE jt.id = $1 AND jt.active = true
	`

	row := r.db.QueryRowContext(ctx, query, req.JobTemplateId)

	var (
		id                     string
		dateCreated            time.Time
		dateModified           time.Time
		active                 bool
		name                   string
		description            sql.NullString
		defaultFulfillmentType sql.NullString
		defaultCostFlowType    sql.NullString
		defaultBillingRuleType sql.NullString
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&name,
		&description,
		&defaultFulfillmentType,
		&defaultCostFlowType,
		&defaultBillingRuleType,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job template with ID '%s' not found", req.JobTemplateId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query job template item page data: %w", err)
	}

	template := &pb.JobTemplate{
		Id:     id,
		Active: active,
		Name:   name,
	}

	if description.Valid {
		template.Description = &description.String
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		template.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		template.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		template.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		template.DateModifiedString = &dmStr
	}

	return &pb.GetJobTemplateItemPageDataResponse{
		JobTemplate: template,
		Success:     true,
	}, nil
}

// NewJobTemplateRepository creates a new PostgreSQL job_template repository (old-style constructor)
func NewJobTemplateRepository(db *sql.DB, tableName string) pb.JobTemplateDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresJobTemplateRepository(dbOps, tableName)
}
