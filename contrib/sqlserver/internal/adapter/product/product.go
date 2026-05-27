//go:build sqlserver

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.Product, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver product repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerProductRepository(dbOps, tableName), nil
	})
}

// SQLServerProductRepository implements product CRUD using SQL Server.
type SQLServerProductRepository struct {
	productpb.UnimplementedProductDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerProductRepository creates a new SQL Server product repository.
func NewSQLServerProductRepository(dbOps interfaces.DatabaseOperation, tableName string) productpb.ProductDomainServiceServer {
	if tableName == "" {
		tableName = "product"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &SQLServerProductRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func (r *SQLServerProductRepository) CreateProduct(ctx context.Context, req *productpb.CreateProductRequest) (*productpb.CreateProductResponse, error) {
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
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	product := &productpb.Product{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, product); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productpb.CreateProductResponse{Data: []*productpb.Product{product}}, nil
}

func (r *SQLServerProductRepository) ReadProduct(ctx context.Context, req *productpb.ReadProductRequest) (*productpb.ReadProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	product := &productpb.Product{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, product); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productpb.ReadProductResponse{Data: []*productpb.Product{product}}, nil
}

func (r *SQLServerProductRepository) UpdateProduct(ctx context.Context, req *productpb.UpdateProductRequest) (*productpb.UpdateProductResponse, error) {
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
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	product := &productpb.Product{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, product); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &productpb.UpdateProductResponse{Data: []*productpb.Product{product}}, nil
}

func (r *SQLServerProductRepository) DeleteProduct(ctx context.Context, req *productpb.DeleteProductRequest) (*productpb.DeleteProductResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product ID is required")
	}
	// Hard delete — catalog entities rely on FK RESTRICT to block deletion
	// when historical references exist (revenue_line_item, inventory_item, etc.).
	if err := r.dbOps.HardDelete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete product: %w", err)
	}
	return &productpb.DeleteProductResponse{Success: true}, nil
}

func (r *SQLServerProductRepository) ListProducts(ctx context.Context, req *productpb.ListProductsRequest) (*productpb.ListProductsResponse, error) {
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
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		product := &productpb.Product{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, product); err != nil {
			continue
		}
		products = append(products, product)
	}
	return &productpb.ListProductsResponse{Data: products}, nil
}

// GetProductListPageData delegates to ListProducts and filters to active-only.
// Canonical pattern (plan 20260429-pagedata-canonicalize §3).
func (r *SQLServerProductRepository) GetProductListPageData(
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

// GetProductItemPageData delegates to ReadProduct and enforces active filter.
// Canonical pattern (plan 20260429-pagedata-canonicalize §3).
func (r *SQLServerProductRepository) GetProductItemPageData(
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
	if !product.GetActive() {
		return nil, fmt.Errorf("product with ID '%s' not found", req.ProductId)
	}
	return &productpb.GetProductItemPageDataResponse{
		Product: product,
		Success: true,
	}, nil
}
