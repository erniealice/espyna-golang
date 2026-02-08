//go:build mock_db

package product

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	datamock "leapfor.xyz/copya/golang"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// MockPriceProductRepository implements product.PriceProductRepository using stateful mock data
type MockPriceProductRepository struct {
	priceproductpb.UnimplementedPriceProductDomainServiceServer
	businessType    string
	priceProducts   map[string]*priceproductpb.PriceProduct // Persistent in-memory store
	mutex           sync.RWMutex                            // Thread-safe concurrent access
	initialized     bool                                    // Prevent double initialization
	skipInitialData bool                                    // Option to skip loading baseline data
	processor       *listdata.ListDataProcessor             // List data processor for filtering, sorting, searching, and pagination
}

// PriceProductRepositoryOption allows configuration of repository behavior
type PriceProductRepositoryOption func(*MockPriceProductRepository)

// WithPriceProductTestOptimizations enables test-specific optimizations
func WithPriceProductTestOptimizations(enabled bool) PriceProductRepositoryOption {
	return func(r *MockPriceProductRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// WithoutPriceProductInitialData prevents the repository from loading the baseline mock data.
func WithoutPriceProductInitialData() PriceProductRepositoryOption {
	return func(r *MockPriceProductRepository) {
		r.skipInitialData = true
	}
}

// NewMockPriceProductRepository creates a new mock price product repository
func NewMockPriceProductRepository(businessType string, options ...PriceProductRepositoryOption) priceproductpb.PriceProductDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockPriceProductRepository{
		businessType:  businessType,
		priceProducts: make(map[string]*priceproductpb.PriceProduct),
		processor:     listdata.NewListDataProcessor(),
	}

	// Apply optional configurations
	for _, option := range options {
		option(repo)
	}

	// Initialize with mock data once, unless skipped.
	if !repo.skipInitialData {
		if err := repo.loadInitialData(); err != nil {
			// Log error but don't fail - allows graceful degradation
			fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
		}
	}

	return repo
}

// loadInitialData loads mock data from the copya package
func (r *MockPriceProductRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawPriceProducts, err := datamock.LoadBusinessTypeModule(r.businessType, "price-product")
	if err != nil {
		return fmt.Errorf("failed to load initial price products: %w", err)
	}

	// Convert and store each price product
	for _, rawPriceProduct := range rawPriceProducts {
		if priceProduct, err := r.mapToProtobufPriceProduct(rawPriceProduct); err == nil {
			r.priceProducts[priceProduct.Id] = priceProduct
		}
	}

	r.initialized = true
	return nil
}

// CreatePriceProduct creates a new price product with stateful storage
func (r *MockPriceProductRepository) CreatePriceProduct(ctx context.Context, req *priceproductpb.CreatePriceProductRequest) (*priceproductpb.CreatePriceProductResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create price product request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("price product data is required")
	}
	if req.Data.ProductId == "" {
		return nil, fmt.Errorf("product ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	priceProductID := fmt.Sprintf("price-product-%d", now.UnixNano())

	priceProduct := &priceproductpb.PriceProduct{
		Id:                 priceProductID,
		ProductId:          req.Data.ProductId,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		Amount:             req.Data.Amount,
		Currency:           req.Data.Currency,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true,
	}

	// Store in persistent map
	r.priceProducts[priceProductID] = priceProduct

	return &priceproductpb.CreatePriceProductResponse{
		Data:    []*priceproductpb.PriceProduct{priceProduct},
		Success: true,
	}, nil
}

// ReadPriceProduct retrieves a price product by ID from stateful storage
func (r *MockPriceProductRepository) ReadPriceProduct(ctx context.Context, req *priceproductpb.ReadPriceProductRequest) (*priceproductpb.ReadPriceProductResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("read price product request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price product ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Retrieve from persistent storage
	priceProduct, exists := r.priceProducts[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("price product with ID '%s' not found", req.Data.Id)
	}

	return &priceproductpb.ReadPriceProductResponse{
		Data:    []*priceproductpb.PriceProduct{priceProduct},
		Success: true,
	}, nil
}

// UpdatePriceProduct updates an existing price product in stateful storage
func (r *MockPriceProductRepository) UpdatePriceProduct(ctx context.Context, req *priceproductpb.UpdatePriceProductRequest) (*priceproductpb.UpdatePriceProductResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("update price product request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price product ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if price product exists
	existingPriceProduct, exists := r.priceProducts[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("price product with ID '%s' not found", req.Data.Id)
	}

	// Update with new modification timestamp
	now := time.Now()
	updatedPriceProduct := &priceproductpb.PriceProduct{
		Id:                 req.Data.Id,
		ProductId:          req.Data.ProductId,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		Amount:             req.Data.Amount,
		Currency:           req.Data.Currency,
		DateCreated:        existingPriceProduct.DateCreated,       // Preserve original creation date
		DateCreatedString:  existingPriceProduct.DateCreatedString, // Preserve original creation date
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Store updated price product
	r.priceProducts[req.Data.Id] = updatedPriceProduct

	return &priceproductpb.UpdatePriceProductResponse{
		Data:    []*priceproductpb.PriceProduct{updatedPriceProduct},
		Success: true,
	}, nil
}

// DeletePriceProduct deletes a price product from stateful storage
func (r *MockPriceProductRepository) DeletePriceProduct(ctx context.Context, req *priceproductpb.DeletePriceProductRequest) (*priceproductpb.DeletePriceProductResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("delete price product request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price product ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if price product exists
	_, exists := r.priceProducts[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("price product with ID '%s' not found", req.Data.Id)
	}

	// Remove from persistent storage
	delete(r.priceProducts, req.Data.Id)

	return &priceproductpb.DeletePriceProductResponse{
		Success: true,
	}, nil
}

// ListPriceProducts retrieves all price products from stateful storage
func (r *MockPriceProductRepository) ListPriceProducts(ctx context.Context, req *priceproductpb.ListPriceProductsRequest) (*priceproductpb.ListPriceProductsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice
	priceProducts := make([]*priceproductpb.PriceProduct, 0, len(r.priceProducts))
	for _, priceProduct := range r.priceProducts {
		priceProducts = append(priceProducts, priceProduct)
	}

	return &priceproductpb.ListPriceProductsResponse{
		Data:    priceProducts,
		Success: true,
	}, nil
}

// mapToProtobufPriceProduct converts raw mock data to protobuf PriceProduct
func (r *MockPriceProductRepository) mapToProtobufPriceProduct(rawPriceProduct map[string]any) (*priceproductpb.PriceProduct, error) {
	priceProduct := &priceproductpb.PriceProduct{}

	// Map required fields
	if id, ok := rawPriceProduct["id"].(string); ok {
		priceProduct.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if productId, ok := rawPriceProduct["productId"].(string); ok {
		priceProduct.ProductId = productId
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawPriceProduct["dateCreated"].(string); ok {
		priceProduct.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			priceProduct.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawPriceProduct["dateModified"].(string); ok {
		priceProduct.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			priceProduct.DateModified = &timestamp
		}
	}

	if active, ok := rawPriceProduct["active"].(bool); ok {
		priceProduct.Active = active
	}

	// Map amount field
	if amount, ok := rawPriceProduct["amount"].(float64); ok {
		priceProduct.Amount = int64(amount)
	}

	// Map currency field
	if currency, ok := rawPriceProduct["currency"].(string); ok {
		priceProduct.Currency = currency
	}

	return priceProduct, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockPriceProductRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// GetPriceProductListPageData retrieves price products with advanced filtering, sorting, searching, and pagination
func (r *MockPriceProductRepository) GetPriceProductListPageData(
	ctx context.Context,
	req *priceproductpb.GetPriceProductListPageDataRequest,
) (*priceproductpb.GetPriceProductListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get price product list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of price products
	priceProducts := make([]*priceproductpb.PriceProduct, 0, len(r.priceProducts))
	for _, priceProduct := range r.priceProducts {
		priceProducts = append(priceProducts, priceProduct)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		priceProducts,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process price product list data: %w", err)
	}

	// Convert processed items back to price product protobuf format
	processedPriceProducts := make([]*priceproductpb.PriceProduct, len(result.Items))
	for i, item := range result.Items {
		if priceProduct, ok := item.(*priceproductpb.PriceProduct); ok {
			processedPriceProducts[i] = priceProduct
		} else {
			return nil, fmt.Errorf("failed to convert item to price product type")
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

	return &priceproductpb.GetPriceProductListPageDataResponse{
		PriceProductList: processedPriceProducts,
		Pagination:       result.PaginationResponse,
		SearchResults:    searchResults,
		Success:          true,
	}, nil
}

// GetPriceProductItemPageData retrieves a single price product with enhanced item page data
func (r *MockPriceProductRepository) GetPriceProductItemPageData(
	ctx context.Context,
	req *priceproductpb.GetPriceProductItemPageDataRequest,
) (*priceproductpb.GetPriceProductItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get price product item page data request is required")
	}
	if req.PriceProductId == "" {
		return nil, fmt.Errorf("price product ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	priceProduct, exists := r.priceProducts[req.PriceProductId]
	if !exists {
		return nil, fmt.Errorf("price product with ID '%s' not found", req.PriceProductId)
	}

	// In a real implementation, you might:
	// 1. Load related data (product details, pricing history)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &priceproductpb.GetPriceProductItemPageDataResponse{
		PriceProduct: priceProduct,
		Success:      true,
	}, nil
}

// NewPriceProductRepository creates a new mock price product repository (registry constructor)
func NewPriceProductRepository(data map[string]*priceproductpb.PriceProduct) priceproductpb.PriceProductDomainServiceServer {
	repo := &MockPriceProductRepository{
		businessType:  "education", // Default business type
		priceProducts: data,
		mutex:         sync.RWMutex{},
	}
	if data == nil {
		repo.priceProducts = make(map[string]*priceproductpb.PriceProduct)
		// Initialize with mock data
		if err := repo.loadInitialData(); err != nil {
			fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
		}
	}
	return repo
}

func init() {
	registry.RegisterRepositoryFactory("mock", "price_product", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockPriceProductRepository(businessType), nil
	})
}
