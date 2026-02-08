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
	productcollectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_collection"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// MockProductCollectionRepository implements product.ProductCollectionRepository using stateful mock data
type MockProductCollectionRepository struct {
	productcollectionpb.UnimplementedProductCollectionDomainServiceServer
	businessType       string
	productCollections map[string]*productcollectionpb.ProductCollection // Persistent in-memory store
	mutex              sync.RWMutex                                      // Thread-safe concurrent access
	initialized        bool                                              // Prevent double initialization
	processor          *listdata.ListDataProcessor                      // List data processing utilities
}

// ProductCollectionRepositoryOption allows configuration of repository behavior
type ProductCollectionRepositoryOption func(*MockProductCollectionRepository)

// WithProductCollectionTestOptimizations enables test-specific optimizations
func WithProductCollectionTestOptimizations(enabled bool) ProductCollectionRepositoryOption {
	return func(r *MockProductCollectionRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockProductCollectionRepository creates a new mock product collection repository
func NewMockProductCollectionRepository(businessType string, options ...ProductCollectionRepositoryOption) productcollectionpb.ProductCollectionDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockProductCollectionRepository{
		businessType:       businessType,
		productCollections: make(map[string]*productcollectionpb.ProductCollection),
		processor:          listdata.NewListDataProcessor(),
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
func (r *MockProductCollectionRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawProductCollections, err := datamock.LoadBusinessTypeModule(r.businessType, "product-collection")
	if err != nil {
		return fmt.Errorf("failed to load initial product collections: %w", err)
	}

	// Convert and store each product collection
	for _, rawProductCollection := range rawProductCollections {
		if productCollection, err := r.mapToProtobufProductCollection(rawProductCollection); err == nil {
			r.productCollections[productCollection.Id] = productCollection
		}
	}

	r.initialized = true
	return nil
}

// CreateProductCollection creates a new product collection with stateful storage
func (r *MockProductCollectionRepository) CreateProductCollection(ctx context.Context, req *productcollectionpb.CreateProductCollectionRequest) (*productcollectionpb.CreateProductCollectionResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create product collection request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("product collection data is required")
	}
	if req.Data.ProductId == "" {
		return nil, fmt.Errorf("product ID is required")
	}
	if req.Data.CollectionId == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	productCollectionID := fmt.Sprintf("product-collection-%d", now.UnixNano())

	productCollection := &productcollectionpb.ProductCollection{
		Id:                 productCollectionID,
		ProductId:          req.Data.ProductId,
		CollectionId:       req.Data.CollectionId,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true,
	}

	// Store in persistent map
	r.productCollections[productCollectionID] = productCollection

	return &productcollectionpb.CreateProductCollectionResponse{
		Data:    []*productcollectionpb.ProductCollection{productCollection},
		Success: true,
	}, nil
}

// ReadProductCollection retrieves a product collection by ID from stateful storage
func (r *MockProductCollectionRepository) ReadProductCollection(ctx context.Context, req *productcollectionpb.ReadProductCollectionRequest) (*productcollectionpb.ReadProductCollectionResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("read product collection request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product collection ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Retrieve from persistent storage
	productCollection, exists := r.productCollections[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("product collection with ID '%s' not found", req.Data.Id)
	}

	return &productcollectionpb.ReadProductCollectionResponse{
		Data:    []*productcollectionpb.ProductCollection{productCollection},
		Success: true,
	}, nil
}

// UpdateProductCollection updates an existing product collection in stateful storage
func (r *MockProductCollectionRepository) UpdateProductCollection(ctx context.Context, req *productcollectionpb.UpdateProductCollectionRequest) (*productcollectionpb.UpdateProductCollectionResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("update product collection request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product collection ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if product collection exists
	existingProductCollection, exists := r.productCollections[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("product collection with ID '%s' not found", req.Data.Id)
	}

	// Update with new modification timestamp
	now := time.Now()
	updatedProductCollection := &productcollectionpb.ProductCollection{
		Id:                 req.Data.Id,
		ProductId:          req.Data.ProductId,
		CollectionId:       req.Data.CollectionId,
		DateCreated:        existingProductCollection.DateCreated,       // Preserve original creation date
		DateCreatedString:  existingProductCollection.DateCreatedString, // Preserve original creation date
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Store updated product collection
	r.productCollections[req.Data.Id] = updatedProductCollection

	return &productcollectionpb.UpdateProductCollectionResponse{
		Data:    []*productcollectionpb.ProductCollection{updatedProductCollection},
		Success: true,
	}, nil
}

// DeleteProductCollection deletes a product collection from stateful storage
func (r *MockProductCollectionRepository) DeleteProductCollection(ctx context.Context, req *productcollectionpb.DeleteProductCollectionRequest) (*productcollectionpb.DeleteProductCollectionResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("delete product collection request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product collection ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if product collection exists
	_, exists := r.productCollections[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("product collection with ID '%s' not found", req.Data.Id)
	}

	// Remove from persistent storage
	delete(r.productCollections, req.Data.Id)

	return &productcollectionpb.DeleteProductCollectionResponse{
		Success: true,
	}, nil
}

// ListProductCollections retrieves all product collections from stateful storage
func (r *MockProductCollectionRepository) ListProductCollections(ctx context.Context, req *productcollectionpb.ListProductCollectionsRequest) (*productcollectionpb.ListProductCollectionsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice
	productCollections := make([]*productcollectionpb.ProductCollection, 0, len(r.productCollections))
	for _, productCollection := range r.productCollections {
		productCollections = append(productCollections, productCollection)
	}

	return &productcollectionpb.ListProductCollectionsResponse{
		Data:    productCollections,
		Success: true,
	}, nil
}

// mapToProtobufProductCollection converts raw mock data to protobuf ProductCollection
func (r *MockProductCollectionRepository) mapToProtobufProductCollection(rawProductCollection map[string]any) (*productcollectionpb.ProductCollection, error) {
	productCollection := &productcollectionpb.ProductCollection{}

	// Map required fields
	if id, ok := rawProductCollection["id"].(string); ok {
		productCollection.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if productId, ok := rawProductCollection["productId"].(string); ok {
		productCollection.ProductId = productId
	}

	if collectionId, ok := rawProductCollection["collectionId"].(string); ok {
		productCollection.CollectionId = collectionId
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawProductCollection["dateCreated"].(string); ok {
		productCollection.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			productCollection.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawProductCollection["dateModified"].(string); ok {
		productCollection.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			productCollection.DateModified = &timestamp
		}
	}

	if active, ok := rawProductCollection["active"].(bool); ok {
		productCollection.Active = active
	}

	return productCollection, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockProductCollectionRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// GetProductCollectionListPageData retrieves product collections with advanced filtering, sorting, searching, and pagination
func (r *MockProductCollectionRepository) GetProductCollectionListPageData(
	ctx context.Context,
	req *productcollectionpb.GetProductCollectionListPageDataRequest,
) (*productcollectionpb.GetProductCollectionListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product collection list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of product collections
	productCollections := make([]*productcollectionpb.ProductCollection, 0, len(r.productCollections))
	for _, productCollection := range r.productCollections {
		productCollections = append(productCollections, productCollection)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		productCollections,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process product collection list data: %w", err)
	}

	// Convert processed items back to product collection protobuf format
	processedProductCollections := make([]*productcollectionpb.ProductCollection, len(result.Items))
	for i, item := range result.Items {
		if productCollection, ok := item.(*productcollectionpb.ProductCollection); ok {
			processedProductCollections[i] = productCollection
		} else {
			return nil, fmt.Errorf("failed to convert item to product collection type")
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

	return &productcollectionpb.GetProductCollectionListPageDataResponse{
		ProductCollectionList: processedProductCollections,
		Pagination:            result.PaginationResponse,
		SearchResults:         searchResults,
		Success:               true,
	}, nil
}

// GetProductCollectionItemPageData retrieves a single product collection with enhanced item page data
func (r *MockProductCollectionRepository) GetProductCollectionItemPageData(
	ctx context.Context,
	req *productcollectionpb.GetProductCollectionItemPageDataRequest,
) (*productcollectionpb.GetProductCollectionItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product collection item page data request is required")
	}
	if req.ProductCollectionId == "" {
		return nil, fmt.Errorf("product collection ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	productCollection, exists := r.productCollections[req.ProductCollectionId]
	if !exists {
		return nil, fmt.Errorf("product collection with ID '%s' not found", req.ProductCollectionId)
	}

	// In a real implementation, you might:
	// 1. Load related data (product details, collection details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &productcollectionpb.GetProductCollectionItemPageDataResponse{
		ProductCollection: productCollection,
		Success:           true,
	}, nil
}

// NewProductCollectionRepository creates a new mock product collection repository (registry constructor)
func NewProductCollectionRepository(data map[string]*productcollectionpb.ProductCollection) productcollectionpb.ProductCollectionDomainServiceServer {
	repo := &MockProductCollectionRepository{
		businessType:       "education", // Default business type
		productCollections: data,
		mutex:              sync.RWMutex{},
		processor:          listdata.NewListDataProcessor(),
	}
	if data == nil {
		repo.productCollections = make(map[string]*productcollectionpb.ProductCollection)
		// Initialize with mock data
		if err := repo.loadInitialData(); err != nil {
			fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
		}
	}
	return repo
}

func init() {
	registry.RegisterRepositoryFactory("mock", "product_collection", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockProductCollectionRepository(businessType), nil
	})
}
