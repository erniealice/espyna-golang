//go:build postgres

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
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "product", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres product repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresProductRepository(dbOps, tableName), nil
	})
}

// PostgresProductRepository implements product CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_product_active ON product(active) - Filter active records
//   - CREATE INDEX idx_product_date_created ON product(date_created DESC) - Default sorting
//   - CREATE INDEX idx_product_name ON product(name) - Search field
//   - CREATE INDEX idx_product_description ON product(description) - Search field (consider GIN index for full-text search)
//   - CREATE INDEX idx_product_attribute_product_id ON product_attribute(product_id) - Junction table FK
//   - CREATE INDEX idx_product_attribute_active ON product_attribute(active) - Junction table filter
//   - CREATE INDEX idx_product_collection_product_id ON product_collection(product_id) - Junction table FK
//   - CREATE INDEX idx_product_collection_collection_id ON product_collection(collection_id) - Junction table FK
//   - CREATE INDEX idx_product_collection_active ON product_collection(active) - Junction table filter
//   - CREATE INDEX idx_product_plan_product_id ON product_plan(product_id) - Junction table FK
//   - CREATE INDEX idx_product_plan_plan_id ON product_plan(plan_id) - Junction table FK
//   - CREATE INDEX idx_product_plan_active ON product_plan(active) - Junction table filter
//   - CREATE INDEX idx_collection_active ON collection(active) - Related table filter
//   - CREATE INDEX idx_plan_active ON plan(active) - Related table filter
//
// TODO: Add comprehensive tests for GetProductListPageData:
//   - Test with no search query (list all active products)
//   - Test with search query matching product name
//   - Test with search query matching product description
//   - Test pagination (page 1, page 2, page size variations)
//   - Test sorting (by different fields, ASC and DESC)
//   - Test with no matching results
//   - Test with inactive products (should be filtered out)
//   - Test with product_attributes aggregation (1:Many relationship)
//   - Test with product_collections aggregation (Many:Many via junction)
//   - Test with product_plans aggregation (Many:Many via junction)
//   - Test with inactive related records (should be filtered out)
//   - Test with null/empty aggregations (no related records)
//
// TODO: Add comprehensive tests for GetProductItemPageData:
//   - Test with valid product ID (with all relationships populated)
//   - Test with valid product ID (without any relationships)
//   - Test with non-existent product ID
//   - Test with inactive product (should return not found)
//   - Test with product having inactive attributes (should be filtered out)
//   - Test with product having inactive collections/plans (should be filtered out)
//   - Test timestamp parsing for date_created and date_modified
//   - Test nullable description field
type PostgresProductRepository struct {
	productpb.UnimplementedProductDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresProductRepository creates a new PostgreSQL product repository
func NewPostgresProductRepository(dbOps interfaces.DatabaseOperation, tableName string) productpb.ProductDomainServiceServer {
	if tableName == "" {
		tableName = "product" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresProductRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateProduct creates a new product using common PostgreSQL operations
func (r *PostgresProductRepository) CreateProduct(ctx context.Context, req *productpb.CreateProductRequest) (*productpb.CreateProductResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product data is required")
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
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	product := &productpb.Product{}
	if err := protojson.Unmarshal(resultJSON, product); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productpb.CreateProductResponse{
		Data: []*productpb.Product{product},
	}, nil
}

// ReadProduct retrieves a product using common PostgreSQL operations
func (r *PostgresProductRepository) ReadProduct(ctx context.Context, req *productpb.ReadProductRequest) (*productpb.ReadProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	product := &productpb.Product{}
	if err := protojson.Unmarshal(resultJSON, product); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productpb.ReadProductResponse{
		Data: []*productpb.Product{product},
	}, nil
}

// UpdateProduct updates a product using common PostgreSQL operations
func (r *PostgresProductRepository) UpdateProduct(ctx context.Context, req *productpb.UpdateProductRequest) (*productpb.UpdateProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product ID is required")
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
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	product := &productpb.Product{}
	if err := protojson.Unmarshal(resultJSON, product); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &productpb.UpdateProductResponse{
		Data: []*productpb.Product{product},
	}, nil
}

// DeleteProduct deletes a product using common PostgreSQL operations
func (r *PostgresProductRepository) DeleteProduct(ctx context.Context, req *productpb.DeleteProductRequest) (*productpb.DeleteProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete product: %w", err)
	}

	return &productpb.DeleteProductResponse{
		Success: true,
	}, nil
}

// ListProducts lists products using common PostgreSQL operations
func (r *PostgresProductRepository) ListProducts(ctx context.Context, req *productpb.ListProductsRequest) (*productpb.ListProductsResponse, error) {
	// List documents using common operations
	listResult, err := r.dbOps.List(ctx, r.tableName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var products []*productpb.Product
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		product := &productpb.Product{}
		if err := protojson.Unmarshal(resultJSON, product); err != nil {
			// Log error and continue with next item
			continue
		}
		products = append(products, product)
	}

	return &productpb.ListProductsResponse{
		Data: products,
	}, nil
}

// GetProductListPageData retrieves products with advanced filtering, sorting, searching, and pagination using CTE
// This method aggregates three types of relationships:
// - product_attribute (1:Many) - Direct attributes on the product
// - product_collection (Many:Many via junction) - Collections the product belongs to
// - product_plan (Many:Many via junction) - Plans associated with the product
func (r *PostgresProductRepository) GetProductListPageData(
	ctx context.Context,
	req *productpb.GetProductListPageDataRequest,
) (*productpb.GetProductListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product list page data request is required")
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

	// CTE Query - Single round-trip with three separate aggregations for relationships
	// Performance Notes:
	// - INDEX RECOMMENDATION: Create indexes on all junction table foreign keys (product_id, collection_id, plan_id)
	// - INDEX RECOMMENDATION: Create indexes on active flags for all junction and related tables
	// - INDEX RECOMMENDATION: Create index on product.name and product.description for search performance
	// - Uses 3 separate CTEs to aggregate each relationship type independently
	// - COALESCE ensures empty arrays when no relationships exist (never NULL)
	query := `
		WITH
		product_attributes_agg AS (
			SELECT
				pa.product_id,
				jsonb_agg(jsonb_build_object(
					'id', pa.id,
					'attribute_id', pa.attribute_id,
					'value', pa.value
				) ORDER BY pa.id) as attributes
			FROM product_attribute pa
			WHERE pa.active = true
			GROUP BY pa.product_id
		),
		product_collections_agg AS (
			SELECT
				pc.product_id,
				jsonb_agg(jsonb_build_object(
					'id', pc.id,
					'collection_id', pc.collection_id,
					'sort_order', pc.sort_order
				) ORDER BY pc.sort_order) as collections
			FROM product_collection pc
			JOIN collection c ON pc.collection_id = c.id
			WHERE pc.active = true AND c.active = true
			GROUP BY pc.product_id
		),
		product_plans_agg AS (
			SELECT
				pp.product_id,
				jsonb_agg(jsonb_build_object(
					'id', pp.id,
					'plan_id', pp.plan_id,
					'name', p.name,
					'description', p.description,
					'price', p.price,
					'currency', p.currency
				) ORDER BY pp.id) as plans
			FROM product_plan pp
			JOIN plan p ON pp.plan_id = p.id
			WHERE pp.active = true AND p.active = true
			GROUP BY pp.product_id
		),
		enriched AS (
			SELECT
				p.id,
				p.date_created,
				p.date_modified,
				p.active,
				p.name,
				p.description,
				p.price,
				p.currency
				COALESCE(paa.attributes, '[]'::jsonb) as product_attributes
				COALESCE(pca.collections, '[]'::jsonb) as product_collections
				COALESCE(ppa.plans, '[]'::jsonb) as product_plans
			FROM product p
			LEFT JOIN product_attributes_agg paa ON p.id = paa.product_id
			LEFT JOIN product_collections_agg pca ON p.id = pca.product_id
			LEFT JOIN product_plans_agg ppa ON p.id = ppa.product_id
			WHERE p.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       p.name ILIKE $1 OR
			       p.description ILIKE $1)
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
		return nil, fmt.Errorf("failed to query product list page data: %w", err)
	}
	defer rows.Close()

	var products []*productpb.Product
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			dateCreated        time.Time
			dateModified       time.Time
			active             bool
			name               string
			description        *string
			price              float64
			currency           string
			productAttributes  []byte // jsonb
			productCollections []byte // jsonb
			productPlans       []byte // jsonb
			total              int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&name,
			&description,
			&price,
			&currency,
			&productAttributes,
			&productCollections,
			&productPlans,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product row: %w", err)
		}

		totalCount = total

		product := &productpb.Product{
			Id:       id,
			Active:   active,
			Name:     name,
			Price:    price,
			Currency: currency,
		}

		// Handle nullable description field
		if description != nil {
			product.Description = description
		}

		// Handle date fields

		// Parse timestamps if provided
		if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		product.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		product.DateCreatedString = &dcStr
	}
		if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		product.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		product.DateModifiedString = &dmStr
	}

		// Note: The aggregated relationship data (productAttributes, productCollections, productPlans)
		// is available in JSONB format but not directly mapped to the Product protobuf structure
		// in this list view. This is intentional as the Product message doesn't include these
		// nested collections in its schema. If needed, these could be returned in a future
		// enhanced response structure or processed separately.

		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &productpb.GetProductListPageDataResponse{
		ProductList: products,
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

// GetProductItemPageData retrieves a single product with enhanced item page data using CTE
// This method aggregates all three relationship types for a complete product view
func (r *PostgresProductRepository) GetProductItemPageData(
	ctx context.Context,
	req *productpb.GetProductItemPageDataRequest,
) (*productpb.GetProductItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product item page data request is required")
	}
	if req.ProductId == "" {
		return nil, fmt.Errorf("product ID is required")
	}

	// CTE Query - Single round-trip with three separate aggregations for relationships
	query := `
		WITH
		product_attributes_agg AS (
			SELECT
				pa.product_id,
				jsonb_agg(jsonb_build_object(
					'id', pa.id,
					'attribute_id', pa.attribute_id,
					'value', pa.value
				) ORDER BY pa.id) as attributes
			FROM product_attribute pa
			WHERE pa.active = true AND pa.product_id = $1
			GROUP BY pa.product_id
		),
		product_collections_agg AS (
			SELECT
				pc.product_id,
				jsonb_agg(jsonb_build_object(
					'id', pc.id,
					'collection_id', pc.collection_id,
					'sort_order', pc.sort_order
				) ORDER BY pc.sort_order) as collections
			FROM product_collection pc
			JOIN collection c ON pc.collection_id = c.id
			WHERE pc.active = true AND c.active = true AND pc.product_id = $1
			GROUP BY pc.product_id
		),
		product_plans_agg AS (
			SELECT
				pp.product_id,
				jsonb_agg(jsonb_build_object(
					'id', pp.id,
					'plan_id', pp.plan_id,
					'name', p.name,
					'description', p.description,
					'price', p.price,
					'currency', p.currency
				) ORDER BY pp.id) as plans
			FROM product_plan pp
			JOIN plan p ON pp.plan_id = p.id
			WHERE pp.active = true AND p.active = true AND pp.product_id = $1
			GROUP BY pp.product_id
		),
		enriched AS (
			SELECT
				p.id,
				p.date_created,
				p.date_modified,
				p.active,
				p.name,
				p.description,
				p.price,
				p.currency
				COALESCE(paa.attributes, '[]'::jsonb) as product_attributes
				COALESCE(pca.collections, '[]'::jsonb) as product_collections
				COALESCE(ppa.plans, '[]'::jsonb) as product_plans
			FROM product p
			LEFT JOIN product_attributes_agg paa ON p.id = paa.product_id
			LEFT JOIN product_collections_agg pca ON p.id = pca.product_id
			LEFT JOIN product_plans_agg ppa ON p.id = ppa.product_id
			WHERE p.id = $1 AND p.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.ProductId)

	var (
		id                 string
		dateCreated        time.Time
		dateModified       time.Time
		active             bool
		name               string
		description        *string
		price              float64
		currency           string
		productAttributes  []byte // jsonb
		productCollections []byte // jsonb
		productPlans       []byte // jsonb
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&name,
		&description,
		&price,
		&currency,
		&productAttributes,
		&productCollections,
		&productPlans,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("product with ID '%s' not found", req.ProductId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query product item page data: %w", err)
	}

	product := &productpb.Product{
		Id:       id,
		Active:   active,
		Name:     name,
		Price:    price,
		Currency: currency,
	}

	// Handle nullable description field
	if description != nil {
		product.Description = description
	}

	// Handle date fields

	// Parse timestamps if provided
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		product.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		product.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		product.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		product.DateModifiedString = &dmStr
	}

	// Note: The aggregated relationship data (productAttributes, productCollections, productPlans)
	// is available in JSONB format but not directly mapped to the Product protobuf structure.
	// This is intentional as the Product message doesn't include these nested collections in
	// its schema. If needed, these could be returned in a future enhanced response structure
	// or processed separately for frontend consumption.

	return &productpb.GetProductItemPageDataResponse{
		Product: product,
		Success: true,
	}, nil
}

// parseTimestamp converts string timestamp to Unix timestamp (milliseconds)
func parseTimestamp(timestampStr string) (int64, error) {
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

// NewProductRepository creates a new PostgreSQL product repository (old-style constructor)
func NewProductRepository(db *sql.DB, tableName string) productpb.ProductDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresProductRepository(dbOps, tableName)
}
