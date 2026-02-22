//go:build postgresql

package product_variant

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
	productvariantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "product_variant", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres product_variant repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresProductVariantRepository(dbOps, tableName), nil
	})
}

// PostgresProductVariantRepository implements product_variant CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_product_variant_active ON product_variant(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_product_variant_product_id ON product_variant(product_id) - FK lookup on product_id
//   - CREATE INDEX idx_product_variant_sku ON product_variant(sku) - Search on sku field
//   - CREATE INDEX idx_product_variant_date_created ON product_variant(date_created DESC) - Default sorting
type PostgresProductVariantRepository struct {
	productvariantpb.UnimplementedProductVariantDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresProductVariantRepository creates a new PostgreSQL product variant repository
func NewPostgresProductVariantRepository(dbOps interfaces.DatabaseOperation, tableName string) productvariantpb.ProductVariantDomainServiceServer {
	if tableName == "" {
		tableName = "product_variant" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresProductVariantRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateProductVariant creates a new product variant using common PostgreSQL operations
func (r *PostgresProductVariantRepository) CreateProductVariant(ctx context.Context, req *productvariantpb.CreateProductVariantRequest) (*productvariantpb.CreateProductVariantResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product variant data is required")
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
		return nil, fmt.Errorf("failed to create product variant: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productVariant := &productvariantpb.ProductVariant{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productVariant); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productvariantpb.CreateProductVariantResponse{
		Data: []*productvariantpb.ProductVariant{productVariant},
	}, nil
}

// ReadProductVariant retrieves a product variant using common PostgreSQL operations
func (r *PostgresProductVariantRepository) ReadProductVariant(ctx context.Context, req *productvariantpb.ReadProductVariantRequest) (*productvariantpb.ReadProductVariantResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product variant ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product variant: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productVariant := &productvariantpb.ProductVariant{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productVariant); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productvariantpb.ReadProductVariantResponse{
		Data: []*productvariantpb.ProductVariant{productVariant},
	}, nil
}

// UpdateProductVariant updates a product variant using common PostgreSQL operations
func (r *PostgresProductVariantRepository) UpdateProductVariant(ctx context.Context, req *productvariantpb.UpdateProductVariantRequest) (*productvariantpb.UpdateProductVariantResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product variant ID is required")
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
		return nil, fmt.Errorf("failed to update product variant: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	productVariant := &productvariantpb.ProductVariant{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productVariant); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productvariantpb.UpdateProductVariantResponse{
		Data: []*productvariantpb.ProductVariant{productVariant},
	}, nil
}

// DeleteProductVariant deletes a product variant using common PostgreSQL operations
func (r *PostgresProductVariantRepository) DeleteProductVariant(ctx context.Context, req *productvariantpb.DeleteProductVariantRequest) (*productvariantpb.DeleteProductVariantResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product variant ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete product variant: %w", err)
	}

	return &productvariantpb.DeleteProductVariantResponse{
		Success: true,
	}, nil
}

// ListProductVariants lists product variants using common PostgreSQL operations
func (r *PostgresProductVariantRepository) ListProductVariants(ctx context.Context, req *productvariantpb.ListProductVariantsRequest) (*productvariantpb.ListProductVariantsResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list product variants: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var productVariants []*productvariantpb.ProductVariant
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal product_variant row: %v", err)
			continue
		}

		productVariant := &productvariantpb.ProductVariant{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, productVariant); err != nil {
			log.Printf("WARN: protojson unmarshal product_variant: %v", err)
			continue
		}
		productVariants = append(productVariants, productVariant)
	}

	return &productvariantpb.ListProductVariantsResponse{
		Data: productVariants,
	}, nil
}

