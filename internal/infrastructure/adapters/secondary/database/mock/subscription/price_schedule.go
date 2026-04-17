//go:build mock_db

package subscription

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	datamock "leapfor.xyz/copya/golang"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
)

func init() {
	registry.RegisterRepositoryFactory("mock", entityid.PriceSchedule, func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockPriceScheduleRepository(businessType), nil
	})
}

// MockPriceScheduleRepository implements subscription.PriceScheduleRepository using stateful mock data
type MockPriceScheduleRepository struct {
	priceschedulepb.UnimplementedPriceScheduleDomainServiceServer
	businessType   string
	priceSchedules map[string]*priceschedulepb.PriceSchedule // Persistent in-memory store
	mutex          sync.RWMutex                              // Thread-safe concurrent access
	initialized    bool                                      // Prevent double initialization
	processor      *listdata.ListDataProcessor               // List data processing utilities
}

// PriceScheduleRepositoryOption allows configuration of repository behavior
type PriceScheduleRepositoryOption func(*MockPriceScheduleRepository)

// WithPriceScheduleTestOptimizations enables test-specific optimizations
func WithPriceScheduleTestOptimizations(enabled bool) PriceScheduleRepositoryOption {
	return func(r *MockPriceScheduleRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockPriceScheduleRepository creates a new mock price schedule repository
func NewMockPriceScheduleRepository(businessType string, options ...PriceScheduleRepositoryOption) priceschedulepb.PriceScheduleDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockPriceScheduleRepository{
		businessType:   businessType,
		priceSchedules: make(map[string]*priceschedulepb.PriceSchedule),
		processor:      listdata.NewListDataProcessor(),
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
func (r *MockPriceScheduleRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawPriceSchedules, err := datamock.LoadBusinessTypeModule(r.businessType, "price-schedule")
	if err != nil {
		return fmt.Errorf("failed to load initial price schedules: %w", err)
	}

	// Convert and store each price schedule
	for _, rawPriceSchedule := range rawPriceSchedules {
		if priceSchedule, err := r.mapToProtobufPriceSchedule(rawPriceSchedule); err == nil {
			r.priceSchedules[priceSchedule.Id] = priceSchedule
		}
	}

	r.initialized = true
	return nil
}

// CreatePriceSchedule creates a new price schedule with stateful storage
func (r *MockPriceScheduleRepository) CreatePriceSchedule(ctx context.Context, req *priceschedulepb.CreatePriceScheduleRequest) (*priceschedulepb.CreatePriceScheduleResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create price schedule request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("price schedule data is required")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("price schedule name is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	priceScheduleID := fmt.Sprintf("price-schedule-%d-%d", now.UnixNano(), len(r.priceSchedules))

	// Create new price schedule with proper timestamps and defaults
	newPriceSchedule := &priceschedulepb.PriceSchedule{
		Id:                 priceScheduleID,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}

	// Copy optional fields
	if req.Data.DateStart != nil {
		newPriceSchedule.DateStart = req.Data.DateStart
	}
	if req.Data.DateEnd != nil {
		newPriceSchedule.DateEnd = req.Data.DateEnd
	}
	if req.Data.LocationId != nil {
		newPriceSchedule.LocationId = req.Data.LocationId
	}

	// Store in persistent map
	r.priceSchedules[priceScheduleID] = newPriceSchedule

	return &priceschedulepb.CreatePriceScheduleResponse{
		Data:    []*priceschedulepb.PriceSchedule{newPriceSchedule},
		Success: true,
	}, nil
}

// ReadPriceSchedule retrieves a price schedule by ID from stateful storage
func (r *MockPriceScheduleRepository) ReadPriceSchedule(ctx context.Context, req *priceschedulepb.ReadPriceScheduleRequest) (*priceschedulepb.ReadPriceScheduleResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read price schedule request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price schedule ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated price schedules)
	if priceSchedule, exists := r.priceSchedules[req.Data.Id]; exists {
		return &priceschedulepb.ReadPriceScheduleResponse{
			Data:    []*priceschedulepb.PriceSchedule{priceSchedule},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("price schedule with ID '%s' not found", req.Data.Id)
}

// UpdatePriceSchedule updates an existing price schedule in stateful storage
func (r *MockPriceScheduleRepository) UpdatePriceSchedule(ctx context.Context, req *priceschedulepb.UpdatePriceScheduleRequest) (*priceschedulepb.UpdatePriceScheduleResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update price schedule request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price schedule ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify price schedule exists
	existingPriceSchedule, exists := r.priceSchedules[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("price schedule with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedPriceSchedule := &priceschedulepb.PriceSchedule{
		Id:                 req.Data.Id,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		DateStart:          req.Data.DateStart,
		DateEnd:            req.Data.DateEnd,
		LocationId:         req.Data.LocationId,
		DateCreated:        existingPriceSchedule.DateCreated,       // Preserve original
		DateCreatedString:  existingPriceSchedule.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],            // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.priceSchedules[req.Data.Id] = updatedPriceSchedule

	return &priceschedulepb.UpdatePriceScheduleResponse{
		Data:    []*priceschedulepb.PriceSchedule{updatedPriceSchedule},
		Success: true,
	}, nil
}

// DeletePriceSchedule deletes a price schedule from stateful storage
func (r *MockPriceScheduleRepository) DeletePriceSchedule(ctx context.Context, req *priceschedulepb.DeletePriceScheduleRequest) (*priceschedulepb.DeletePriceScheduleResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete price schedule request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("price schedule ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify price schedule exists before deletion
	if _, exists := r.priceSchedules[req.Data.Id]; !exists {
		return nil, fmt.Errorf("price schedule with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.priceSchedules, req.Data.Id)

	return &priceschedulepb.DeletePriceScheduleResponse{
		Success: true,
	}, nil
}

// ListPriceSchedules retrieves all price schedules from stateful storage
func (r *MockPriceScheduleRepository) ListPriceSchedules(ctx context.Context, req *priceschedulepb.ListPriceSchedulesRequest) (*priceschedulepb.ListPriceSchedulesResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of price schedules
	priceSchedules := make([]*priceschedulepb.PriceSchedule, 0, len(r.priceSchedules))
	for _, priceSchedule := range r.priceSchedules {
		priceSchedules = append(priceSchedules, priceSchedule)
	}

	return &priceschedulepb.ListPriceSchedulesResponse{
		Data:    priceSchedules,
		Success: true,
	}, nil
}

// GetPriceScheduleListPageData retrieves price schedules with advanced filtering, sorting, searching, and pagination
func (r *MockPriceScheduleRepository) GetPriceScheduleListPageData(
	ctx context.Context,
	req *priceschedulepb.GetPriceScheduleListPageDataRequest,
) (*priceschedulepb.GetPriceScheduleListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get price schedule list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of price schedules
	priceSchedules := make([]*priceschedulepb.PriceSchedule, 0, len(r.priceSchedules))
	for _, priceSchedule := range r.priceSchedules {
		priceSchedules = append(priceSchedules, priceSchedule)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		priceSchedules,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process price schedule list data: %w", err)
	}

	// Convert processed items back to price schedule protobuf format
	processedPriceSchedules := make([]*priceschedulepb.PriceSchedule, len(result.Items))
	for i, item := range result.Items {
		if priceSchedule, ok := item.(*priceschedulepb.PriceSchedule); ok {
			processedPriceSchedules[i] = priceSchedule
		} else {
			return nil, fmt.Errorf("failed to convert item to price schedule type")
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

	return &priceschedulepb.GetPriceScheduleListPageDataResponse{
		PriceScheduleList: processedPriceSchedules,
		Pagination:        result.PaginationResponse,
		SearchResults:     searchResults,
		Success:           true,
	}, nil
}

// GetPriceScheduleItemPageData retrieves a single price schedule with enhanced item page data
func (r *MockPriceScheduleRepository) GetPriceScheduleItemPageData(
	ctx context.Context,
	req *priceschedulepb.GetPriceScheduleItemPageDataRequest,
) (*priceschedulepb.GetPriceScheduleItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get price schedule item page data request is required")
	}
	if req.PriceScheduleId == "" {
		return nil, fmt.Errorf("price schedule ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	priceSchedule, exists := r.priceSchedules[req.PriceScheduleId]
	if !exists {
		return nil, fmt.Errorf("price schedule with ID '%s' not found", req.PriceScheduleId)
	}

	return &priceschedulepb.GetPriceScheduleItemPageDataResponse{
		PriceSchedule: priceSchedule,
		Success:       true,
	}, nil
}

// mapToProtobufPriceSchedule converts raw mock data to protobuf PriceSchedule
func (r *MockPriceScheduleRepository) mapToProtobufPriceSchedule(rawPriceSchedule map[string]any) (*priceschedulepb.PriceSchedule, error) {
	priceSchedule := &priceschedulepb.PriceSchedule{}

	// Map required fields
	if id, ok := rawPriceSchedule["id"].(string); ok {
		priceSchedule.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawPriceSchedule["name"].(string); ok {
		priceSchedule.Name = name
	} else {
		return nil, fmt.Errorf("missing or invalid name field")
	}

	// Map optional fields
	if description, ok := rawPriceSchedule["description"].(string); ok {
		priceSchedule.Description = description
	}

	if dateStart, ok := rawPriceSchedule["dateStart"].(string); ok {
		priceSchedule.DateStart = &dateStart
	}

	if dateEnd, ok := rawPriceSchedule["dateEnd"].(string); ok {
		priceSchedule.DateEnd = &dateEnd
	}

	if locationId, ok := rawPriceSchedule["locationId"].(string); ok {
		priceSchedule.LocationId = &locationId
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawPriceSchedule["dateCreated"].(string); ok {
		priceSchedule.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			priceSchedule.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawPriceSchedule["dateModified"].(string); ok {
		priceSchedule.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			priceSchedule.DateModified = &timestamp
		}
	}

	if active, ok := rawPriceSchedule["active"].(bool); ok {
		priceSchedule.Active = active
	}

	return priceSchedule, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockPriceScheduleRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewPriceScheduleRepository creates a new mock price schedule repository (registry constructor)
func NewPriceScheduleRepository(data map[string]*priceschedulepb.PriceSchedule) priceschedulepb.PriceScheduleDomainServiceServer {
	repo := &MockPriceScheduleRepository{
		businessType:   "education", // Default business type
		priceSchedules: data,
		mutex:          sync.RWMutex{},
		processor:      listdata.NewListDataProcessor(),
	}
	if data == nil {
		repo.priceSchedules = make(map[string]*priceschedulepb.PriceSchedule)
		// Initialize with mock data
		if err := repo.loadInitialData(); err != nil {
			fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
		}
	}
	return repo
}
