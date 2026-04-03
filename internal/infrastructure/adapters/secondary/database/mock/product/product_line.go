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
	productlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_line"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
)

// MockProductLineRepository implements product.ProductLineRepository using stateful mock data
type MockProductLineRepository struct {
	productlinepb.UnimplementedProductLineDomainServiceServer
	businessType       string
	productLines map[string]*productlinepb.ProductLine // Persistent in-memory store
	mutex              sync.RWMutex                                      // Thread-safe concurrent access
	initialized        bool                                              // Prevent double initialization
	processor          *listdata.ListDataProcessor                      // List data processing utilities
}

// ProductLineRepositoryOption allows configuration of repository behavior
type ProductLineRepositoryOption func(*MockProductLineRepository)

// WithProductLineTestOptimizations enables test-specific optimizations
func WithProductLineTestOptimizations(enabled bool) ProductLineRepositoryOption {
	return func(r *MockProductLineRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockProductLineRepository creates a new mock product line repository
func NewMockProductLineRepository(businessType string, options ...ProductLineRepositoryOption) productlinepb.ProductLineDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockProductLineRepository{
		businessType:       businessType,
		productLines: make(map[string]*productlinepb.ProductLine),
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
func (r *MockProductLineRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawProductLines, err := datamock.LoadBusinessTypeModule(r.businessType, "product-line")
	if err != nil {
		return fmt.Errorf("failed to load initial product lines: %w", err)
	}

	// Convert and store each product line
	for _, rawProductLine := range rawProductLines {
		if productLine, err := r.mapToProtobufProductLine(rawProductLine); err == nil {
			r.productLines[productLine.Id] = productLine
		}
	}

	r.initialized = true
	return nil
}

// CreateProductLine creates a new product line with stateful storage
func (r *MockProductLineRepository) CreateProductLine(ctx context.Context, req *productlinepb.CreateProductLineRequest) (*productlinepb.CreateProductLineResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create product line request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("product line data is required")
	}
	if req.Data.ProductId == "" {
		return nil, fmt.Errorf("product ID is required")
	}
	if req.Data.LineId == "" {
		return nil, fmt.Errorf("line ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	productLineID := fmt.Sprintf("product-line-%d", now.UnixNano())

	productLine := &productlinepb.ProductLine{
		Id:                 productLineID,
		ProductId:          req.Data.ProductId,
		LineId:       req.Data.LineId,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true,
	}

	// Store in persistent map
	r.productLines[productLineID] = productLine

	return &productlinepb.CreateProductLineResponse{
		Data:    []*productlinepb.ProductLine{productLine},
		Success: true,
	}, nil
}

// ReadProductLine retrieves a product line by ID from stateful storage
func (r *MockProductLineRepository) ReadProductLine(ctx context.Context, req *productlinepb.ReadProductLineRequest) (*productlinepb.ReadProductLineResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("read product line request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product line ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Retrieve from persistent storage
	productLine, exists := r.productLines[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("product line with ID '%s' not found", req.Data.Id)
	}

	return &productlinepb.ReadProductLineResponse{
		Data:    []*productlinepb.ProductLine{productLine},
		Success: true,
	}, nil
}

// UpdateProductLine updates an existing product line in stateful storage
func (r *MockProductLineRepository) UpdateProductLine(ctx context.Context, req *productlinepb.UpdateProductLineRequest) (*productlinepb.UpdateProductLineResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("update product line request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product line ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if product line exists
	existingProductLine, exists := r.productLines[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("product line with ID '%s' not found", req.Data.Id)
	}

	// Update with new modification timestamp
	now := time.Now()
	updatedProductLine := &productlinepb.ProductLine{
		Id:                 req.Data.Id,
		ProductId:          req.Data.ProductId,
		LineId:       req.Data.LineId,
		DateCreated:        existingProductLine.DateCreated,       // Preserve original creation date
		DateCreatedString:  existingProductLine.DateCreatedString, // Preserve original creation date
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Store updated product line
	r.productLines[req.Data.Id] = updatedProductLine

	return &productlinepb.UpdateProductLineResponse{
		Data:    []*productlinepb.ProductLine{updatedProductLine},
		Success: true,
	}, nil
}

// DeleteProductLine deletes a product line from stateful storage
func (r *MockProductLineRepository) DeleteProductLine(ctx context.Context, req *productlinepb.DeleteProductLineRequest) (*productlinepb.DeleteProductLineResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("delete product line request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product line ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if product line exists
	_, exists := r.productLines[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("product line with ID '%s' not found", req.Data.Id)
	}

	// Remove from persistent storage
	delete(r.productLines, req.Data.Id)

	return &productlinepb.DeleteProductLineResponse{
		Success: true,
	}, nil
}

// ListProductLines retrieves all product lines from stateful storage
func (r *MockProductLineRepository) ListProductLines(ctx context.Context, req *productlinepb.ListProductLinesRequest) (*productlinepb.ListProductLinesResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice
	productLines := make([]*productlinepb.ProductLine, 0, len(r.productLines))
	for _, productLine := range r.productLines {
		productLines = append(productLines, productLine)
	}

	return &productlinepb.ListProductLinesResponse{
		Data:    productLines,
		Success: true,
	}, nil
}

// mapToProtobufProductLine converts raw mock data to protobuf ProductLine
func (r *MockProductLineRepository) mapToProtobufProductLine(rawProductLine map[string]any) (*productlinepb.ProductLine, error) {
	productLine := &productlinepb.ProductLine{}

	// Map required fields
	if id, ok := rawProductLine["id"].(string); ok {
		productLine.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if productId, ok := rawProductLine["productId"].(string); ok {
		productLine.ProductId = productId
	}

	if lineId, ok := rawProductLine["lineId"].(string); ok {
		productLine.LineId = lineId
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawProductLine["dateCreated"].(string); ok {
		productLine.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			productLine.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawProductLine["dateModified"].(string); ok {
		productLine.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			productLine.DateModified = &timestamp
		}
	}

	if active, ok := rawProductLine["active"].(bool); ok {
		productLine.Active = active
	}

	return productLine, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockProductLineRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// GetProductLineListPageData retrieves product lines with advanced filtering, sorting, searching, and pagination
func (r *MockProductLineRepository) GetProductLineListPageData(
	ctx context.Context,
	req *productlinepb.GetProductLineListPageDataRequest,
) (*productlinepb.GetProductLineListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product line list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of product lines
	productLines := make([]*productlinepb.ProductLine, 0, len(r.productLines))
	for _, productLine := range r.productLines {
		productLines = append(productLines, productLine)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		productLines,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process product line list data: %w", err)
	}

	// Convert processed items back to product line protobuf format
	processedProductLines := make([]*productlinepb.ProductLine, len(result.Items))
	for i, item := range result.Items {
		if productLine, ok := item.(*productlinepb.ProductLine); ok {
			processedProductLines[i] = productLine
		} else {
			return nil, fmt.Errorf("failed to convert item to product line type")
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

	return &productlinepb.GetProductLineListPageDataResponse{
		ProductLineList: processedProductLines,
		Pagination:            result.PaginationResponse,
		SearchResults:         searchResults,
		Success:               true,
	}, nil
}

// GetProductLineItemPageData retrieves a single product line with enhanced item page data
func (r *MockProductLineRepository) GetProductLineItemPageData(
	ctx context.Context,
	req *productlinepb.GetProductLineItemPageDataRequest,
) (*productlinepb.GetProductLineItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product line item page data request is required")
	}
	if req.ProductLineId == "" {
		return nil, fmt.Errorf("product line ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	productLine, exists := r.productLines[req.ProductLineId]
	if !exists {
		return nil, fmt.Errorf("product line with ID '%s' not found", req.ProductLineId)
	}

	// In a real implementation, you might:
	// 1. Load related data (product details, line details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &productlinepb.GetProductLineItemPageDataResponse{
		ProductLine: productLine,
		Success:           true,
	}, nil
}

// NewProductLineRepository creates a new mock product line repository (registry constructor)
func NewProductLineRepository(data map[string]*productlinepb.ProductLine) productlinepb.ProductLineDomainServiceServer {
	repo := &MockProductLineRepository{
		businessType:       "education", // Default business type
		productLines: data,
		mutex:              sync.RWMutex{},
		processor:          listdata.NewListDataProcessor(),
	}
	if data == nil {
		repo.productLines = make(map[string]*productlinepb.ProductLine)
		// Initialize with mock data
		if err := repo.loadInitialData(); err != nil {
			fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
		}
	}
	return repo
}

func init() {
	registry.RegisterRepositoryFactory("mock", entityid.ProductLine, func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockProductLineRepository(businessType), nil
	})
}