// GetProductVariantListPageData retrieves product variants with advanced filtering, sorting, searching, and pagination using CTE
// This method joins with the product table to include the parent product name
func (r *PostgresProductVariantRepository) GetProductVariantListPageData(
	ctx context.Context,
	req *productvariantpb.GetProductVariantListPageDataRequest,
) (*productvariantpb.GetProductVariantListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product variant list page data request is required")
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
	sortField := "pv.date_created"
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
				pv.id,
				pv.date_created,
				pv.date_modified,
				pv.active,
				pv.product_id,
				pv.sku,
				pv.price_override,
				COALESCE(p.name, '') as product_name
			FROM product_variant pv
			LEFT JOIN product p ON pv.product_id = p.id AND p.active = true
			WHERE pv.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       pv.sku ILIKE $1 OR
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
		return nil, fmt.Errorf("failed to query product variant list page data: %w", err)
	}
	defer rows.Close()

	var productVariants []*productvariantpb.ProductVariant
	var totalCount int64

	for rows.Next() {
		var (
			id            string
			dateCreated   time.Time
			dateModified  time.Time
			active        bool
			productID     string
			sku           string
			priceOverride float64
			productName   string
			total         int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&productID,
			&sku,
			&priceOverride,
			&productName,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product variant row: %w", err)
		}

		totalCount = total

		productVariant := &productvariantpb.ProductVariant{
			Id:            id,
			Active:        active,
			ProductId:     productID,
			Sku:           sku,
			PriceOverride: priceOverride,
		}

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			productVariant.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			productVariant.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			productVariant.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			productVariant.DateModifiedString = &dmStr
		}

		// Note: productName is available but not mapped to the ProductVariant protobuf
		// in this list view. The product reference could be populated if needed.

		productVariants = append(productVariants, productVariant)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product variant rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &productvariantpb.GetProductVariantListPageDataResponse{
		ProductVariantList: productVariants,
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

// GetProductVariantItemPageData retrieves a single product variant with enhanced item page data using CTE
// This method joins with the product table for the parent product reference
func (r *PostgresProductVariantRepository) GetProductVariantItemPageData(
	ctx context.Context,
	req *productvariantpb.GetProductVariantItemPageDataRequest,
) (*productvariantpb.GetProductVariantItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product variant item page data request is required")
	}
	if req.ProductVariantId == "" {
		return nil, fmt.Errorf("product variant ID is required")
	}

	// CTE Query - Single round-trip with product join
	query := `
		WITH enriched AS (
			SELECT
				pv.id,
				pv.date_created,
				pv.date_modified,
				pv.active,
				pv.product_id,
				pv.sku,
				pv.price_override,
				COALESCE(p.name, '') as product_name,
				COALESCE(p.price, 0) as product_price,
				COALESCE(p.currency, '') as product_currency
			FROM product_variant pv
			LEFT JOIN product p ON pv.product_id = p.id AND p.active = true
			WHERE pv.id = $1 AND pv.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.ProductVariantId)

	var (
		id              string
		dateCreated     time.Time
		dateModified    time.Time
		active          bool
		productID       string
		sku             string
		priceOverride   float64
		productName     string
		productPrice    float64
		productCurrency string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&productID,
		&sku,
		&priceOverride,
		&productName,
		&productPrice,
		&productCurrency,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("product variant with ID '%s' not found", req.ProductVariantId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query product variant item page data: %w", err)
	}

	productVariant := &productvariantpb.ProductVariant{
		Id:            id,
		Active:        active,
		ProductId:     productID,
		Sku:           sku,
		PriceOverride: priceOverride,
	}

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		productVariant.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		productVariant.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		productVariant.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		productVariant.DateModifiedString = &dmStr
	}

	// Note: productName, productPrice, productCurrency are available for the
	// product reference but not directly mapped to the ProductVariant protobuf.
	// These could be returned via the Product field or processed separately.

	return &productvariantpb.GetProductVariantItemPageDataResponse{
		ProductVariant: productVariant,
		Success:        true,
	}, nil
}

// NewProductVariantRepository creates a new PostgreSQL product variant repository (old-style constructor)
func NewProductVariantRepository(db *sql.DB, tableName string) productvariantpb.ProductVariantDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresProductVariantRepository(dbOps, tableName)
}
