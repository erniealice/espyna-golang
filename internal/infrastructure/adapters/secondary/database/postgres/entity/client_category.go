//go:build postgres

package entity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/operations"
	postgresCore "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/postgres/core"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	clientcategorypb "leapfor.xyz/esqyma/golang/v1/domain/entity/client_category"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "client_category", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres client_category repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresClientCategoryRepository(dbOps, tableName), nil
	})
}

// PostgresClientCategoryRepository implements client_category CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_client_category_active ON client_category(active) WHERE active = true - Filter active categories
//   - CREATE INDEX idx_client_category_name ON client_category(name) - Search on name field
//   - CREATE INDEX idx_client_category_name_trgm ON client_category USING gin(name gin_trgm_ops) - Fuzzy search support
//   - CREATE INDEX idx_client_category_date_created ON client_category(date_created DESC) - Default sorting
type PostgresClientCategoryRepository struct {
	clientcategorypb.UnimplementedClientCategoryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresClientCategoryRepository creates a new PostgreSQL client_category repository
func NewPostgresClientCategoryRepository(dbOps interfaces.DatabaseOperation, tableName string) clientcategorypb.ClientCategoryDomainServiceServer {
	if tableName == "" {
		tableName = "client_category" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresClientCategoryRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateClientCategory creates a new client_category using common PostgreSQL operations
func (r *PostgresClientCategoryRepository) CreateClientCategory(ctx context.Context, req *clientcategorypb.CreateClientCategoryRequest) (*clientcategorypb.CreateClientCategoryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("client_category data is required")
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
		return nil, fmt.Errorf("failed to create client_category: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	clientCategory := &clientcategorypb.ClientCategory{}
	if err := protojson.Unmarshal(resultJSON, clientCategory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &clientcategorypb.CreateClientCategoryResponse{
		Data: []*clientcategorypb.ClientCategory{clientCategory},
	}, nil
}

// ReadClientCategory retrieves a client_category using common PostgreSQL operations
func (r *PostgresClientCategoryRepository) ReadClientCategory(ctx context.Context, req *clientcategorypb.ReadClientCategoryRequest) (*clientcategorypb.ReadClientCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client_category ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read client_category: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	clientCategory := &clientcategorypb.ClientCategory{}
	if err := protojson.Unmarshal(resultJSON, clientCategory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &clientcategorypb.ReadClientCategoryResponse{
		Data: []*clientcategorypb.ClientCategory{clientCategory},
	}, nil
}

// UpdateClientCategory updates a client_category using common PostgreSQL operations
func (r *PostgresClientCategoryRepository) UpdateClientCategory(ctx context.Context, req *clientcategorypb.UpdateClientCategoryRequest) (*clientcategorypb.UpdateClientCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client_category ID is required")
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
		return nil, fmt.Errorf("failed to update client_category: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	clientCategory := &clientcategorypb.ClientCategory{}
	if err := protojson.Unmarshal(resultJSON, clientCategory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &clientcategorypb.UpdateClientCategoryResponse{
		Data: []*clientcategorypb.ClientCategory{clientCategory},
	}, nil
}

// DeleteClientCategory deletes a client_category using common PostgreSQL operations
func (r *PostgresClientCategoryRepository) DeleteClientCategory(ctx context.Context, req *clientcategorypb.DeleteClientCategoryRequest) (*clientcategorypb.DeleteClientCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("client_category ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete client_category: %w", err)
	}

	return &clientcategorypb.DeleteClientCategoryResponse{
		Success: true,
	}, nil
}

// ListClientCategories lists client_categories using common PostgreSQL operations
func (r *PostgresClientCategoryRepository) ListClientCategories(ctx context.Context, req *clientcategorypb.ListClientCategoriesRequest) (*clientcategorypb.ListClientCategoriesResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list client_categories: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var clientCategories []*clientcategorypb.ClientCategory
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		clientCategory := &clientcategorypb.ClientCategory{}
		if err := protojson.Unmarshal(resultJSON, clientCategory); err != nil {
			// Log error and continue with next item
			continue
		}
		clientCategories = append(clientCategories, clientCategory)
	}

	return &clientcategorypb.ListClientCategoriesResponse{
		Data: clientCategories,
	}, nil
}

// GetClientCategoryListPageData retrieves client_categories with advanced filtering, sorting, searching, and pagination using CTE
func (r *PostgresClientCategoryRepository) GetClientCategoryListPageData(
	ctx context.Context,
	req *clientcategorypb.GetClientCategoryListPageDataRequest,
) (*clientcategorypb.GetClientCategoryListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get client_category list page data request is required")
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
		// Handle offset pagination
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

	// CTE Query - Single round-trip with filtering and pagination
	// Performance Notes:
	// - INDEX RECOMMENDATION: Create index on client_category.active for filtering active records
	// - INDEX RECOMMENDATION: Create index on client_category.name for search performance
	// - INDEX RECOMMENDATION: Create index on client_category.date_created for default sorting
	query := `
		WITH enriched AS (
			SELECT
				cc.id,
				cc.name,
				cc.description,
				cc.active,
				cc.date_created,
				cc.date_created_string,
				cc.date_modified,
				cc.date_modified_string
			FROM client_category cc
			WHERE cc.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
				   cc.name ILIKE $1 OR
				   cc.description ILIKE $1)
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
		return nil, fmt.Errorf("failed to query client_category list page data: %w", err)
	}
	defer rows.Close()

	var clientCategories []*clientcategorypb.ClientCategory
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			name               string
			description        *string
			active             bool
			dateCreated        *string
			dateCreatedString  *string
			dateModified       *string
			dateModifiedString *string
			total              int64
		)

		err := rows.Scan(
			&id,
			&name,
			&description,
			&active,
			&dateCreated,
			&dateCreatedString,
			&dateModified,
			&dateModifiedString,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan client_category row: %w", err)
		}

		totalCount = total

		cat := &commonpb.Category{
			Name: name,
		}
		if description != nil {
			cat.Description = *description
		}

		clientCategory := &clientcategorypb.ClientCategory{
			Id:       id,
			Category: cat,
			Active:   active,
		}

		// Handle nullable timestamp fields
		if dateCreatedString != nil {
			clientCategory.DateCreatedString = dateCreatedString
		}
		if dateModifiedString != nil {
			clientCategory.DateModifiedString = dateModifiedString
		}

		// Parse timestamps if provided
		if dateCreated != nil && *dateCreated != "" {
			if ts, err := operations.ParseTimestamp(*dateCreated); err == nil {
				clientCategory.DateCreated = &ts
			}
		}
		if dateModified != nil && *dateModified != "" {
			if ts, err := operations.ParseTimestamp(*dateModified); err == nil {
				clientCategory.DateModified = &ts
			}
		}

		clientCategories = append(clientCategories, clientCategory)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating client_category rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &clientcategorypb.GetClientCategoryListPageDataResponse{
		ClientCategoryList: clientCategories,
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

// GetClientCategoryItemPageData retrieves a single client_category with enhanced item page data
func (r *PostgresClientCategoryRepository) GetClientCategoryItemPageData(
	ctx context.Context,
	req *clientcategorypb.GetClientCategoryItemPageDataRequest,
) (*clientcategorypb.GetClientCategoryItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get client_category item page data request is required")
	}
	if req.ClientCategoryId == "" {
		return nil, fmt.Errorf("client_category ID is required")
	}

	// Simple query for single client_category item
	query := `
		SELECT
			cc.id,
			cc.name,
			cc.description,
			cc.active,
			cc.date_created,
			cc.date_created_string,
			cc.date_modified,
			cc.date_modified_string
		FROM client_category cc
		WHERE cc.id = $1 AND cc.active = true
		LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.ClientCategoryId)

	var (
		id                 string
		name               string
		description        *string
		active             bool
		dateCreated        *string
		dateCreatedString  *string
		dateModified       *string
		dateModifiedString *string
	)

	err := row.Scan(
		&id,
		&name,
		&description,
		&active,
		&dateCreated,
		&dateCreatedString,
		&dateModified,
		&dateModifiedString,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("client_category with ID '%s' not found", req.ClientCategoryId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query client_category item page data: %w", err)
	}

	cat := &commonpb.Category{
		Name: name,
	}
	if description != nil {
		cat.Description = *description
	}

	clientCategory := &clientcategorypb.ClientCategory{
		Id:       id,
		Category: cat,
		Active:   active,
	}

	// Handle nullable timestamp fields
	if dateCreatedString != nil {
		clientCategory.DateCreatedString = dateCreatedString
	}
	if dateModifiedString != nil {
		clientCategory.DateModifiedString = dateModifiedString
	}

	// Parse timestamps if provided
	if dateCreated != nil && *dateCreated != "" {
		if ts, err := operations.ParseTimestamp(*dateCreated); err == nil {
			clientCategory.DateCreated = &ts
		}
	}
	if dateModified != nil && *dateModified != "" {
		if ts, err := operations.ParseTimestamp(*dateModified); err == nil {
			clientCategory.DateModified = &ts
		}
	}

	return &clientcategorypb.GetClientCategoryItemPageDataResponse{
		ClientCategory: clientCategory,
		Success:        true,
	}, nil
}

// NewClientCategoryRepository creates a new PostgreSQL client_category repository (old-style constructor)
func NewClientCategoryRepository(db *sql.DB, tableName string) clientcategorypb.ClientCategoryDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresClientCategoryRepository(dbOps, tableName)
}
