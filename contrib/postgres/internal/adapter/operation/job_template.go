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

	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/identity"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.JobTemplate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres job_template repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
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

	// protojson drops a false `active` (proto3 zero-omission), which would leave
	// the column unwritten on this partial UPDATE — unchecking Active + Save
	// would silently no-op. Force the field so a deactivation persists.
	postgresCore.ForceBoolField(req.Data, data, "active")

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

var jobTemplateSortableSQLCols = []string{
	"id", "active", "name", "description",
	"default_fulfillment_type", "default_cost_flow_type", "default_billing_rule_type",
	"date_created", "date_modified",
}

var jobTemplateSortSpec = espynahttp.SortSpec{AllowedCols: jobTemplateSortableSQLCols}

// ListJobTemplates lists job template records with optional filters
func (r *PostgresJobTemplateRepository) ListJobTemplates(ctx context.Context, req *pb.ListJobTemplatesRequest) (*pb.ListJobTemplatesResponse, error) {
	if err := espynahttp.ValidateSortColumns(jobTemplateSortSpec, req.GetSort(), "job_template"); err != nil {
		return nil, err
	}

	params := &interfaces.ListParams{}
	if req != nil {
		params.Filters = req.Filters
		params.Search = req.Search
		params.Sort = req.Sort
		params.Pagination = req.Pagination
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

	searchPattern, err := postgresCore.BoundedContainsSearchPattern(req.GetSearch())
	if err != nil {
		return nil, fmt.Errorf("invalid list search: %w", err)
	}

	page, err := postgresCore.BoundedPageRequest(req.GetPagination(), 50)
	if err != nil {
		return nil, fmt.Errorf("invalid list pagination: %w", err)
	}

	// Sort — fail-closed against the per-entity whitelist (A2 guard). Route the
	// caller-supplied sort column through core.BuildOrderBy so an unknown column
	// errors instead of being interpolated verbatim into ORDER BY.
	orderByClause, err := postgresCore.BuildOrderBy(jobTemplateSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}

	// Extract active filter (default to active-only)
	activeFilter := true
	if req.Filters != nil {
		for _, f := range req.Filters.Filters {
			switch f.GetField() {
			case "jt.active", "active":
				if bf := f.GetBooleanFilter(); bf != nil {
					activeFilter = bf.GetValue()
				}
			}
		}
	}

	// A1 (CRITICAL): scope to the caller's workspace. This raw-SQL path bypasses
	// the WorkspaceAwareOperations decorator. job_template carries its own
	// workspace_id column (verified against the baseline schema; the item method
	// already scopes jt.workspace_id). Empty wsID = service-to-service call → no
	// scoping. $5 carries the workspace_id.
	wsID := identity.Must(ctx).WorkspaceID

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
			FROM ` + entityid.JobTemplate + ` jt
			WHERE jt.active = $2
			  AND ($3::text = '' OR jt.workspace_id = $3::text)
			  AND ($1::text IS NULL OR $1::text = '' OR
			       jt.name ILIKE $1 OR
			       jt.description ILIKE $1)
		)
		SELECT e.* FROM enriched e
	`
	sortKeys, err := scopedPageSort(orderByClause)
	if err != nil {
		return nil, err
	}
	set := postgresCore.ScopedPageSet{SQL: query, Args: []any{searchPattern, activeFilter, wsID}, Sort: sortKeys}
	queries, err := postgresCore.ResolveScopedPage(ctx, r.db, set, page)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, queries.PageSQL, queries.PageArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query job template list page data: %w", err)
	}

	var templates []*pb.JobTemplate
	var firstID, lastID string

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
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job template row: %w", err)
		}

		if firstID == "" {
			firstID = id
		}
		lastID = id

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

	if err := rows.Close(); err != nil {
		return nil, err
	}
	var totalCount int64
	if err := r.db.QueryRowContext(ctx, queries.CountSQL, queries.CountArgs...).Scan(&totalCount); err != nil {
		return nil, err
	}

	return &pb.GetJobTemplateListPageDataResponse{
		JobTemplateList: templates,
		Pagination:      scopedPageMetadata(queries.Page, totalCount, firstID, lastID),
		Success:         true,
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

	// A1 (CRITICAL): scope to the caller's workspace. job_template carries its
	// own workspace_id column (verified against the baseline schema), so scope
	// directly on jt.workspace_id. Without this a caller could fetch another
	// tenant's job template by ID. Empty wsID = service-to-service call → no
	// scoping. wsID is appended as $2.
	wsID := identity.Must(ctx).WorkspaceID
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
		FROM ` + entityid.JobTemplate + ` jt
		WHERE jt.id = $1 AND jt.active = true
		  AND ($2::text = '' OR jt.workspace_id = $2::text)
	`

	row := r.db.QueryRowContext(ctx, query, req.JobTemplateId, wsID)

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
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresJobTemplateRepository(dbOps, tableName)
}
