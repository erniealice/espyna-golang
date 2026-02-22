//go:build postgresql

package product_option_value

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	productoptionvaluepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_option_value"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "product_option_value", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres product_option_value repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresProductOptionValueRepository(dbOps, tableName), nil
	})
}

// PostgresProductOptionValueRepository implements product_option_value CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_product_option_value_active ON product_option_value(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_product_option_value_product_option_id ON product_option_value(product_option_id) - FK lookup on product_option_id
//   - CREATE INDEX idx_product_option_value_label ON product_option_value(label) - Search on label field
//   - CREATE INDEX idx_product_option_value_date_created ON product_option_value(date_created DESC) - Default sorting
type PostgresProductOptionValueRepository struct {
	productoptionvaluepb.UnimplementedProductOptionValueDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresProductOptionValueRepository creates a new PostgreSQL product option value repository
func NewPostgresProductOptionValueRepository(dbOps interfaces.DatabaseOperation, tableName string) productoptionvaluepb.ProductOptionValueDomainServiceServer {
	if tableName == "" {
		tableName = "product_option_value" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresProductOptionValueRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateProductOptionValue creates a new product option value using common PostgreSQL operations
func (r *PostgresProductOptionValueRepository) CreateProductOptionValue(ctx context.Context, req *productoptionvaluepb.CreateProductOptionValueRequest) (*productoptionvaluepb.CreateProductOptionValueResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product option value data is required")
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
		return nil, fmt.Errorf("failed to create product option value: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productOptionValue := &productoptionvaluepb.ProductOptionValue{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productOptionValue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productoptionvaluepb.CreateProductOptionValueResponse{
		Data: []*productoptionvaluepb.ProductOptionValue{productOptionValue},
	}, nil
}

// ReadProductOptionValue retrieves a product option value using common PostgreSQL operations
func (r *PostgresProductOptionValueRepository) ReadProductOptionValue(ctx context.Context, req *productoptionvaluepb.ReadProductOptionValueRequest) (*productoptionvaluepb.ReadProductOptionValueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product option value ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product option value: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productOptionValue := &productoptionvaluepb.ProductOptionValue{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productOptionValue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productoptionvaluepb.ReadProductOptionValueResponse{
		Data: []*productoptionvaluepb.ProductOptionValue{productOptionValue},
	}, nil
}

// UpdateProductOptionValue updates a product option value using common PostgreSQL operations
func (r *PostgresProductOptionValueRepository) UpdateProductOptionValue(ctx context.Context, req *productoptionvaluepb.UpdateProductOptionValueRequest) (*productoptionvaluepb.UpdateProductOptionValueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product option value ID is required")
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
		return nil, fmt.Errorf("failed to update product option value: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productOptionValue := &productoptionvaluepb.ProductOptionValue{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productOptionValue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productoptionvaluepb.UpdateProductOptionValueResponse{
		Data: []*productoptionvaluepb.ProductOptionValue{productOptionValue},
	}, nil
}

// DeleteProductOptionValue deletes a product option value using common PostgreSQL operations
func (r *PostgresProductOptionValueRepository) DeleteProductOptionValue(ctx context.Context, req *productoptionvaluepb.DeleteProductOptionValueRequest) (*productoptionvaluepb.DeleteProductOptionValueResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product option value ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete product option value: %w", err)
	}

	return &productoptionvaluepb.DeleteProductOptionValueResponse{
		Success: true,
	}, nil
}

// ListProductOptionValues lists product option values using common PostgreSQL operations
func (r *PostgresProductOptionValueRepository) ListProductOptionValues(ctx context.Context, req *productoptionvaluepb.ListProductOptionValuesRequest) (*productoptionvaluepb.ListProductOptionValuesResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list product option values: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var productOptionValues []*productoptionvaluepb.ProductOptionValue
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal product_option_value row: %v", err)
			continue
		}

		productOptionValue := &productoptionvaluepb.ProductOptionValue{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productOptionValue); err != nil {
			log.Printf("WARN: protojson unmarshal product_option_value: %v", err)
			continue
		}
		productOptionValues = append(productOptionValues, productOptionValue)
	}

	return &productoptionvaluepb.ListProductOptionValuesResponse{
		Data: productOptionValues,
	}, nil
}

// GetProductOptionValueListPageData retrieves product option values with advanced filtering, sorting, searching, and pagination using CTE
// This method joins with the product_option table to include the parent option name
func (r *PostgresProductOptionValueRepository) GetProductOptionValueListPageData(
	ctx context.Context,
	req *productoptionvaluepb.GetProductOptionValueListPageDataRequest,
) (*productoptionvaluepb.GetProductOptionValueListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product option value list page data request is required")
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
	sortField := "pov.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// CTE Query - Single round-trip with product_option join for parent option name
	query := `
		WITH enriched AS (
			SELECT
				pov.id,
				pov.date_created,
				pov.date_modified,
				pov.active,
				pov.product_option_id,
				pov.label,
				pov.value,
				pov.sort_order,
				COALESCE(pov.metadata::text, '{}') as metadata,
				COALESCE(po.name, '') as option_name
			FROM product_option_value pov
			LEFT JOIN product_option po ON pov.product_option_id = po.id AND po.active = true
			WHERE pov.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       pov.label ILIKE $1 OR
			       pov.value ILIKE $1 OR
			       po.name ILIKE $1)
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
		return nil, fmt.Errorf("failed to query product option value list page data: %w", err)
	}
	defer rows.Close()

	var productOptionValues []*productoptionvaluepb.ProductOptionValue
	var totalCount int64

	for rows.Next() {
		var (
			id              string
			dateCreated     time.Time
			dateModified    time.Time
			active          bool
			productOptionID string
			label           string
			value           string
			sortOrderVal    int32
			metadataStr     string
			optionName      string
			total           int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&productOptionID,
			&label,
			&value,
			&sortOrderVal,
			&metadataStr,
			&optionName,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product option value row: %w", err)
		}

		totalCount = total

		productOptionValue := &productoptionvaluepb.ProductOptionValue{
			Id:              id,
			Active:          active,
			ProductOptionId: productOptionID,
			Label:           label,
			Value:           value,
			SortOrder:       sortOrderVal,
		}

		// Parse metadata JSONB field
		if metadataStr != "" && metadataStr != "{}" {
			metadata := &structpb.Struct{}
			if err := protojson.Unmarshal([]byte(metadataStr), metadata); err == nil {
				productOptionValue.Metadata = metadata
			}
		}

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			productOptionValue.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			productOptionValue.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			productOptionValue.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			productOptionValue.DateModifiedString = &dmStr
		}

		productOptionValues = append(productOptionValues, productOptionValue)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product option value rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &productoptionvaluepb.GetProductOptionValueListPageDataResponse{
		ProductOptionValueList: productOptionValues,
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

// GetProductOptionValueItemPageData retrieves a single product option value with enhanced item page data using CTE
// This method joins with the product_option table for the parent option reference
func (r *PostgresProductOptionValueRepository) GetProductOptionValueItemPageData(
	ctx context.Context,
	req *productoptionvaluepb.GetProductOptionValueItemPageDataRequest,
) (*productoptionvaluepb.GetProductOptionValueItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product option value item page data request is required")
	}
	if req.ProductOptionValueId == "" {
		return nil, fmt.Errorf("product option value ID is required")
	}

	// CTE Query - Single round-trip with product_option join
	query := `
		WITH enriched AS (
			SELECT
				pov.id,
				pov.date_created,
				pov.date_modified,
				pov.active,
				pov.product_option_id,
				pov.label,
				pov.value,
				pov.sort_order,
				COALESCE(pov.metadata::text, '{}') as metadata,
				COALESCE(po.name, '') as option_name,
				COALESCE(po.code, '') as option_code
			FROM product_option_value pov
			LEFT JOIN product_option po ON pov.product_option_id = po.id AND po.active = true
			WHERE pov.id = $1 AND pov.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.ProductOptionValueId)

	var (
		id              string
		dateCreated     time.Time
		dateModified    time.Time
		active          bool
		productOptionID string
		label           string
		value           string
		sortOrderVal    int32
		metadataStr     string
		optionName      string
		optionCode      string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&productOptionID,
		&label,
		&value,
		&sortOrderVal,
		&metadataStr,
		&optionName,
		&optionCode,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("product option value with ID '%s' not found", req.ProductOptionValueId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query product option value item page data: %w", err)
	}

	productOptionValue := &productoptionvaluepb.ProductOptionValue{
		Id:              id,
		Active:          active,
		ProductOptionId: productOptionID,
		Label:           label,
		Value:           value,
		SortOrder:       sortOrderVal,
	}

	// Parse metadata JSONB field
	if metadataStr != "" && metadataStr != "{}" {
		metadata := &structpb.Struct{}
		if err := protojson.Unmarshal([]byte(metadataStr), metadata); err == nil {
			productOptionValue.Metadata = metadata
		}
	}

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		productOptionValue.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		productOptionValue.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		productOptionValue.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		productOptionValue.DateModifiedString = &dmStr
	}

	// Note: optionName, optionCode are available for the product_option reference
	// but not directly mapped to the ProductOptionValue protobuf.

	return &productoptionvaluepb.GetProductOptionValueItemPageDataResponse{
		ProductOptionValue: productOptionValue,
		Success:            true,
	}, nil
}

// NewProductOptionValueRepository creates a new PostgreSQL product option value repository (old-style constructor)
func NewProductOptionValueRepository(db *sql.DB, tableName string) productoptionvaluepb.ProductOptionValueDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresProductOptionValueRepository(dbOps, tableName)
}
