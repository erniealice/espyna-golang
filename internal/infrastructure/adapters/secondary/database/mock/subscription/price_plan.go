//go:build mock_db

package subscription

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	datamock "leapfor.xyz/copya/golang"
	"leapfor.xyz/espyna/internal/application/shared/listdata"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	priceplanpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/price_plan"
)

func init() {
	registry.RegisterRepositoryFactory("mock", "price_plan", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockPricePlanRepository(businessType), nil
	})
}

// MockPricePlanRepository implements subscription.PricePlanRepository using stateful mock data
type MockPricePlanRepository struct {
	priceplanpb.UnimplementedPricePlanDomainServiceServer
	businessType string
	pricePlans   map[string]*priceplanpb.PricePlan // Persistent in-memory store
	mutex        sync.RWMutex                      // Thread-safe concurrent access
	initialized  bool                              // Prevent double initialization
	processor    *listdata.ListDataProcessor       // List data processing utilities
}

// PricePlanRepositoryOption allows configuration of repository behavior
type PricePlanRepositoryOption func(*MockPricePlanRepository)

// WithPricePlanTestOptimizations enables test-specific optimizations
func WithPricePlanTestOptimizations(enabled bool) PricePlanRepositoryOption {
	return func(r *MockPricePlanRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockPricePlanRepository creates a new mock price plan repository
func NewMockPricePlanRepository(businessType string, options ...PricePlanRepositoryOption) priceplanpb.PricePlanDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockPricePlanRepository{
		businessType: businessType,
		pricePlans:   make(map[string]*priceplanpb.PricePlan),
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
func (r *MockPricePlanRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawPricePlans, err := datamock.LoadBusinessTypeModule(r.businessType, "price-plan")
	if err != nil {
		return fmt.Errorf("failed to load initial price plans: %w", err)
	}

	// Convert and store each price plan
	for _, rawPricePlan := range rawPricePlans {
		if pricePlan, err := r.mapToProtobufPricePlan(rawPricePlan); err == nil {
			r.pricePlans[pricePlan.Id] = pricePlan
		}
	}

	r.initialized = true
	return nil
}

// CreatePricePlan creates a new price plan with stateful storage
func (r *MockPricePlanRepository) CreatePricePlan(ctx context.Context, req *priceplanpb.CreatePricePlanRequest) (*priceplanpb.CreatePricePlanResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create price plan request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("price plan data is required")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("price plan name is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	pricePlanID := fmt.Sprintf("price-plan-%d-%d", now.UnixNano(), len(r.pricePlans))

	// Create new price plan with proper timestamps and defaults
	newPricePlan := &priceplanpb.PricePlan{
		Id:                 pricePlanID,
		PlanId:             req.Data.PlanId,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		Amount:             req.Data.Amount,
		Currency:           req.Data.Currency,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.pricePlans[pricePlanID] = newPricePlan

	return &priceplanpb.CreatePricePlanResponse{
		Data:    []*priceplanpb.PricePlan{newPricePlan},
		Success: true,
	}, nil
}

// ReadPricePlan retrieves a price plan by ID from stateful storage
func (r *MockPricePlanRepository) ReadPricePlan(ctx context.Context, req *priceplanpb.ReadPricePlanRequest) (*priceplanpb.ReadPricePlanResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read price plan request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price plan ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated price plans)
	if pricePlan, exists := r.pricePlans[req.Data.Id]; exists {
		return &priceplanpb.ReadPricePlanResponse{
			Data:    []*priceplanpb.PricePlan{pricePlan},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("price plan with ID '%s' not found", req.Data.Id)
}

// UpdatePricePlan updates an existing price plan in stateful storage
func (r *MockPricePlanRepository) UpdatePricePlan(ctx context.Context, req *priceplanpb.UpdatePricePlanRequest) (*priceplanpb.UpdatePricePlanResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update price plan request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price plan ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify price plan exists
	existingPricePlan, exists := r.pricePlans[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("price plan with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedPricePlan := &priceplanpb.PricePlan{
		Id:                 req.Data.Id,
		PlanId:             req.Data.PlanId,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		Amount:             req.Data.Amount,
		Currency:           req.Data.Currency,
		DateCreated:        existingPricePlan.DateCreated,       // Preserve original
		DateCreatedString:  existingPricePlan.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],             // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.pricePlans[req.Data.Id] = updatedPricePlan

	return &priceplanpb.UpdatePricePlanResponse{
		Data:    []*priceplanpb.PricePlan{updatedPricePlan},
		Success: true,
	}, nil
}

// DeletePricePlan deletes a price plan from stateful storage
func (r *MockPricePlanRepository) DeletePricePlan(ctx context.Context, req *priceplanpb.DeletePricePlanRequest) (*priceplanpb.DeletePricePlanResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete price plan request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price plan ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify price plan exists before deletion
	if _, exists := r.pricePlans[req.Data.Id]; !exists {
		return nil, fmt.Errorf("price plan with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.pricePlans, req.Data.Id)

	return &priceplanpb.DeletePricePlanResponse{
		Success: true,
	}, nil
}

// ListPricePlans retrieves all price plans from stateful storage
func (r *MockPricePlanRepository) ListPricePlans(ctx context.Context, req *priceplanpb.ListPricePlansRequest) (*priceplanpb.ListPricePlansResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of price plans
	pricePlans := make([]*priceplanpb.PricePlan, 0, len(r.pricePlans))
	for _, pricePlan := range r.pricePlans {
		pricePlans = append(pricePlans, pricePlan)
	}

	return &priceplanpb.ListPricePlansResponse{
		Data:    pricePlans,
		Success: true,
	}, nil
}

// GetPricePlanListPageData retrieves price plans with advanced filtering, sorting, searching, and pagination
func (r *MockPricePlanRepository) GetPricePlanListPageData(
	ctx context.Context,
	req *priceplanpb.GetPricePlanListPageDataRequest,
) (*priceplanpb.GetPricePlanListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get price plan list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of price plans
	pricePlans := make([]*priceplanpb.PricePlan, 0, len(r.pricePlans))
	for _, pricePlan := range r.pricePlans {
		pricePlans = append(pricePlans, pricePlan)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		pricePlans,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process price plan list data: %w", err)
	}

	// Convert processed items back to price plan protobuf format
	processedPricePlans := make([]*priceplanpb.PricePlan, len(result.Items))
	for i, item := range result.Items {
		if pricePlan, ok := item.(*priceplanpb.PricePlan); ok {
			processedPricePlans[i] = pricePlan
		} else {
			return nil, fmt.Errorf("failed to convert item to price plan type")
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

	return &priceplanpb.GetPricePlanListPageDataResponse{
		PricePlanList: processedPricePlans,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// GetPricePlanItemPageData retrieves a single price plan with enhanced item page data
func (r *MockPricePlanRepository) GetPricePlanItemPageData(
	ctx context.Context,
	req *priceplanpb.GetPricePlanItemPageDataRequest,
) (*priceplanpb.GetPricePlanItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get price plan item page data request is required")
	}
	if req.PricePlanId == "" {
		return nil, fmt.Errorf("price plan ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	pricePlan, exists := r.pricePlans[req.PricePlanId]
	if !exists {
		return nil, fmt.Errorf("price plan with ID '%s' not found", req.PricePlanId)
	}

	// In a real implementation, you might:
	// 1. Load related data (plan details, subscription details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &priceplanpb.GetPricePlanItemPageDataResponse{
		PricePlan: pricePlan,
		Success:   true,
	}, nil
}

// mapToProtobufPricePlan converts raw mock data to protobuf PricePlan
func (r *MockPricePlanRepository) mapToProtobufPricePlan(rawPricePlan map[string]any) (*priceplanpb.PricePlan, error) {
	pricePlan := &priceplanpb.PricePlan{}

	// Map required fields
	if id, ok := rawPricePlan["id"].(string); ok {
		pricePlan.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawPricePlan["name"].(string); ok {
		pricePlan.Name = name
	} else {
		return nil, fmt.Errorf("missing or invalid name field")
	}

	// Map optional fields
	if planId, ok := rawPricePlan["planId"].(string); ok {
		pricePlan.PlanId = planId
	}

	if description, ok := rawPricePlan["description"].(string); ok {
		pricePlan.Description = description
	}

	if amount, ok := rawPricePlan["amount"].(float64); ok {
		pricePlan.Amount = float64(amount)
	}

	if currency, ok := rawPricePlan["currency"].(string); ok {
		pricePlan.Currency = currency
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawPricePlan["dateCreated"].(string); ok {
		pricePlan.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			pricePlan.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawPricePlan["dateModified"].(string); ok {
		pricePlan.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			pricePlan.DateModified = &timestamp
		}
	}

	if active, ok := rawPricePlan["active"].(bool); ok {
		pricePlan.Active = active
	}

	return pricePlan, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockPricePlanRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewPricePlanRepository creates a new mock price plan repository (registry constructor)
func NewPricePlanRepository(data map[string]*priceplanpb.PricePlan) priceplanpb.PricePlanDomainServiceServer {
	repo := &MockPricePlanRepository{
		businessType: "education", // Default business type
		pricePlans:   data,
		mutex:        sync.RWMutex{},
		processor:    listdata.NewListDataProcessor(),
	}
	if data == nil {
		repo.pricePlans = make(map[string]*priceplanpb.PricePlan)
		// Initialize with mock data
		if err := repo.loadInitialData(); err != nil {
			fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
		}
	}
	return repo
}
