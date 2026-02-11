//go:build postgresql

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	resourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/resource"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "resource", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres resource repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresResourceRepository(dbOps, tableName), nil
	})
}

// PostgresResourceRepository implements resource CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_resource_active ON resource(active) WHERE active = true - Filter active resources
//   - CREATE INDEX idx_resource_name ON resource(name) - Search on name field
//   - CREATE INDEX idx_resource_name_trgm ON resource USING gin(name gin_trgm_ops) - Fuzzy search support
//   - CREATE INDEX idx_resource_date_created ON resource(date_created DESC) - Default sorting
type PostgresResourceRepository struct {
	resourcepb.UnimplementedResourceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresResourceRepository creates a new PostgreSQL resource repository
func NewPostgresResourceRepository(dbOps interfaces.DatabaseOperation, tableName string) resourcepb.ResourceDomainServiceServer {
	if tableName == "" {
		tableName = "resource" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresResourceRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateResource creates a new resource using common PostgreSQL operations
func (r *PostgresResourceRepository) CreateResource(ctx context.Context, req *resourcepb.CreateResourceRequest) (*resourcepb.CreateResourceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("resource data is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	resource := &resourcepb.Resource{}
	if err := protojson.Unmarshal(resultJSON, resource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &resourcepb.CreateResourceResponse{
		Data: []*resourcepb.Resource{resource},
	}, nil
}

// ReadResource retrieves a resource using common PostgreSQL operations
func (r *PostgresResourceRepository) ReadResource(ctx context.Context, req *resourcepb.ReadResourceRequest) (*resourcepb.ReadResourceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("resource ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read resource: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	resource := &resourcepb.Resource{}
	if err := protojson.Unmarshal(resultJSON, resource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &resourcepb.ReadResourceResponse{
		Data: []*resourcepb.Resource{resource},
	}, nil
}

// UpdateResource updates a resource using common PostgreSQL operations
func (r *PostgresResourceRepository) UpdateResource(ctx context.Context, req *resourcepb.UpdateResourceRequest) (*resourcepb.UpdateResourceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("resource ID is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update resource: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	resource := &resourcepb.Resource{}
	if err := protojson.Unmarshal(resultJSON, resource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &resourcepb.UpdateResourceResponse{
		Data: []*resourcepb.Resource{resource},
	}, nil
}

// DeleteResource deletes a resource using common PostgreSQL operations
func (r *PostgresResourceRepository) DeleteResource(ctx context.Context, req *resourcepb.DeleteResourceRequest) (*resourcepb.DeleteResourceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("resource ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete resource: %w", err)
	}

	return &resourcepb.DeleteResourceResponse{
		Success: true,
	}, nil
}

// ListResources lists resources using common PostgreSQL operations
func (r *PostgresResourceRepository) ListResources(ctx context.Context, req *resourcepb.ListResourcesRequest) (*resourcepb.ListResourcesResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var resources []*resourcepb.Resource
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		resource := &resourcepb.Resource{}
		if err := protojson.Unmarshal(resultJSON, resource); err != nil {
			// Log error and continue with next item
			continue
		}
		resources = append(resources, resource)
	}

	return &resourcepb.ListResourcesResponse{
		Data: resources,
	}, nil
}

// GetResourceListPageData retrieves resources with advanced filtering, sorting, searching, and pagination using CTE
func (r *PostgresResourceRepository) GetResourceListPageData(
	ctx context.Context,
	req *resourcepb.GetResourceListPageDataRequest,
) (*resourcepb.GetResourceListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get resource list page data request is required")
	}

	// Build search condition
	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	// Default pagination values
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

	// Default sort
	sortField := "date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// CTE Query - Simple entity pattern with name/description/product search
	query := `
		WITH enriched AS (
			SELECT
				r.id,
				r.name,
				r.description,
				r.product_id,
				r.active,
				r.date_created,
				r.date_modified
			FROM resource r
			WHERE r.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       r.name ILIKE $1 OR
			       r.description ILIKE $1 OR
			       r.product_id ILIKE $1)
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
		return nil, fmt.Errorf("failed to query resource list page data: %w", err)
	}
	defer rows.Close()

	var resources []*resourcepb.Resource
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			name               string
			description        *string
			productId          string
			active             bool
			dateCreated        time.Time
			dateModified       time.Time
			total              int64
		)

		err := rows.Scan(
			&id,
			&name,
			&description,
			&productId,
			&active,
			&dateCreated,
			&dateModified,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan resource row: %w", err)
		}

		totalCount = total

		resource := &resourcepb.Resource{
			Id:        id,
			Name:      name,
			ProductId: productId,
			Active:    active,
		}

		if description != nil {
			resource.Description = description
		}

		// Handle nullable timestamp fields

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		resource.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		resource.DateCreatedString = &dcStr
	}
		if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		resource.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		resource.DateModifiedString = &dmStr
	}

		resources = append(resources, resource)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating resource rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &resourcepb.GetResourceListPageDataResponse{
		ResourceList: resources,
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

// GetResourceItemPageData retrieves a single resource with enhanced item page data
func (r *PostgresResourceRepository) GetResourceItemPageData(
	ctx context.Context,
	req *resourcepb.GetResourceItemPageDataRequest,
) (*resourcepb.GetResourceItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get resource item page data request is required")
	}
	if req.ResourceId == "" {
		return nil, fmt.Errorf("resource ID is required")
	}

	// Simple query for single resource item
	query := `
		SELECT
			r.id,
			r.name,
			r.description,
			r.product_id,
			r.active,
			r.date_created,
			r.date_modified
		FROM resource r
		WHERE r.id = $1 AND r.active = true
		LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.ResourceId)

	var (
		id                 string
		name               string
		description        *string
		productId          string
		active             bool
		dateCreated        time.Time
		dateModified       time.Time
	)

	err := row.Scan(
		&id,
		&name,
		&description,
		&productId,
		&active,
		&dateCreated,
		&dateModified,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("resource with ID '%s' not found", req.ResourceId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query resource item page data: %w", err)
	}

	resource := &resourcepb.Resource{
		Id:        id,
		Name:      name,
		ProductId: productId,
		Active:    active,
	}

	if description != nil {
		resource.Description = description
	}

	// Handle nullable timestamp fields

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		resource.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		resource.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		resource.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		resource.DateModifiedString = &dmStr
	}

	return &resourcepb.GetResourceItemPageDataResponse{
		Resource: resource,
		Success:  true,
	}, nil
}

// parseResourceTimestamp converts string timestamp to Unix timestamp (milliseconds)
func parseResourceTimestamp(timestampStr string) (int64, error) {
	// Try parsing as RFC3339 format first (most common)
	if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
		return t.UnixMilli(), nil
	}

	// Try other common formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.000Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestampStr); err == nil {
			return t.UnixMilli(), nil
		}
	}

	return 0, fmt.Errorf("unable to parse timestamp: %s", timestampStr)
}

// NewResourceRepository creates a new PostgreSQL resource repository (old-style constructor)
func NewResourceRepository(db *sql.DB, tableName string) resourcepb.ResourceDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresResourceRepository(dbOps, tableName)
}
