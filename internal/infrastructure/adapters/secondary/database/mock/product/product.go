//go:build mock_db

package product

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	datamock "leapfor.xyz/copya/golang"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// MockProductRepository implements product.ProductRepository using stateful mock data
type MockProductRepository struct {
	productpb.UnimplementedProductDomainServiceServer
	businessType string
	products     map[string]*productpb.Product // Persistent in-memory store
	mutex        sync.RWMutex                  // Thread-safe concurrent access
	initialized  bool                          // Prevent double initialization
	processor    *listdata.ListDataProcessor   // List data processing utilities
}

// ProductRepositoryOption allows configuration of repository behavior
type ProductRepositoryOption func(*MockProductRepository)

// WithProductTestOptimizations enables test-specific optimizations
func WithProductTestOptimizations(enabled bool) ProductRepositoryOption {
	return func(r *MockProductRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockProductRepository creates a new mock product repository
func NewMockProductRepository(businessType string, options ...ProductRepositoryOption) productpb.ProductDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockProductRepository{
		businessType: businessType,
		products:     make(map[string]*productpb.Product),
		processor:    listdata.NewListDataProcessor(),
	}

	// Apply optional configurations
	for _, option := range options {
		option(repo)
	}

	// Initialize with mock data once
	if err := repo.loadInitialData(); err != nil {
		// Log error but don't fail - allows graceful degradation
		fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
	}

	return repo
}

// loadInitialData loads mock data from the copya package
func (r *MockProductRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawProducts, err := datamock.LoadBusinessTypeModule(r.businessType, "product")
	if err != nil {
		return fmt.Errorf("failed to load initial products: %w", err)
	}

	// Convert and store each product
	for _, rawProduct := range rawProducts {
		if product, err := r.mapToProtobufProduct(rawProduct); err == nil {
			r.products[product.Id] = product
		}
	}

	r.initialized = true
	return nil
}

// CreateProduct creates a new product with stateful storage
func (r *MockProductRepository) CreateProduct(ctx context.Context, req *productpb.CreateProductRequest) (*productpb.CreateProductResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create product request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("product data is required")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("product name is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate ID and timestamps for new product
	now := time.Now()
	productID := req.Data.Id
	if productID == "" {
		productID = fmt.Sprintf("product-%d", now.UnixNano())
	}

	product := &productpb.Product{
		Id:                 productID,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		Price:              req.Data.Price,
		Currency:           req.Data.Currency,
		ProductCollections: req.Data.ProductCollections,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true,
	}

	// Store in persistent map
	r.products[productID] = product

	return &productpb.CreateProductResponse{
		Data:    []*productpb.Product{product},
		Success: true,
	}, nil
}

// ReadProduct retrieves a product by ID from stateful storage
func (r *MockProductRepository) ReadProduct(ctx context.Context, req *productpb.ReadProductRequest) (*productpb.ReadProductResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("read product request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Retrieve from persistent storage
	product, exists := r.products[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("product with ID '%s' not found", req.Data.Id)
	}

	return &productpb.ReadProductResponse{
		Data:    []*productpb.Product{product},
		Success: true,
	}, nil
}

// UpdateProduct updates an existing product in stateful storage
func (r *MockProductRepository) UpdateProduct(ctx context.Context, req *productpb.UpdateProductRequest) (*productpb.UpdateProductResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("update product request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if product exists
	existingProduct, exists := r.products[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("product with ID '%s' not found", req.Data.Id)
	}

	// Update with new modification timestamp
	now := time.Now()
	updatedProduct := &productpb.Product{
		Id:                 req.Data.Id,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		DateCreated:        existingProduct.DateCreated,       // Preserve original creation date
		DateCreatedString:  existingProduct.DateCreatedString, // Preserve original creation date
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Store updated product
	r.products[req.Data.Id] = updatedProduct

	return &productpb.UpdateProductResponse{
		Data:    []*productpb.Product{updatedProduct},
		Success: true,
	}, nil
}

// DeleteProduct deletes a product from stateful storage
func (r *MockProductRepository) DeleteProduct(ctx context.Context, req *productpb.DeleteProductRequest) (*productpb.DeleteProductResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("delete product request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if product exists
	_, exists := r.products[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("product with ID '%s' not found", req.Data.Id)
	}

	// Remove from persistent storage
	delete(r.products, req.Data.Id)

	return &productpb.DeleteProductResponse{
		Success: true,
	}, nil
}

// ListProducts retrieves all products from stateful storage
func (r *MockProductRepository) ListProducts(ctx context.Context, req *productpb.ListProductsRequest) (*productpb.ListProductsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice
	products := make([]*productpb.Product, 0, len(r.products))
	for _, product := range r.products {
		products = append(products, product)
	}

	return &productpb.ListProductsResponse{
		Data:    products,
		Success: true,
	}, nil
}

// mapToProtobufProduct converts raw mock data to protobuf Product
func (r *MockProductRepository) mapToProtobufProduct(rawProduct map[string]any) (*productpb.Product, error) {
	product := &productpb.Product{}

	// Map required fields
	if id, ok := rawProduct["id"].(string); ok {
		product.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawProduct["name"].(string); ok {
		product.Name = name
	}

	if description, ok := rawProduct["description"].(string); ok {
		product.Description = &description
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawProduct["dateCreated"].(string); ok {
		product.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			product.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawProduct["dateModified"].(string); ok {
		product.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			product.DateModified = &timestamp
		}
	}

	if active, ok := rawProduct["active"].(bool); ok {
		product.Active = active
	}

	return product, nil
}

// GetProductListPageData retrieves products with advanced filtering, sorting, searching, and pagination
func (r *MockProductRepository) GetProductListPageData(
	ctx context.Context,
	req *productpb.GetProductListPageDataRequest,
) (*productpb.GetProductListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of products
	products := make([]*productpb.Product, 0, len(r.products))
	for _, product := range r.products {
		products = append(products, product)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		products,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process product list data: %w", err)
	}

	// Convert processed items back to product protobuf format
	processedProducts := make([]*productpb.Product, len(result.Items))
	for i, item := range result.Items {
		if product, ok := item.(*productpb.Product); ok {
			processedProducts[i] = product
		} else {
			return nil, fmt.Errorf("failed to convert item to product type")
		}
	}

	// Convert search results to protobuf format
	searchResults := make([]*commonpb.SearchResult, len(result.SearchResults))
	for i, searchResult := range result.SearchResults {
		searchResults[i] = &commonpb.SearchResult{
			Score:      searchResult.Score,
			Highlights: searchResult.Highlights,
		}
	}

	return &productpb.GetProductListPageDataResponse{
		ProductList:   processedProducts,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// GetProductItemPageData retrieves a single product with enhanced item page data
func (r *MockProductRepository) GetProductItemPageData(
	ctx context.Context,
	req *productpb.GetProductItemPageDataRequest,
) (*productpb.GetProductItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product item page data request is required")
	}
	if req.ProductId == "" {
		return nil, fmt.Errorf("product ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	product, exists := r.products[req.ProductId]
	if !exists {
		return nil, fmt.Errorf("product with ID '%s' not found", req.ProductId)
	}

	// In a real implementation, you might:
	// 1. Load related data (collection details, plan details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &productpb.GetProductItemPageDataResponse{
		Product: product,
		Success: true,
	}, nil
}

// NewProductRepository creates a new mock product repository (registry constructor)
func NewProductRepository(data map[string]*productpb.Product) productpb.ProductDomainServiceServer {
	repo := &MockProductRepository{
		businessType: "education", // Default business type
		products:     data,
		mutex:        sync.RWMutex{},
		processor:    listdata.NewListDataProcessor(),
	}
	if data == nil {
		repo.products = make(map[string]*productpb.Product)
		// Initialize with mock data
		if err := repo.loadInitialData(); err != nil {
			fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
		}
	}
	return repo
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockProductRepository) parseTimestamp(timestampStr string) (int64, error) {
	// Try parsing as Unix timestamp first
	if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
		return timestamp, nil
	}

	// Try parsing as RFC3339 format
	if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
		return t.UnixMilli(), nil
	}

	// Try parsing as other common formats
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

func init() {
	registry.RegisterRepositoryFactory("mock", "product", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockProductRepository(businessType), nil
	})
}
