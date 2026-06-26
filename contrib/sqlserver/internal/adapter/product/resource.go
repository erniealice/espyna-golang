//go:build sqlserver

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	resourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/resource"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.Resource, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver resource repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerResourceRepository(dbOps, tableName), nil
	})
}

// SQLServerResourceRepository implements resource CRUD using SQL Server.
type SQLServerResourceRepository struct {
	resourcepb.UnimplementedResourceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerResourceRepository creates a new SQL Server resource repository.
func NewSQLServerResourceRepository(dbOps interfaces.DatabaseOperation, tableName string) resourcepb.ResourceDomainServiceServer {
	if tableName == "" {
		tableName = "resource"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &SQLServerResourceRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func (r *SQLServerResourceRepository) CreateResource(ctx context.Context, req *resourcepb.CreateResourceRequest) (*resourcepb.CreateResourceResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("resource data is required")
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
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	resource := &resourcepb.Resource{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, resource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &resourcepb.CreateResourceResponse{Data: []*resourcepb.Resource{resource}}, nil
}

func (r *SQLServerResourceRepository) ReadResource(ctx context.Context, req *resourcepb.ReadResourceRequest) (*resourcepb.ReadResourceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("resource ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read resource: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	resource := &resourcepb.Resource{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, resource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &resourcepb.ReadResourceResponse{Data: []*resourcepb.Resource{resource}}, nil
}

func (r *SQLServerResourceRepository) UpdateResource(ctx context.Context, req *resourcepb.UpdateResourceRequest) (*resourcepb.UpdateResourceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("resource ID is required")
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
		return nil, fmt.Errorf("failed to update resource: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	resource := &resourcepb.Resource{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, resource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &resourcepb.UpdateResourceResponse{Data: []*resourcepb.Resource{resource}}, nil
}

func (r *SQLServerResourceRepository) DeleteResource(ctx context.Context, req *resourcepb.DeleteResourceRequest) (*resourcepb.DeleteResourceResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("resource ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete resource: %w", err)
	}
	return &resourcepb.DeleteResourceResponse{Success: true}, nil
}

func (r *SQLServerResourceRepository) ListResources(ctx context.Context, req *resourcepb.ListResourcesRequest) (*resourcepb.ListResourcesResponse, error) {
	params := &interfaces.ListParams{}
	if req != nil {
		params.Filters = req.Filters
		params.Search = req.Search
		params.Sort = req.Sort
		params.Pagination = req.Pagination
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}
	var resources []*resourcepb.Resource
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		resource := &resourcepb.Resource{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, resource); err != nil {
			continue
		}
		resources = append(resources, resource)
	}
	return &resourcepb.ListResourcesResponse{Data: resources}, nil
}

// GetResourceListPageData retrieves resources with filtering, sorting, searching, and pagination.
//
// SQL Server: ILIKE → LIKE; LIMIT/OFFSET → OFFSET/FETCH; $N → @pN; active = 1.
func (r *SQLServerResourceRepository) GetResourceListPageData(
	ctx context.Context,
	req *resourcepb.GetResourceListPageDataRequest,
) (*resourcepb.GetResourceListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get resource list page data request is required")
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

	sortField := "date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}
	allowedSortFields := map[string]bool{
		"date_created": true, "date_modified": true, "name": true,
		"description": true, "product_id": true, "active": true,
	}
	if !allowedSortFields[sortField] {
		sortField = "date_created"
	}

	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

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
			WHERE r.active = 1
			  AND (@p1 = '' OR
			       r.name LIKE @p1 OR
			       r.description LIKE @p1 OR
			       r.product_id LIKE @p1)
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY e.` + sortField + ` ` + sortOrder + `
		OFFSET @p3 ROWS FETCH NEXT @p2 ROWS ONLY
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
			id           string
			name         string
			description  *string
			productId    string
			active       bool
			dateCreated  time.Time
			dateModified time.Time
			total        int64
		)
		if err := rows.Scan(
			&id, &name, &description, &productId, &active, &dateCreated, &dateModified, &total,
		); err != nil {
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

// GetResourceItemPageData retrieves a single resource.
func (r *SQLServerResourceRepository) GetResourceItemPageData(
	ctx context.Context,
	req *resourcepb.GetResourceItemPageDataRequest,
) (*resourcepb.GetResourceItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get resource item page data request is required")
	}
	if req.ResourceId == "" {
		return nil, fmt.Errorf("resource ID is required")
	}
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	query := `
		SELECT TOP 1
			r.id,
			r.name,
			r.description,
			r.product_id,
			r.active,
			r.date_created,
			r.date_modified
		FROM resource r
		WHERE r.id = @p1 AND r.active = 1
	`

	var (
		id           string
		name         string
		description  *string
		productId    string
		active       bool
		dateCreated  time.Time
		dateModified time.Time
	)
	row := r.db.QueryRowContext(ctx, query, req.ResourceId)
	if err := row.Scan(&id, &name, &description, &productId, &active, &dateCreated, &dateModified); err == sql.ErrNoRows {
		return nil, fmt.Errorf("resource with ID '%s' not found", req.ResourceId)
	} else if err != nil {
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
