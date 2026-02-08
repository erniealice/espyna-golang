//go:build mock_db

package product

import (
	"context"
	"fmt"
	"sync"
	"time"

	datamock "leapfor.xyz/copya/golang"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	collectionplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_plan"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// MockCollectionPlanRepository implements product.CollectionPlanRepository using stateful mock data
type MockCollectionPlanRepository struct {
	collectionplanpb.UnimplementedCollectionPlanDomainServiceServer
	businessType    string
	collectionPlans map[string]*collectionplanpb.CollectionPlan // Persistent in-memory store
	mutex           sync.RWMutex                                // Thread-safe concurrent access
	initialized     bool                                        // Prevent double initialization
	skipInitialData bool                                        // Option to skip loading baseline data
	processor       *listdata.ListDataProcessor                 // List data processor for filtering, sorting, searching, and pagination
}

// CollectionPlanRepositoryOption allows configuration of repository behavior
type CollectionPlanRepositoryOption func(*MockCollectionPlanRepository)

// WithoutCollectionPlanInitialData prevents the repository from loading the baseline mock data.
func WithoutCollectionPlanInitialData() CollectionPlanRepositoryOption {
	return func(r *MockCollectionPlanRepository) {
		r.skipInitialData = true
	}
}

// WithCollectionPlanTestOptimizations enables test-specific optimizations
func WithCollectionPlanTestOptimizations(enabled bool) CollectionPlanRepositoryOption {
	return func(r *MockCollectionPlanRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockCollectionPlanRepository creates a new mock collection plan repository
func NewMockCollectionPlanRepository(businessType string, options ...CollectionPlanRepositoryOption) collectionplanpb.CollectionPlanDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockCollectionPlanRepository{
		businessType:    businessType,
		collectionPlans: make(map[string]*collectionplanpb.CollectionPlan),
		processor:       listdata.NewListDataProcessor(),
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
func (r *MockCollectionPlanRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawCollectionPlans, err := datamock.LoadBusinessTypeModule(r.businessType, "collection-plan")
	if err != nil {
		return fmt.Errorf("failed to load initial collection plans: %w", err)
	}

	// Convert and store each collection plan
	for _, rawCollectionPlan := range rawCollectionPlans {
		if collectionPlan, err := r.mapToProtobufCollectionPlan(rawCollectionPlan); err == nil {
			r.collectionPlans[collectionPlan.Id] = collectionPlan
		}
	}

	r.initialized = true
	return nil
}

// CreateCollectionPlan creates a new collection plan with stateful storage
func (r *MockCollectionPlanRepository) CreateCollectionPlan(ctx context.Context, req *collectionplanpb.CreateCollectionPlanRequest) (*collectionplanpb.CreateCollectionPlanResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create collection plan request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("collection plan data is required")
	}
	if req.Data.CollectionId == "" {
		return nil, fmt.Errorf("collection ID is required")
	}
	if req.Data.PlanId == "" {
		return nil, fmt.Errorf("plan ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	collectionPlanID := req.Data.Id
	if collectionPlanID == "" {
		// Generate unique ID with timestamp
		now := time.Now()
		collectionPlanID = fmt.Sprintf("collection-plan-%d-%d", now.UnixNano(), len(r.collectionPlans))
	}

	// Create new collection plan with proper timestamps and defaults
	newCollectionPlan := &collectionplanpb.CollectionPlan{
		Id:                 collectionPlanID,
		CollectionId:       req.Data.CollectionId,
		PlanId:             req.Data.PlanId,
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.collectionPlans[collectionPlanID] = newCollectionPlan

	return &collectionplanpb.CreateCollectionPlanResponse{
		Data:    []*collectionplanpb.CollectionPlan{newCollectionPlan},
		Success: true,
	}, nil
}

// ReadCollectionPlan retrieves a collection plan by ID from stateful storage
func (r *MockCollectionPlanRepository) ReadCollectionPlan(ctx context.Context, req *collectionplanpb.ReadCollectionPlanRequest) (*collectionplanpb.ReadCollectionPlanResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read collection plan request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection plan ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated collection plans)
	if collectionPlan, exists := r.collectionPlans[req.Data.Id]; exists {
		return &collectionplanpb.ReadCollectionPlanResponse{
			Data:    []*collectionplanpb.CollectionPlan{collectionPlan},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("collection plan with ID '%s' not found", req.Data.Id)
}

// UpdateCollectionPlan updates an existing collection plan in stateful storage
func (r *MockCollectionPlanRepository) UpdateCollectionPlan(ctx context.Context, req *collectionplanpb.UpdateCollectionPlanRequest) (*collectionplanpb.UpdateCollectionPlanResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update collection plan request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection plan ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify collection plan exists
	existingCollectionPlan, exists := r.collectionPlans[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("collection plan with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedCollectionPlan := &collectionplanpb.CollectionPlan{
		Id:                 req.Data.Id,
		CollectionId:       req.Data.CollectionId,
		PlanId:             req.Data.PlanId,
		DateCreated:        existingCollectionPlan.DateCreated,       // Preserve original
		DateCreatedString:  existingCollectionPlan.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],                  // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.collectionPlans[req.Data.Id] = updatedCollectionPlan

	return &collectionplanpb.UpdateCollectionPlanResponse{
		Data:    []*collectionplanpb.CollectionPlan{updatedCollectionPlan},
		Success: true,
	}, nil
}

// DeleteCollectionPlan deletes a collection plan from stateful storage
func (r *MockCollectionPlanRepository) DeleteCollectionPlan(ctx context.Context, req *collectionplanpb.DeleteCollectionPlanRequest) (*collectionplanpb.DeleteCollectionPlanResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete collection plan request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection plan ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify collection plan exists before deletion
	if _, exists := r.collectionPlans[req.Data.Id]; !exists {
		return nil, fmt.Errorf("collection plan with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.collectionPlans, req.Data.Id)

	return &collectionplanpb.DeleteCollectionPlanResponse{
		Success: true,
	}, nil
}

// ListCollectionPlans retrieves all collection plans from stateful storage
func (r *MockCollectionPlanRepository) ListCollectionPlans(ctx context.Context, req *collectionplanpb.ListCollectionPlansRequest) (*collectionplanpb.ListCollectionPlansResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of collection plans
	collectionPlans := make([]*collectionplanpb.CollectionPlan, 0, len(r.collectionPlans))
	for _, collectionPlan := range r.collectionPlans {
		collectionPlans = append(collectionPlans, collectionPlan)
	}

	return &collectionplanpb.ListCollectionPlansResponse{
		Data:    collectionPlans,
		Success: true,
	}, nil
}

// mapToProtobufCollectionPlan converts raw mock data to protobuf CollectionPlan
func (r *MockCollectionPlanRepository) mapToProtobufCollectionPlan(rawCollectionPlan map[string]any) (*collectionplanpb.CollectionPlan, error) {
	collectionPlan := &collectionplanpb.CollectionPlan{}

	// Map required fields
	if id, ok := rawCollectionPlan["id"].(string); ok {
		collectionPlan.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	// Map optional fields - add basic field mapping as needed
	if collectionId, ok := rawCollectionPlan["collectionId"].(string); ok {
		collectionPlan.CollectionId = collectionId
	}

	if planId, ok := rawCollectionPlan["planId"].(string); ok {
		collectionPlan.PlanId = planId
	}

	// Note: Description field may not exist in the protobuf definition
	// Remove or comment out if not needed

	// Set default active status
	collectionPlan.Active = true

	return collectionPlan, nil
}

// GetCollectionPlanListPageData retrieves collection plans with advanced filtering, sorting, searching, and pagination
func (r *MockCollectionPlanRepository) GetCollectionPlanListPageData(
	ctx context.Context,
	req *collectionplanpb.GetCollectionPlanListPageDataRequest,
) (*collectionplanpb.GetCollectionPlanListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get collection plan list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of collection plans
	collectionPlans := make([]*collectionplanpb.CollectionPlan, 0, len(r.collectionPlans))
	for _, collectionPlan := range r.collectionPlans {
		collectionPlans = append(collectionPlans, collectionPlan)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		collectionPlans,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process collection plan list data: %w", err)
	}

	// Convert processed items back to collection plan protobuf format
	processedCollectionPlans := make([]*collectionplanpb.CollectionPlan, len(result.Items))
	for i, item := range result.Items {
		if collectionPlan, ok := item.(*collectionplanpb.CollectionPlan); ok {
			processedCollectionPlans[i] = collectionPlan
		} else {
			return nil, fmt.Errorf("failed to convert item to collection plan type")
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

	return &collectionplanpb.GetCollectionPlanListPageDataResponse{
		CollectionPlanList: processedCollectionPlans,
		Pagination:         result.PaginationResponse,
		SearchResults:      searchResults,
		Success:            true,
	}, nil
}

// GetCollectionPlanItemPageData retrieves a single collection plan with enhanced item page data
func (r *MockCollectionPlanRepository) GetCollectionPlanItemPageData(
	ctx context.Context,
	req *collectionplanpb.GetCollectionPlanItemPageDataRequest,
) (*collectionplanpb.GetCollectionPlanItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get collection plan item page data request is required")
	}
	if req.CollectionPlanId == "" {
		return nil, fmt.Errorf("collection plan ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	collectionPlan, exists := r.collectionPlans[req.CollectionPlanId]
	if !exists {
		return nil, fmt.Errorf("collection plan with ID '%s' not found", req.CollectionPlanId)
	}

	// In a real implementation, you might:
	// 1. Load related data (collection details, plan details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &collectionplanpb.GetCollectionPlanItemPageDataResponse{
		CollectionPlan: collectionPlan,
		Success:        true,
	}, nil
}

// NewCollectionPlanRepository creates a new mock collection plan repository (registry constructor)
func NewCollectionPlanRepository(data map[string]*collectionplanpb.CollectionPlan) collectionplanpb.CollectionPlanDomainServiceServer {
	repo := &MockCollectionPlanRepository{
		businessType:    "education", // Default business type
		collectionPlans: data,
		mutex:           sync.RWMutex{},
	}
	if data == nil {
		repo.collectionPlans = make(map[string]*collectionplanpb.CollectionPlan)
		// Initialize with mock data
		if err := repo.loadInitialData(); err != nil {
			fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
		}
	}
	return repo
}

// NewCollectionPlanRepositoryCompat creates a new collection plan repository - Provider interface compatibility
func NewCollectionPlanRepositoryCompat(businessType string) collectionplanpb.CollectionPlanDomainServiceServer {
	return NewMockCollectionPlanRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock", "collection_plan", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockCollectionPlanRepository(businessType), nil
	})
}
