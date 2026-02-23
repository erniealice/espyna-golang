//go:build postgresql

package revenue_category

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	revenuecategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_category"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "revenue_category", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres revenue_category repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresRevenueCategoryRepository(dbOps, tableName), nil
	})
}

// PostgresRevenueCategoryRepository implements revenue_category CRUD operations using PostgreSQL
type PostgresRevenueCategoryRepository struct {
	revenuecategorypb.UnimplementedRevenueCategoryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresRevenueCategoryRepository creates a new PostgreSQL revenue category repository
func NewPostgresRevenueCategoryRepository(dbOps interfaces.DatabaseOperation, tableName string) revenuecategorypb.RevenueCategoryDomainServiceServer {
	if tableName == "" {
		tableName = "revenue_category"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresRevenueCategoryRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateRevenueCategory creates a new revenue category
func (r *PostgresRevenueCategoryRepository) CreateRevenueCategory(ctx context.Context, req *revenuecategorypb.CreateRevenueCategoryRequest) (*revenuecategorypb.CreateRevenueCategoryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("revenue category data is required")
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
		return nil, fmt.Errorf("failed to create revenue category: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	category := &revenuecategorypb.RevenueCategory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, category); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenuecategorypb.CreateRevenueCategoryResponse{
		Data: []*revenuecategorypb.RevenueCategory{category},
	}, nil
}

// ReadRevenueCategory retrieves a revenue category by ID
func (r *PostgresRevenueCategoryRepository) ReadRevenueCategory(ctx context.Context, req *revenuecategorypb.ReadRevenueCategoryRequest) (*revenuecategorypb.ReadRevenueCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue category ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read revenue category: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	category := &revenuecategorypb.RevenueCategory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, category); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenuecategorypb.ReadRevenueCategoryResponse{
		Data: []*revenuecategorypb.RevenueCategory{category},
	}, nil
}

// UpdateRevenueCategory updates a revenue category
func (r *PostgresRevenueCategoryRepository) UpdateRevenueCategory(ctx context.Context, req *revenuecategorypb.UpdateRevenueCategoryRequest) (*revenuecategorypb.UpdateRevenueCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue category ID is required")
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
		return nil, fmt.Errorf("failed to update revenue category: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	category := &revenuecategorypb.RevenueCategory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, category); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &revenuecategorypb.UpdateRevenueCategoryResponse{
		Data: []*revenuecategorypb.RevenueCategory{category},
	}, nil
}

// DeleteRevenueCategory deletes a revenue category (soft delete)
func (r *PostgresRevenueCategoryRepository) DeleteRevenueCategory(ctx context.Context, req *revenuecategorypb.DeleteRevenueCategoryRequest) (*revenuecategorypb.DeleteRevenueCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("revenue category ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete revenue category: %w", err)
	}

	return &revenuecategorypb.DeleteRevenueCategoryResponse{
		Success: true,
	}, nil
}

// ListRevenueCategories lists revenue categories with optional filters
func (r *PostgresRevenueCategoryRepository) ListRevenueCategories(ctx context.Context, req *revenuecategorypb.ListRevenueCategoriesRequest) (*revenuecategorypb.ListRevenueCategoriesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list revenue categories: %w", err)
	}

	var categories []*revenuecategorypb.RevenueCategory
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal revenue_category row: %v", err)
			continue
		}

		category := &revenuecategorypb.RevenueCategory{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, category); err != nil {
			log.Printf("WARN: protojson unmarshal revenue_category: %v", err)
			continue
		}
		categories = append(categories, category)
	}

	return &revenuecategorypb.ListRevenueCategoriesResponse{
		Data: categories,
	}, nil
}

// GetRevenueCategoryListPageData retrieves revenue categories with pagination, sorting, and search using CTE
func (r *PostgresRevenueCategoryRepository) GetRevenueCategoryListPageData(
	ctx context.Context,
	req *revenuecategorypb.GetRevenueCategoryListPageDataRequest,
) (*revenuecategorypb.GetRevenueCategoryListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get revenue category list page data request is required")
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

	sortField := "rc.date_created"
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
				rc.id,
				rc.date_created,
				rc.date_modified,
				rc.active,
				rc.name,
				rc.code,
				rc.description,
				rc.parent_category_id
			FROM revenue_category rc
			WHERE rc.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       rc.name ILIKE $1 OR
			       rc.code ILIKE $1 OR
			       rc.description ILIKE $1)
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
		return nil, fmt.Errorf("failed to query revenue category list page data: %w", err)
	}
	defer rows.Close()

	var categories []*revenuecategorypb.RevenueCategory
	var totalCount int64

	for rows.Next() {
		var (
			id               string
			dateCreated      time.Time
			dateModified     time.Time
			active           bool
			name             string
			code             string
			description      *string
			parentCategoryID *string
			total            int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&name,
			&code,
			&description,
			&parentCategoryID,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan revenue category row: %w", err)
		}

		totalCount = total

		category := &revenuecategorypb.RevenueCategory{
			Id:               id,
			Active:           active,
			Name:             name,
			Code:             code,
			Description:      description,
			ParentCategoryId: parentCategoryID,
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			category.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			category.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			category.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			category.DateModifiedString = &dmStr
		}

		categories = append(categories, category)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating revenue category rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &revenuecategorypb.GetRevenueCategoryListPageDataResponse{
		RevenueCategoryList: categories,
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

// GetRevenueCategoryItemPageData retrieves a single revenue category with enriched data
func (r *PostgresRevenueCategoryRepository) GetRevenueCategoryItemPageData(
	ctx context.Context,
	req *revenuecategorypb.GetRevenueCategoryItemPageDataRequest,
) (*revenuecategorypb.GetRevenueCategoryItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get revenue category item page data request is required")
	}
	if req.RevenueCategoryId == "" {
		return nil, fmt.Errorf("revenue category ID is required")
	}

	query := `
		SELECT
			rc.id,
			rc.date_created,
			rc.date_modified,
			rc.active,
			rc.name,
			rc.code,
			rc.description,
			rc.parent_category_id
		FROM revenue_category rc
		WHERE rc.id = $1 AND rc.active = true
		LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.RevenueCategoryId)

	var (
		id               string
		dateCreated      time.Time
		dateModified     time.Time
		active           bool
		name             string
		code             string
		description      *string
		parentCategoryID *string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&name,
		&code,
		&description,
		&parentCategoryID,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("revenue category with ID '%s' not found", req.RevenueCategoryId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query revenue category item page data: %w", err)
	}

	category := &revenuecategorypb.RevenueCategory{
		Id:               id,
		Active:           active,
		Name:             name,
		Code:             code,
		Description:      description,
		ParentCategoryId: parentCategoryID,
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		category.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		category.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		category.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		category.DateModifiedString = &dmStr
	}

	return &revenuecategorypb.GetRevenueCategoryItemPageDataResponse{
		RevenueCategory: category,
		Success:         true,
	}, nil
}

// NewRevenueCategoryRepository creates a new PostgreSQL revenue category repository (old-style constructor)
func NewRevenueCategoryRepository(db *sql.DB, tableName string) revenuecategorypb.RevenueCategoryDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresRevenueCategoryRepository(dbOps, tableName)
}
