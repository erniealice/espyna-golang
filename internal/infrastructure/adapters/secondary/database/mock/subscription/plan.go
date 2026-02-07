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
	planpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan"
)

func init() {
	registry.RegisterRepositoryFactory("mock", "plan", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockPlanRepository(businessType), nil
	})
}

// MockPlanRepository implements subscription.PlanRepository using stateful mock data
type MockPlanRepository struct {
	planpb.UnimplementedPlanDomainServiceServer
	businessType string
	plans        map[string]*planpb.Plan // Persistent in-memory store
	mutex        sync.RWMutex            // Thread-safe concurrent access
	initialized  bool                    // Prevent double initialization
	processor    *listdata.ListDataProcessor // List data processing utilities
}

// PlanRepositoryOption allows configuration of repository behavior
type PlanRepositoryOption func(*MockPlanRepository)

// WithPlanTestOptimizations enables test-specific optimizations
func WithPlanTestOptimizations(enabled bool) PlanRepositoryOption {
	return func(r *MockPlanRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockPlanRepository creates a new mock plan repository
func NewMockPlanRepository(businessType string, options ...PlanRepositoryOption) planpb.PlanDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockPlanRepository{
		businessType: businessType,
		plans:        make(map[string]*planpb.Plan),
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
func (r *MockPlanRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawPlans, err := datamock.LoadBusinessTypeModule(r.businessType, "plan")
	if err != nil {
		return fmt.Errorf("failed to load initial plans: %w", err)
	}

	// Convert and store each plan
	for _, rawPlan := range rawPlans {
		if plan, err := r.mapToProtobufPlan(rawPlan); err == nil {
			if plan.Id != nil {
				r.plans[*plan.Id] = plan
			}
		}
	}

	r.initialized = true
	return nil
}

// CreatePlan creates a new plan with stateful storage
func (r *MockPlanRepository) CreatePlan(ctx context.Context, req *planpb.CreatePlanRequest) (*planpb.CreatePlanResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create plan request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("plan data is required")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("plan name is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	var planID string
	if req.Data.Id != nil {
		planID = *req.Data.Id
	}
	if planID == "" {
		// Generate unique ID with timestamp
		now := time.Now()
		planID = fmt.Sprintf("plan-%d-%d", now.UnixNano(), len(r.plans))
	}

	// Create new plan with proper timestamps and defaults
	newPlan := &planpb.Plan{
		Id:                 &planID,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.plans[planID] = newPlan

	return &planpb.CreatePlanResponse{
		Data:    []*planpb.Plan{newPlan},
		Success: true,
	}, nil
}

// ReadPlan retrieves a plan by ID from stateful storage
func (r *MockPlanRepository) ReadPlan(ctx context.Context, req *planpb.ReadPlanRequest) (*planpb.ReadPlanResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read plan request is required")
	}
	if req.Data == nil || req.Data.Id == nil || *req.Data.Id == "" {
		return nil, fmt.Errorf("plan ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated plans)
	planID := *req.Data.Id
	if plan, exists := r.plans[planID]; exists {
		return &planpb.ReadPlanResponse{
			Data:    []*planpb.Plan{plan},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("plan with ID '%s' not found", planID)
}

// UpdatePlan updates an existing plan in stateful storage
func (r *MockPlanRepository) UpdatePlan(ctx context.Context, req *planpb.UpdatePlanRequest) (*planpb.UpdatePlanResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update plan request is required")
	}
	if req.Data == nil || req.Data.Id == nil || *req.Data.Id == "" {
		return nil, fmt.Errorf("plan ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify plan exists
	planID := *req.Data.Id
	existingPlan, exists := r.plans[planID]
	if !exists {
		return nil, fmt.Errorf("plan with ID '%s' not found", planID)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedPlan := &planpb.Plan{
		Id:                 req.Data.Id,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		DateCreated:        existingPlan.DateCreated,       // Preserve original
		DateCreatedString:  existingPlan.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],    // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.plans[planID] = updatedPlan

	return &planpb.UpdatePlanResponse{
		Data:    []*planpb.Plan{updatedPlan},
		Success: true,
	}, nil
}

// DeletePlan deletes a plan from stateful storage
func (r *MockPlanRepository) DeletePlan(ctx context.Context, req *planpb.DeletePlanRequest) (*planpb.DeletePlanResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete plan request is required")
	}
	if req.Data == nil || req.Data.Id == nil || *req.Data.Id == "" {
		return nil, fmt.Errorf("plan ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify plan exists before deletion
	planID := *req.Data.Id
	if _, exists := r.plans[planID]; !exists {
		return nil, fmt.Errorf("plan with ID '%s' not found", planID)
	}

	// Perform actual deletion from persistent store
	delete(r.plans, planID)

	return &planpb.DeletePlanResponse{
		Success: true,
	}, nil
}

// ListPlans retrieves all plans from stateful storage
func (r *MockPlanRepository) ListPlans(ctx context.Context, req *planpb.ListPlansRequest) (*planpb.ListPlansResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of plans
	plans := make([]*planpb.Plan, 0, len(r.plans))
	for _, plan := range r.plans {
		plans = append(plans, plan)
	}

	return &planpb.ListPlansResponse{
		Data:    plans,
		Success: true,
	}, nil
}

// GetPlanListPageData retrieves plans with advanced filtering, sorting, searching, and pagination
func (r *MockPlanRepository) GetPlanListPageData(
	ctx context.Context,
	req *planpb.GetPlanListPageDataRequest,
) (*planpb.GetPlanListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get plan list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of plans
	plans := make([]*planpb.Plan, 0, len(r.plans))
	for _, plan := range r.plans {
		plans = append(plans, plan)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		plans,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process plan list data: %w", err)
	}

	// Convert processed items back to plan protobuf format
	processedPlans := make([]*planpb.Plan, len(result.Items))
	for i, item := range result.Items {
		if plan, ok := item.(*planpb.Plan); ok {
			processedPlans[i] = plan
		} else {
			return nil, fmt.Errorf("failed to convert item to plan type")
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

	return &planpb.GetPlanListPageDataResponse{
		PlanList:      processedPlans,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// GetPlanItemPageData retrieves a single plan with enhanced item page data
func (r *MockPlanRepository) GetPlanItemPageData(
	ctx context.Context,
	req *planpb.GetPlanItemPageDataRequest,
) (*planpb.GetPlanItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get plan item page data request is required")
	}
	if req.PlanId == "" {
		return nil, fmt.Errorf("plan ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	plan, exists := r.plans[req.PlanId]
	if !exists {
		return nil, fmt.Errorf("plan with ID '%s' not found", req.PlanId)
	}

	// In a real implementation, you might:
	// 1. Load related data (price plans, plan settings)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &planpb.GetPlanItemPageDataResponse{
		Plan:    plan,
		Success: true,
	}, nil
}

// mapToProtobufPlan converts raw mock data to protobuf Plan
func (r *MockPlanRepository) mapToProtobufPlan(rawPlan map[string]any) (*planpb.Plan, error) {
	plan := &planpb.Plan{}

	// Map required fields
	if id, ok := rawPlan["id"].(string); ok {
		plan.Id = &id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawPlan["name"].(string); ok {
		plan.Name = name
	} else {
		return nil, fmt.Errorf("missing or invalid name field")
	}

	// Map optional fields
	if description, ok := rawPlan["description"].(*string); ok {
		plan.Description = description
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawPlan["dateCreated"].(string); ok {
		plan.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			plan.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawPlan["dateModified"].(string); ok {
		plan.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			plan.DateModified = &timestamp
		}
	}

	if active, ok := rawPlan["active"].(bool); ok {
		plan.Active = active
	}

	return plan, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockPlanRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewPlanRepository creates a new mock plan repository (registry constructor)
func NewPlanRepository(data map[string]*planpb.Plan) planpb.PlanDomainServiceServer {
	repo := &MockPlanRepository{
		businessType: "education", // Default business type
		plans:        data,
		mutex:        sync.RWMutex{},
		processor:    listdata.NewListDataProcessor(),
	}
	if data == nil {
		repo.plans = make(map[string]*planpb.Plan)
		// Initialize with mock data
		if err := repo.loadInitialData(); err != nil {
			fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
		}
	}
	return repo
}
