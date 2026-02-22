//go:build postgresql

package product_variant_option

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
	productvariantoptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant_option"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "product_variant_option", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres product_variant_option repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresProductVariantOptionRepository(dbOps, tableName), nil
	})
}

// PostgresProductVariantOptionRepository implements product_variant_option CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_product_variant_option_active ON product_variant_option(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_product_variant_option_variant_id ON product_variant_option(product_variant_id) - FK lookup on product_variant_id
//   - CREATE INDEX idx_product_variant_option_value_id ON product_variant_option(product_option_value_id) - FK lookup on product_option_value_id
//   - CREATE INDEX idx_product_variant_option_date_created ON product_variant_option(date_created DESC) - Default sorting
type PostgresProductVariantOptionRepository struct {
	productvariantoptionpb.UnimplementedProductVariantOptionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresProductVariantOptionRepository creates a new PostgreSQL product variant option repository
func NewPostgresProductVariantOptionRepository(dbOps interfaces.DatabaseOperation, tableName string) productvariantoptionpb.ProductVariantOptionDomainServiceServer {
	if tableName == "" {
		tableName = "product_variant_option" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresProductVariantOptionRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateProductVariantOption creates a new product variant option using common PostgreSQL operations
func (r *PostgresProductVariantOptionRepository) CreateProductVariantOption(ctx context.Context, req *productvariantoptionpb.CreateProductVariantOptionRequest) (*productvariantoptionpb.CreateProductVariantOptionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product variant option data is required")
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
		return nil, fmt.Errorf("failed to create product variant option: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productVariantOption := &productvariantoptionpb.ProductVariantOption{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productVariantOption); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productvariantoptionpb.CreateProductVariantOptionResponse{
		Data: []*productvariantoptionpb.ProductVariantOption{productVariantOption},
	}, nil
}

// ReadProductVariantOption retrieves a product variant option using common PostgreSQL operations
func (r *PostgresProductVariantOptionRepository) ReadProductVariantOption(ctx context.Context, req *productvariantoptionpb.ReadProductVariantOptionRequest) (*productvariantoptionpb.ReadProductVariantOptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product variant option ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product variant option: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productVariantOption := &productvariantoptionpb.ProductVariantOption{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productVariantOption); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productvariantoptionpb.ReadProductVariantOptionResponse{
		Data: []*productvariantoptionpb.ProductVariantOption{productVariantOption},
	}, nil
}

// UpdateProductVariantOption updates a product variant option using common PostgreSQL operations
func (r *PostgresProductVariantOptionRepository) UpdateProductVariantOption(ctx context.Context, req *productvariantoptionpb.UpdateProductVariantOptionRequest) (*productvariantoptionpb.UpdateProductVariantOptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product variant option ID is required")
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
		return nil, fmt.Errorf("failed to update product variant option: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productVariantOption := &productvariantoptionpb.ProductVariantOption{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productVariantOption); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productvariantoptionpb.UpdateProductVariantOptionResponse{
		Data: []*productvariantoptionpb.ProductVariantOption{productVariantOption},
	}, nil
}

// DeleteProductVariantOption deletes a product variant option using common PostgreSQL operations
func (r *PostgresProductVariantOptionRepository) DeleteProductVariantOption(ctx context.Context, req *productvariantoptionpb.DeleteProductVariantOptionRequest) (*productvariantoptionpb.DeleteProductVariantOptionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product variant option ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete product variant option: %w", err)
	}

	return &productvariantoptionpb.DeleteProductVariantOptionResponse{
		Success: true,
	}, nil
}

// ListProductVariantOptions lists product variant options using common PostgreSQL operations
func (r *PostgresProductVariantOptionRepository) ListProductVariantOptions(ctx context.Context, req *productvariantoptionpb.ListProductVariantOptionsRequest) (*productvariantoptionpb.ListProductVariantOptionsResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list product variant options: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var productVariantOptions []*productvariantoptionpb.ProductVariantOption
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal product_variant_option row: %v", err)
			continue
		}

		productVariantOption := &productvariantoptionpb.ProductVariantOption{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productVariantOption); err != nil {
			log.Printf("WARN: protojson unmarshal product_variant_option: %v", err)
			continue
		}
		productVariantOptions = append(productVariantOptions, productVariantOption)
	}

	return &productvariantoptionpb.ListProductVariantOptionsResponse{
		Data: productVariantOptions,
	}, nil
}

