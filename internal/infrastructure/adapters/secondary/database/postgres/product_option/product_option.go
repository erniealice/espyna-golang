//go:build postgresql

package product_option

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
	productoptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_option"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "product_option", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres product_option repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresProductOptionRepository(dbOps, tableName), nil
	})
}

// PostgresProductOptionRepository implements product_option CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_product_option_active ON product_option(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_product_option_product_id ON product_option(product_id) - FK lookup on product_id
//   - CREATE INDEX idx_product_option_code ON product_option(code) - Search on code field
//   - CREATE INDEX idx_product_option_date_created ON product_option(date_created DESC) - Default sorting
type PostgresProductOptionRepository struct {
	productoptionpb.UnimplementedProductOptionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresProductOptionRepository creates a new PostgreSQL product option repository
func NewPostgresProductOptionRepository(dbOps interfaces.DatabaseOperation, tableName string) productoptionpb.ProductOptionDomainServiceServer {
	if tableName == "" {
		tableName = "product_option" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresProductOptionRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateProductOption creates a new product option using common PostgreSQL operations
func (r *PostgresProductOptionRepository) CreateProductOption(ctx context.Context, req *productoptionpb.CreateProductOptionRequest) (*productoptionpb.CreateProductOptionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product option data is required")
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
		return nil, fmt.Errorf("failed to create product option: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productOption := &productoptionpb.ProductOption{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productOption); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productoptionpb.CreateProductOptionResponse{
		Data: []*productoptionpb.ProductOption{productOption},
	}, nil
}

// ReadProductOption retrieves a product option using common PostgreSQL operations
func (r *PostgresProductOptionRepository) ReadProductOption(ctx context.Context, req *productoptionpb.ReadProductOptionRequest) (*productoptionpb.ReadProductOptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product option ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product option: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productOption := &productoptionpb.ProductOption{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productOption); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productoptionpb.ReadProductOptionResponse{
		Data: []*productoptionpb.ProductOption{productOption},
	}, nil
}

// UpdateProductOption updates a product option using common PostgreSQL operations
func (r *PostgresProductOptionRepository) UpdateProductOption(ctx context.Context, req *productoptionpb.UpdateProductOptionRequest) (*productoptionpb.UpdateProductOptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product option ID is required")
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
		return nil, fmt.Errorf("failed to update product option: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productOption := &productoptionpb.ProductOption{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productOption); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productoptionpb.UpdateProductOptionResponse{
		Data: []*productoptionpb.ProductOption{productOption},
	}, nil
}

// DeleteProductOption deletes a product option using common PostgreSQL operations
func (r *PostgresProductOptionRepository) DeleteProductOption(ctx context.Context, req *productoptionpb.DeleteProductOptionRequest) (*productoptionpb.DeleteProductOptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product option ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete product option: %w", err)
	}

	return &productoptionpb.DeleteProductOptionResponse{
		Success: true,
	}, nil
}

// ListProductOptions lists product options using common PostgreSQL operations
func (r *PostgresProductOptionRepository) ListProductOptions(ctx context.Context, req *productoptionpb.ListProductOptionsRequest) (*productoptionpb.ListProductOptionsResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list product options: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var productOptions []*productoptionpb.ProductOption
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal product_option row: %v", err)
			continue
		}

		productOption := &productoptionpb.ProductOption{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productOption); err != nil {
			log.Printf("WARN: protojson unmarshal product_option: %v", err)
			continue
		}
		productOptions = append(productOptions, productOption)
	}

	return &productoptionpb.ListProductOptionsResponse{
		Data: productOptions,
	}, nil
}

