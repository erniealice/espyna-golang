//go:build postgresql

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Product, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres product repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
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
//   - CREATE INDEX idx_product_plan_product_id ON product_plan(product_id) - Junction table FK
//   - CREATE INDEX idx_product_plan_plan_id ON product_plan(plan_id) - Junction table FK
//   - CREATE INDEX idx_product_plan_active ON product_plan(active) - Junction table filter
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
//   - Test with product having inactive plans (should be filtered out)
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
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	product := &productpb.Product{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, product); err != nil {
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
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	product := &productpb.Product{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, product); err != nil {
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

	// Always include active flag — proto3 omits bool=false during JSON marshal,
	// which would silently skip deactivation via the form toggle.
	data["active"] = req.Data.GetActive()

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	product := &productpb.Product{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, product); err != nil {
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

	// Hard delete — catalog entities rely on FK RESTRICT to block deletion
	// when historical references exist (revenue_line_item, inventory_item, etc.).
	err := r.dbOps.HardDelete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete product: %w", err)
	}

	return &productpb.DeleteProductResponse{
		Success: true,
	}, nil
}

var productSortableSQLCols = []string{
	"id", "active", "name", "description", "price", "tracking_mode",
	"product_type", "unit_of_measure", "date_created", "date_modified",
}

var productSortSpec = espynahttp.SortSpec{AllowedCols: productSortableSQLCols}

// ListProducts lists products using common PostgreSQL operations
func (r *PostgresProductRepository) ListProducts(ctx context.Context, req *productpb.ListProductsRequest) (*productpb.ListProductsResponse, error) {
	if err := espynahttp.ValidateSortColumns(productSortSpec, req.GetSort(), "product"); err != nil {
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
		return nil, fmt.Errorf("failed to list products: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var products []*productpb.Product
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal product row: %v", err)
			continue
		}

		product := &productpb.Product{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, product); err != nil {
			log.Printf("WARN: protojson unmarshal product: %v", err)
			continue
		}
		products = append(products, product)
	}

	return &productpb.ListProductsResponse{
		Data: products,
	}, nil
}

// GetProductListPageData retrieves products by composing over ListProducts.
//
// Canonical pattern (plan 20260429-pagedata-canonicalize §3): the page-data
// layer adds caller intent (active filter, pagination metadata) on top of the
// field-agnostic ListProducts adapter. Avoids drift when proto fields are added.
func (r *PostgresProductRepository) GetProductListPageData(
	ctx context.Context,
	req *productpb.GetProductListPageDataRequest,
) (*productpb.GetProductListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product list page data request is required")
	}

	// Default pagination values
	limit := int32(50)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil && offsetPag.Page > 0 {
			page = offsetPag.Page
		}
	}

	// Delegate field projection + filtering to the canonical ListProducts.
	lr, err := r.ListProducts(ctx, &productpb.ListProductsRequest{
		Search:     req.Search,
		Filters:    req.Filters,
		Sort:       req.Sort,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}

	// Preserve page-data intent: only active products on this surface.
	all := lr.GetData()
	products := make([]*productpb.Product, 0, len(all))
	for _, p := range all {
		if p != nil && p.GetActive() {
			products = append(products, p)
		}
	}

	totalCount := int64(len(products))
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

// GetProductItemPageData retrieves a single product by composing over ReadProduct.
//
// Canonical pattern (plan 20260429-pagedata-canonicalize §3): delegate the full
// proto field projection to ReadProduct (which round-trips through dbOps.Read +
// protojson with DiscardUnknown — picks up new fields automatically). The
// page-data layer only enforces caller intent (active filter, "not found" if
// inactive) and adjacent denorms. Per plan §3 caveats, default_variant lookup
// stays via ReadProductVariant when the response message carries that field;
// the current GetProductItemPageDataResponse only carries Product, so no extra
// reads are needed here.
func (r *PostgresProductRepository) GetProductItemPageData(
	ctx context.Context,
	req *productpb.GetProductItemPageDataRequest,
) (*productpb.GetProductItemPageDataResponse, error) {
	if req == nil || req.ProductId == "" {
		return nil, fmt.Errorf("product ID is required")
	}

	rr, err := r.ReadProduct(ctx, &productpb.ReadProductRequest{
		Data: &productpb.Product{Id: req.ProductId},
	})
	if err != nil {
		return nil, err
	}
	if len(rr.GetData()) == 0 {
		return nil, fmt.Errorf("product with ID '%s' not found", req.ProductId)
	}
	product := rr.GetData()[0]

	// Preserve page-data intent: only active products on this surface.
	if !product.GetActive() {
		return nil, fmt.Errorf("product with ID '%s' not found", req.ProductId)
	}

	return &productpb.GetProductItemPageDataResponse{
		Product: product,
		Success: true,
	}, nil
}

// NewProductRepository creates a new PostgreSQL product repository (old-style constructor)
func NewProductRepository(db *sql.DB, tableName string) productpb.ProductDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresProductRepository(dbOps, tableName)
}