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
	productattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_attribute"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// MockProductAttributeRepository implements product.ProductAttributeRepository using stateful mock data
type MockProductAttributeRepository struct {
	productattributepb.UnimplementedProductAttributeDomainServiceServer
	businessType      string
	productAttributes map[string]*productattributepb.ProductAttribute // Persistent in-memory store
	mutex             sync.RWMutex                                    // Thread-safe concurrent access
	initialized       bool                                            // Prevent double initialization
	skipInitialData   bool                                            // Option to skip loading baseline data
	processor         *listdata.ListDataProcessor                     // List data processor for filtering, sorting, searching, and pagination
}

// ProductAttributeRepositoryOption allows configuration of repository behavior
type ProductAttributeRepositoryOption func(*MockProductAttributeRepository)

// WithoutProductAttributeInitialData prevents the repository from loading the baseline mock data.
func WithoutProductAttributeInitialData() ProductAttributeRepositoryOption {
	return func(r *MockProductAttributeRepository) {
		r.skipInitialData = true
	}
}

// WithProductAttributeTestOptimizations enables test-specific optimizations
func WithProductAttributeTestOptimizations(enabled bool) ProductAttributeRepositoryOption {
	return func(r *MockProductAttributeRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockProductAttributeRepository creates a new mock product attribute repository
func NewMockProductAttributeRepository(businessType string, options ...ProductAttributeRepositoryOption) productattributepb.ProductAttributeDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockProductAttributeRepository{
		businessType:      businessType,
		productAttributes: make(map[string]*productattributepb.ProductAttribute),
		processor:         listdata.NewListDataProcessor(),
	}

	// Apply optional configurations
	for _, option := range options {
		option(repo)
	}

	// Initialize with mock data once
	if !repo.skipInitialData {
		if err := repo.loadInitialData(); err != nil {
			// Log error but don't fail - allows graceful degradation
			fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
		}
	}

	return repo
}

// loadInitialData loads mock data from the copya package
func (r *MockProductAttributeRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawProductAttributes, err := datamock.LoadBusinessTypeModule(r.businessType, "product-attribute")
	if err != nil {
		return fmt.Errorf("failed to load initial product attributes: %w", err)
	}

	// Convert and store each product attribute
	for _, rawProductAttribute := range rawProductAttributes {
		if productAttribute, err := r.mapToProtobufProductAttribute(rawProductAttribute); err == nil {
			// Use proper primary ID from protobuf model
			if productAttribute.Id != "" {
				r.productAttributes[productAttribute.Id] = productAttribute
			}
		}
	}

	r.initialized = true
	return nil
}

// CreateProductAttribute creates a new product attribute with stateful storage
func (r *MockProductAttributeRepository) CreateProductAttribute(ctx context.Context, req *productattributepb.CreateProductAttributeRequest) (*productattributepb.CreateProductAttributeResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create product attribute request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("product attribute data is required")
	}
	if req.Data.ProductId == "" {
		return nil, fmt.Errorf("product ID is required")
	}
	if req.Data.AttributeId == "" {
		return nil, fmt.Errorf("attribute ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()

	// Generate ID if not provided
	id := req.Data.Id
	if id == "" {
		id = fmt.Sprintf("mock-product-attr-%d", time.Now().UnixNano())
	}

	productAttribute := &productattributepb.ProductAttribute{
		Id:                 id,
		ProductId:          req.Data.ProductId,
		AttributeId:        req.Data.AttributeId,
		Value:              req.Data.Value,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
	}

	// Store in persistent map using proper primary ID
	r.productAttributes[productAttribute.Id] = productAttribute

	return &productattributepb.CreateProductAttributeResponse{
		Data:    []*productattributepb.ProductAttribute{productAttribute},
		Success: true,
	}, nil
}

// ReadProductAttribute retrieves a product attribute by primary ID from stateful storage
func (r *MockProductAttributeRepository) ReadProductAttribute(ctx context.Context, req *productattributepb.ReadProductAttributeRequest) (*productattributepb.ReadProductAttributeResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("read product attribute request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product attribute ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Use proper primary ID
	id := req.Data.Id

	// Retrieve from persistent storage
	productAttribute, exists := r.productAttributes[id]
	if !exists {
		return nil, fmt.Errorf("product attribute with ID '%s' not found", id)
	}

	return &productattributepb.ReadProductAttributeResponse{
		Data:    []*productattributepb.ProductAttribute{productAttribute},
		Success: true,
	}, nil
}

// UpdateProductAttribute updates an existing product attribute in stateful storage
func (r *MockProductAttributeRepository) UpdateProductAttribute(ctx context.Context, req *productattributepb.UpdateProductAttributeRequest) (*productattributepb.UpdateProductAttributeResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("update product attribute request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product attribute ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Use proper primary ID
	id := req.Data.Id

	// Check if product attribute exists
	existingProductAttribute, exists := r.productAttributes[id]
	if !exists {
		return nil, fmt.Errorf("product attribute with ID '%s' not found", id)
	}

	// Update with new modification timestamp
	now := time.Now()
	updatedProductAttribute := &productattributepb.ProductAttribute{
		Id:                 id,
		ProductId:          req.Data.ProductId,
		AttributeId:        req.Data.AttributeId,
		Value:              req.Data.Value,
		DateCreated:        existingProductAttribute.DateCreated,       // Preserve original creation date
		DateCreatedString:  existingProductAttribute.DateCreatedString, // Preserve original creation date
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
	}

	// Store updated product attribute
	r.productAttributes[id] = updatedProductAttribute

	return &productattributepb.UpdateProductAttributeResponse{
		Data:    []*productattributepb.ProductAttribute{updatedProductAttribute},
		Success: true,
	}, nil
}

// DeleteProductAttribute deletes a product attribute from stateful storage
func (r *MockProductAttributeRepository) DeleteProductAttribute(ctx context.Context, req *productattributepb.DeleteProductAttributeRequest) (*productattributepb.DeleteProductAttributeResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("delete product attribute request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product attribute ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Use proper primary ID
	id := req.Data.Id

	// Check if product attribute exists
	_, exists := r.productAttributes[id]
	if !exists {
		return nil, fmt.Errorf("product attribute with ID '%s' not found", id)
	}

	// Remove from persistent storage
	delete(r.productAttributes, id)

	return &productattributepb.DeleteProductAttributeResponse{
		Success: true,
	}, nil
}

// ListProductAttributes retrieves all product attributes from stateful storage
func (r *MockProductAttributeRepository) ListProductAttributes(ctx context.Context, req *productattributepb.ListProductAttributesRequest) (*productattributepb.ListProductAttributesResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice
	productAttributes := make([]*productattributepb.ProductAttribute, 0, len(r.productAttributes))
	for _, productAttribute := range r.productAttributes {
		productAttributes = append(productAttributes, productAttribute)
	}

	return &productattributepb.ListProductAttributesResponse{
		Data:    productAttributes,
		Success: true,
	}, nil
}

// mapToProtobufProductAttribute converts raw mock data to protobuf ProductAttribute
func (r *MockProductAttributeRepository) mapToProtobufProductAttribute(rawProductAttribute map[string]any) (*productattributepb.ProductAttribute, error) {
	productAttribute := &productattributepb.ProductAttribute{}

	// Map ID field (generate if missing)
	if id, ok := rawProductAttribute["id"].(string); ok {
		productAttribute.Id = id
	} else {
		// Generate ID if missing from mock data
		productAttribute.Id = fmt.Sprintf("mock-product-attr-%d", time.Now().UnixNano())
	}

	// Map required fields
	if productId, ok := rawProductAttribute["productId"].(string); ok {
		productAttribute.ProductId = productId
	} else {
		return nil, fmt.Errorf("missing or invalid productId field")
	}

	if attributeId, ok := rawProductAttribute["attributeId"].(string); ok {
		productAttribute.AttributeId = attributeId
	} else {
		return nil, fmt.Errorf("missing or invalid attributeId field")
	}

	if value, ok := rawProductAttribute["value"].(string); ok {
		productAttribute.Value = value
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawProductAttribute["dateCreated"].(string); ok {
		productAttribute.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			productAttribute.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawProductAttribute["dateModified"].(string); ok {
		productAttribute.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			productAttribute.DateModified = &timestamp
		}
	}

	return productAttribute, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockProductAttributeRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// GetProductAttributeListPageData retrieves product attributes with advanced filtering, sorting, searching, and pagination
func (r *MockProductAttributeRepository) GetProductAttributeListPageData(
	ctx context.Context,
	req *productattributepb.GetProductAttributeListPageDataRequest,
) (*productattributepb.GetProductAttributeListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product attribute list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of product attributes
	productAttributes := make([]*productattributepb.ProductAttribute, 0, len(r.productAttributes))
	for _, productAttribute := range r.productAttributes {
		productAttributes = append(productAttributes, productAttribute)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		productAttributes,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process product attribute list data: %w", err)
	}

	// Convert processed items back to product attribute protobuf format
	processedProductAttributes := make([]*productattributepb.ProductAttribute, len(result.Items))
	for i, item := range result.Items {
		if productAttribute, ok := item.(*productattributepb.ProductAttribute); ok {
			processedProductAttributes[i] = productAttribute
		} else {
			return nil, fmt.Errorf("failed to convert item to product attribute type")
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

	return &productattributepb.GetProductAttributeListPageDataResponse{
		ProductAttributeList: processedProductAttributes,
		Pagination:           result.PaginationResponse,
		SearchResults:        searchResults,
		Success:              true,
	}, nil
}

// GetProductAttributeItemPageData retrieves a single product attribute with enhanced item page data
func (r *MockProductAttributeRepository) GetProductAttributeItemPageData(
	ctx context.Context,
	req *productattributepb.GetProductAttributeItemPageDataRequest,
) (*productattributepb.GetProductAttributeItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product attribute item page data request is required")
	}
	if req.ProductAttributeId == "" {
		return nil, fmt.Errorf("product attribute ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	productAttribute, exists := r.productAttributes[req.ProductAttributeId]
	if !exists {
		return nil, fmt.Errorf("product attribute with ID '%s' not found", req.ProductAttributeId)
	}

	// In a real implementation, you might:
	// 1. Load related data (product details, attribute details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &productattributepb.GetProductAttributeItemPageDataResponse{
		ProductAttribute: productAttribute,
		Success:          true,
	}, nil
}

// NewProductAttributeRepository creates a new mock product attribute repository (registry constructor)
func NewProductAttributeRepository(data map[string]*productattributepb.ProductAttribute) productattributepb.ProductAttributeDomainServiceServer {
	repo := &MockProductAttributeRepository{
		businessType:      "education", // Default business type
		productAttributes: data,
		mutex:             sync.RWMutex{},
	}
	if data == nil {
		repo.productAttributes = make(map[string]*productattributepb.ProductAttribute)
		// Initialize with mock data
		if err := repo.loadInitialData(); err != nil {
			fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
		}
	}
	return repo
}

func init() {
	registry.RegisterRepositoryFactory("mock", "product_attribute", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockProductAttributeRepository(businessType), nil
	})
}