// GetProductOptionListPageData retrieves product options with advanced filtering, sorting, searching, and pagination using CTE
// This method joins with the product table to include the parent product name
func (r *PostgresProductOptionRepository) GetProductOptionListPageData(
	ctx context.Context,
	req *productoptionpb.GetProductOptionListPageDataRequest,
) (*productoptionpb.GetProductOptionListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product option list page data request is required")
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
	sortField := "po.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// CTE Query - Single round-trip with product join for parent product name
	query := `
		WITH enriched AS (
			SELECT
				po.id,
				po.date_created,
				po.date_modified,
				po.active,
				po.product_id,
				po.name,
				po.code,
				po.data_type,
				po.sort_order,
				po.min_value,
				po.max_value,
				COALESCE(p.name, '') as product_name
			FROM product_option po
			LEFT JOIN product p ON po.product_id = p.id AND p.active = true
			WHERE po.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       po.name ILIKE $1 OR
			       po.code ILIKE $1 OR
			       p.name ILIKE $1)
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
		return nil, fmt.Errorf("failed to query product option list page data: %w", err)
	}
	defer rows.Close()

	var productOptions []*productoptionpb.ProductOption
	var totalCount int64

	for rows.Next() {
		var (
			id           string
			dateCreated  time.Time
			dateModified time.Time
			active       bool
			productID    string
			name         string
			code         string
			dataType     string
			sortOrder    int32
			minValue     *float64
			maxValue     *float64
			productName  string
			total        int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&productID,
			&name,
			&code,
			&dataType,
			&sortOrder,
			&minValue,
			&maxValue,
			&productName,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product option row: %w", err)
		}

		totalCount = total

		productOption := &productoptionpb.ProductOption{
			Id:        id,
			Active:    active,
			ProductId: productID,
			Name:      name,
			Code:      code,
			DataType:  dataType,
			SortOrder: sortOrder,
			MinValue:  minValue,
			MaxValue:  maxValue,
		}

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			productOption.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			productOption.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			productOption.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			productOption.DateModifiedString = &dmStr
		}

		productOptions = append(productOptions, productOption)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product option rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &productoptionpb.GetProductOptionListPageDataResponse{
		ProductOptionList: productOptions,
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

// GetProductOptionItemPageData retrieves a single product option with enhanced item page data using CTE
// This method joins with the product table for the parent product reference
func (r *PostgresProductOptionRepository) GetProductOptionItemPageData(
	ctx context.Context,
	req *productoptionpb.GetProductOptionItemPageDataRequest,
) (*productoptionpb.GetProductOptionItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product option item page data request is required")
	}
	if req.ProductOptionId == "" {
		return nil, fmt.Errorf("product option ID is required")
	}

	// CTE Query - Single round-trip with product join
	query := `
		WITH enriched AS (
			SELECT
				po.id,
				po.date_created,
				po.date_modified,
				po.active,
				po.product_id,
				po.name,
				po.code,
				po.data_type,
				po.sort_order,
				po.min_value,
				po.max_value,
				COALESCE(p.name, '') as product_name
			FROM product_option po
			LEFT JOIN product p ON po.product_id = p.id AND p.active = true
			WHERE po.id = $1 AND po.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.ProductOptionId)

	var (
		id           string
		dateCreated  time.Time
		dateModified time.Time
		active       bool
		productID    string
		name         string
		code         string
		dataType     string
		sortOrderVal int32
		minValue     *float64
		maxValue     *float64
		productName  string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&productID,
		&name,
		&code,
		&dataType,
		&sortOrderVal,
		&minValue,
		&maxValue,
		&productName,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("product option with ID '%s' not found", req.ProductOptionId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query product option item page data: %w", err)
	}

	productOption := &productoptionpb.ProductOption{
		Id:        id,
		Active:    active,
		ProductId: productID,
		Name:      name,
		Code:      code,
		DataType:  dataType,
		SortOrder: sortOrderVal,
		MinValue:  minValue,
		MaxValue:  maxValue,
	}

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		productOption.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		productOption.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		productOption.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		productOption.DateModifiedString = &dmStr
	}

	return &productoptionpb.GetProductOptionItemPageDataResponse{
		ProductOption: productOption,
		Success:       true,
	}, nil
}

// NewProductOptionRepository creates a new PostgreSQL product option repository (old-style constructor)
func NewProductOptionRepository(db *sql.DB, tableName string) productoptionpb.ProductOptionDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresProductOptionRepository(dbOps, tableName)
}
