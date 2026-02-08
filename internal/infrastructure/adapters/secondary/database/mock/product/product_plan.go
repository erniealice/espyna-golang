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
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// MockProductPlanRepository implements product.ProductPlanRepository using stateful mock data
type MockProductPlanRepository struct {
	productplanpb.UnimplementedProductPlanDomainServiceServer
	businessType string
	productPlans map[string]*productplanpb.ProductPlan // Persistent in-memory store
	mutex        sync.RWMutex                          // Thread-safe concurrent access
	initialized  bool                                  // Prevent double initialization
	processor    *listdata.ListDataProcessor           // List data processing utilities
}

// ProductPlanRepositoryOption allows configuration of repository behavior
type ProductPlanRepositoryOption func(*MockProductPlanRepository)

// WithProductPlanTestOptimizations enables test-specific optimizations
func WithProductPlanTestOptimizations(enabled bool) ProductPlanRepositoryOption {
	return func(r *MockProductPlanRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockProductPlanRepository creates a new mock product plan repository
func NewMockProductPlanRepository(businessType string, options ...ProductPlanRepositoryOption) productplanpb.ProductPlanDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockProductPlanRepository{
		businessType: businessType,
		productPlans: make(map[string]*productplanpb.ProductPlan),
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
func (r *MockProductPlanRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawProductPlans, err := datamock.LoadBusinessTypeModule(r.businessType, "product-plan")
	if err != nil {
		return fmt.Errorf("failed to load initial product plans: %w", err)
	}

	// Convert and store each product plan
	for _, rawProductPlan := range rawProductPlans {
		if productPlan, err := r.mapToProtobufProductPlan(rawProductPlan); err == nil {
			r.productPlans[productPlan.Id] = productPlan
		}
	}

	r.initialized = true
	return nil
}

// CreateProductPlan creates a new product plan with stateful storage
func (r *MockProductPlanRepository) CreateProductPlan(ctx context.Context, req *productplanpb.CreateProductPlanRequest) (*productplanpb.CreateProductPlanResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create product plan request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("product plan data is required")
	}
	if req.Data.ProductId == "" {
		return nil, fmt.Errorf("product ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	productPlanID := fmt.Sprintf("product-plan-%d", now.UnixNano())

	productPlan := &productplanpb.ProductPlan{
		Id:                 productPlanID,
		ProductId:          req.Data.ProductId,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		Price:              req.Data.Price,
		Currency:           req.Data.Currency,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true,
	}

	// Store in persistent map
	r.productPlans[productPlanID] = productPlan

	return &productplanpb.CreateProductPlanResponse{
		Data:    []*productplanpb.ProductPlan{productPlan},
		Success: true,
	}, nil
}

// ReadProductPlan retrieves a product plan by ID from stateful storage
func (r *MockProductPlanRepository) ReadProductPlan(ctx context.Context, req *productplanpb.ReadProductPlanRequest) (*productplanpb.ReadProductPlanResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("read product plan request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product plan ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Retrieve from persistent storage
	productPlan, exists := r.productPlans[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("product plan with ID '%s' not found", req.Data.Id)
	}

	return &productplanpb.ReadProductPlanResponse{
		Data:    []*productplanpb.ProductPlan{productPlan},
		Success: true,
	}, nil
}

// UpdateProductPlan updates an existing product plan in stateful storage
func (r *MockProductPlanRepository) UpdateProductPlan(ctx context.Context, req *productplanpb.UpdateProductPlanRequest) (*productplanpb.UpdateProductPlanResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("update product plan request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product plan ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if product plan exists
	existingProductPlan, exists := r.productPlans[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("product plan with ID '%s' not found", req.Data.Id)
	}

	// Update with new modification timestamp
	now := time.Now()
	updatedProductPlan := &productplanpb.ProductPlan{
		Id:                 req.Data.Id,
		ProductId:          req.Data.ProductId,
		DateCreated:        existingProductPlan.DateCreated,       // Preserve original creation date
		DateCreatedString:  existingProductPlan.DateCreatedString, // Preserve original creation date
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Store updated product plan
	r.productPlans[req.Data.Id] = updatedProductPlan

	return &productplanpb.UpdateProductPlanResponse{
		Data:    []*productplanpb.ProductPlan{updatedProductPlan},
		Success: true,
	}, nil
}

// DeleteProductPlan deletes a product plan from stateful storage
func (r *MockProductPlanRepository) DeleteProductPlan(ctx context.Context, req *productplanpb.DeleteProductPlanRequest) (*productplanpb.DeleteProductPlanResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("delete product plan request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("product plan ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if product plan exists
	_, exists := r.productPlans[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("product plan with ID '%s' not found", req.Data.Id)
	}

	// Remove from persistent storage
	delete(r.productPlans, req.Data.Id)

	return &productplanpb.DeleteProductPlanResponse{
		Success: true,
	}, nil
}

// ListProductPlans retrieves all product plans from stateful storage
func (r *MockProductPlanRepository) ListProductPlans(ctx context.Context, req *productplanpb.ListProductPlansRequest) (*productplanpb.ListProductPlansResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice
	productPlans := make([]*productplanpb.ProductPlan, 0, len(r.productPlans))
	for _, productPlan := range r.productPlans {
		productPlans = append(productPlans, productPlan)
	}

	return &productplanpb.ListProductPlansResponse{
		Data:    productPlans,
		Success: true,
	}, nil
}

// mapToProtobufProductPlan converts raw mock data to protobuf ProductPlan
func (r *MockProductPlanRepository) mapToProtobufProductPlan(rawProductPlan map[string]any) (*productplanpb.ProductPlan, error) {
	productPlan := &productplanpb.ProductPlan{}

	// Map required fields
	if id, ok := rawProductPlan["id"].(string); ok {
		productPlan.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if productId, ok := rawProductPlan["productId"].(string); ok {
		productPlan.ProductId = productId
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawProductPlan["dateCreated"].(string); ok {
		productPlan.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			productPlan.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawProductPlan["dateModified"].(string); ok {
		productPlan.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			productPlan.DateModified = &timestamp
		}
	}

	if active, ok := rawProductPlan["active"].(bool); ok {
		productPlan.Active = active
	}

	return productPlan, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockProductPlanRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// GetProductPlanListPageData retrieves product plans with advanced filtering, sorting, searching, and pagination
func (r *MockProductPlanRepository) GetProductPlanListPageData(
	ctx context.Context,
	req *productplanpb.GetProductPlanListPageDataRequest,
) (*productplanpb.GetProductPlanListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product plan list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of product plans
	productPlans := make([]*productplanpb.ProductPlan, 0, len(r.productPlans))
	for _, productPlan := range r.productPlans {
		productPlans = append(productPlans, productPlan)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		productPlans,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process product plan list data: %w", err)
	}

	// Convert processed items back to product plan protobuf format
	processedProductPlans := make([]*productplanpb.ProductPlan, len(result.Items))
	for i, item := range result.Items {
		if productPlan, ok := item.(*productplanpb.ProductPlan); ok {
			processedProductPlans[i] = productPlan
		} else {
			return nil, fmt.Errorf("failed to convert item to product plan type")
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

	return &productplanpb.GetProductPlanListPageDataResponse{
		ProductPlanList: processedProductPlans,
		Pagination:      result.PaginationResponse,
		SearchResults:   searchResults,
		Success:         true,
	}, nil
}

// GetProductPlanItemPageData retrieves a single product plan with enhanced item page data
func (r *MockProductPlanRepository) GetProductPlanItemPageData(
	ctx context.Context,
	req *productplanpb.GetProductPlanItemPageDataRequest,
) (*productplanpb.GetProductPlanItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get product plan item page data request is required")
	}
	if req.ProductPlanId == "" {
		return nil, fmt.Errorf("product plan ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	productPlan, exists := r.productPlans[req.ProductPlanId]
	if !exists {
		return nil, fmt.Errorf("product plan with ID '%s' not found", req.ProductPlanId)
	}

	// In a real implementation, you might:
	// 1. Load related data (product details, plan details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &productplanpb.GetProductPlanItemPageDataResponse{
		ProductPlan: productPlan,
		Success:     true,
	}, nil
}

// NewProductPlanRepository creates a new mock product plan repository (registry constructor)
func NewProductPlanRepository(data map[string]*productplanpb.ProductPlan) productplanpb.ProductPlanDomainServiceServer {
	repo := &MockProductPlanRepository{
		businessType: "education", // Default business type
		productPlans: data,
		mutex:        sync.RWMutex{},
		processor:    listdata.NewListDataProcessor(),
	}
	if data == nil {
		repo.productPlans = make(map[string]*productplanpb.ProductPlan)
		// Initialize with mock data
		if err := repo.loadInitialData(); err != nil {
			fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
		}
	}
	return repo
}

func init() {
	registry.RegisterRepositoryFactory("mock", "product_plan", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockProductPlanRepository(businessType), nil
	})
}