// GetProductVariantOptionListPageData retrieves product variant options with advanced filtering, sorting, searching, and pagination using CTE
// This method joins with product_variant and product_option_value tables for enriched data
func (r *PostgresProductVariantOptionRepository) GetProductVariantOptionListPageData(
	ctx context.Context,
	req *productvariantoptionpb.GetProductVariantOptionListPageDataRequest,
) (*productvariantoptionpb.GetProductVariantOptionListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product variant option list page data request is required")
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
	sortField := "pvo.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// CTE Query - Single round-trip with joins for variant SKU and option value label
	query := `
		WITH enriched AS (
			SELECT
				pvo.id,
				pvo.date_created,
				pvo.date_modified,
				pvo.active,
				pvo.product_variant_id,
				pvo.product_option_value_id,
				COALESCE(pv.sku, '') as variant_sku,
				COALESCE(povl.label, '') as option_value_label
			FROM product_variant_option pvo
			LEFT JOIN product_variant pv ON pvo.product_variant_id = pv.id AND pv.active = true
			LEFT JOIN product_option_value povl ON pvo.product_option_value_id = povl.id AND povl.active = true
			WHERE pvo.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       pv.sku ILIKE $1 OR
			       povl.label ILIKE $1)
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
		return nil, fmt.Errorf("failed to query product variant option list page data: %w", err)
	}
	defer rows.Close()

	var productVariantOptions []*productvariantoptionpb.ProductVariantOption
	var totalCount int64

	for rows.Next() {
		var (
			id                   string
			dateCreated          time.Time
			dateModified         time.Time
			active               bool
			productVariantID     string
			productOptionValueID string
			variantSku           string
			optionValueLabel     string
			total                int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&productVariantID,
			&productOptionValueID,
			&variantSku,
			&optionValueLabel,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product variant option row: %w", err)
		}

		totalCount = total

		productVariantOption := &productvariantoptionpb.ProductVariantOption{
			Id:                   id,
			Active:               active,
			ProductVariantId:     productVariantID,
			ProductOptionValueId: productOptionValueID,
		}

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			productVariantOption.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			productVariantOption.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			productVariantOption.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			productVariantOption.DateModifiedString = &dmStr
		}

		// Note: variantSku, optionValueLabel are available but not directly mapped
		// to the ProductVariantOption protobuf. They could be returned via nested refs.

		productVariantOptions = append(productVariantOptions, productVariantOption)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product variant option rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &productvariantoptionpb.GetProductVariantOptionListPageDataResponse{
		ProductVariantOptionList: productVariantOptions,
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

// GetProductVariantOptionItemPageData retrieves a single product variant option with enhanced item page data using CTE
// This method joins with product_variant and product_option_value for enriched data
func (r *PostgresProductVariantOptionRepository) GetProductVariantOptionItemPageData(
	ctx context.Context,
	req *productvariantoptionpb.GetProductVariantOptionItemPageDataRequest,
) (*productvariantoptionpb.GetProductVariantOptionItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product variant option item page data request is required")
	}
	if req.ProductVariantOptionId == "" {
		return nil, fmt.Errorf("product variant option ID is required")
	}

	// CTE Query - Single round-trip with joins
	query := `
		WITH enriched AS (
			SELECT
				pvo.id,
				pvo.date_created,
				pvo.date_modified,
				pvo.active,
				pvo.product_variant_id,
				pvo.product_option_value_id,
				COALESCE(pv.sku, '') as variant_sku,
				COALESCE(povl.label, '') as option_value_label,
				COALESCE(povl.value, '') as option_value_value
			FROM product_variant_option pvo
			LEFT JOIN product_variant pv ON pvo.product_variant_id = pv.id AND pv.active = true
			LEFT JOIN product_option_value povl ON pvo.product_option_value_id = povl.id AND povl.active = true
			WHERE pvo.id = $1 AND pvo.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.ProductVariantOptionId)

	var (
		id                   string
		dateCreated          time.Time
		dateModified         time.Time
		active               bool
		productVariantID     string
		productOptionValueID string
		variantSku           string
		optionValueLabel     string
		optionValueValue     string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&productVariantID,
		&productOptionValueID,
		&variantSku,
		&optionValueLabel,
		&optionValueValue,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("product variant option with ID '%s' not found", req.ProductVariantOptionId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query product variant option item page data: %w", err)
	}

	productVariantOption := &productvariantoptionpb.ProductVariantOption{
		Id:                   id,
		Active:               active,
		ProductVariantId:     productVariantID,
		ProductOptionValueId: productOptionValueID,
	}

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		productVariantOption.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		productVariantOption.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		productVariantOption.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		productVariantOption.DateModifiedString = &dmStr
	}

	// Note: variantSku, optionValueLabel, optionValueValue are available for
	// the nested references but not directly mapped to the ProductVariantOption protobuf.

	return &productvariantoptionpb.GetProductVariantOptionItemPageDataResponse{
		ProductVariantOption: productVariantOption,
		Success:              true,
	}, nil
}

// NewProductVariantOptionRepository creates a new PostgreSQL product variant option repository (old-style constructor)
func NewProductVariantOptionRepository(db *sql.DB, tableName string) productvariantoptionpb.ProductVariantOptionDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresProductVariantOptionRepository(dbOps, tableName)
}
