//go:build mysql

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.Product, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql product repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLProductRepository(dbOps, tableName), nil
	})
}

// MySQLProductRepository implements product CRUD operations using MySQL 8.0+.
type MySQLProductRepository struct {
	productpb.UnimplementedProductDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewMySQLProductRepository creates a new MySQL product repository.
func NewMySQLProductRepository(dbOps interfaces.DatabaseOperation, tableName string) productpb.ProductDomainServiceServer {
	if tableName == "" {
		tableName = "product" // default fallback
	}
	return &MySQLProductRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreateProduct creates a new product using common MySQL operations.
func (r *MySQLProductRepository) CreateProduct(ctx context.Context, req *productpb.CreateProductRequest) (*productpb.CreateProductResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("product data is required")
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
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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

// ReadProduct retrieves a product using common MySQL operations.
func (r *MySQLProductRepository) ReadProduct(ctx context.Context, req *productpb.ReadProductRequest) (*productpb.ReadProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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

// UpdateProduct updates a product using common MySQL operations.
func (r *MySQLProductRepository) UpdateProduct(ctx context.Context, req *productpb.UpdateProductRequest) (*productpb.UpdateProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product ID is required")
	}

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

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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

// DeleteProduct deletes a product using common MySQL operations.
// Hard delete — catalog entities rely on FK RESTRICT to block deletion when
// historical references exist (revenue_line_item, inventory_item, etc.).
func (r *MySQLProductRepository) DeleteProduct(ctx context.Context, req *productpb.DeleteProductRequest) (*productpb.DeleteProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product ID is required")
	}

	if err := r.dbOps.HardDelete(ctx, r.tableName, req.Data.Id); err != nil {
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

// ListProducts lists products using common MySQL operations.
func (r *MySQLProductRepository) ListProducts(ctx context.Context, req *productpb.ListProductsRequest) (*productpb.ListProductsResponse, error) {
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

	var products []*productpb.Product
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
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
// Canonical pattern: the page-data layer adds caller intent (active filter,
// pagination metadata) on top of the field-agnostic ListProducts adapter.
func (r *MySQLProductRepository) GetProductListPageData(
	ctx context.Context,
	req *productpb.GetProductListPageDataRequest,
) (*productpb.GetProductListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product list page data request is required")
	}

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
// Canonical pattern: delegate the full proto field projection to ReadProduct.
// The page-data layer only enforces caller intent (active filter, "not found"
// if inactive).
func (r *MySQLProductRepository) GetProductItemPageData(
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

// NewProductRepository creates a new MySQL product repository (old-style constructor).
func NewProductRepository(db *sql.DB, tableName string) productpb.ProductDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLProductRepository(dbOps, tableName)
}
