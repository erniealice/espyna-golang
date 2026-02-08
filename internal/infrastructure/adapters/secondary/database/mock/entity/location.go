//go:build mock_db

package entity

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	datamock "leapfor.xyz/copya/golang"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
)

// MockLocationRepository implements entity.LocationRepository using stateful mock data
type MockLocationRepository struct {
	locationpb.UnimplementedLocationDomainServiceServer
	businessType string
	locations    map[string]*locationpb.Location // Persistent in-memory store
	mutex        sync.RWMutex                    // Thread-safe concurrent access
	initialized  bool                            // Prevent double initialization
	processor    *listdata.ListDataProcessor     // List data processing capabilities
}

// LocationRepositoryOption allows configuration of repository behavior
type LocationRepositoryOption func(*MockLocationRepository)

// WithLocationTestOptimizations enables test-specific optimizations
func WithLocationTestOptimizations(enabled bool) LocationRepositoryOption {
	return func(r *MockLocationRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockLocationRepository creates a new mock location repository
func NewMockLocationRepository(businessType string, options ...LocationRepositoryOption) locationpb.LocationDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockLocationRepository{
		businessType: businessType,
		locations:    make(map[string]*locationpb.Location),
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
func (r *MockLocationRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawLocations, err := datamock.LoadLocations(r.businessType)
	if err != nil {
		return fmt.Errorf("failed to load initial locations: %w", err)
	}

	// Convert and store each location
	for _, rawLocation := range rawLocations {
		if location, err := r.mapToProtobufLocation(rawLocation); err == nil {
			r.locations[location.Id] = location
		}
	}

	r.initialized = true
	return nil
}

// CreateLocation creates a new location with stateful storage
func (r *MockLocationRepository) CreateLocation(ctx context.Context, req *locationpb.CreateLocationRequest) (*locationpb.CreateLocationResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create location request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("location data is required")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("location name is required")
	}
	if req.Data.Address == "" {
		return nil, fmt.Errorf("location address is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	locationID := fmt.Sprintf("location-%d-%d", now.UnixNano(), len(r.locations))

	// Create new location with proper timestamps and defaults
	newLocation := &locationpb.Location{
		Id:                 locationID,
		Name:               req.Data.Name,
		Address:            req.Data.Address,
		Description:        req.Data.Description,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.locations[locationID] = newLocation

	return &locationpb.CreateLocationResponse{
		Data:    []*locationpb.Location{newLocation},
		Success: true,
	}, nil
}

// ReadLocation retrieves a location by ID from stateful storage
func (r *MockLocationRepository) ReadLocation(ctx context.Context, req *locationpb.ReadLocationRequest) (*locationpb.ReadLocationResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read location request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated locations)
	if location, exists := r.locations[req.Data.Id]; exists {
		return &locationpb.ReadLocationResponse{
			Data:    []*locationpb.Location{location},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("location with ID '%s' not found", req.Data.Id)
}

// UpdateLocation updates an existing location in stateful storage
func (r *MockLocationRepository) UpdateLocation(ctx context.Context, req *locationpb.UpdateLocationRequest) (*locationpb.UpdateLocationResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update location request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify location exists
	existingLocation, exists := r.locations[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("location with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedLocation := &locationpb.Location{
		Id:                 req.Data.Id,
		Name:               req.Data.Name,
		Address:            req.Data.Address,
		Description:        req.Data.Description,
		DateCreated:        existingLocation.DateCreated,       // Preserve original
		DateCreatedString:  existingLocation.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],            // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.locations[req.Data.Id] = updatedLocation

	return &locationpb.UpdateLocationResponse{
		Data:    []*locationpb.Location{updatedLocation},
		Success: true,
	}, nil
}

// DeleteLocation deletes a location from stateful storage
func (r *MockLocationRepository) DeleteLocation(ctx context.Context, req *locationpb.DeleteLocationRequest) (*locationpb.DeleteLocationResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete location request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify location exists before deletion
	if _, exists := r.locations[req.Data.Id]; !exists {
		return nil, fmt.Errorf("location with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.locations, req.Data.Id)

	return &locationpb.DeleteLocationResponse{
		Success: true,
	}, nil
}

// ListLocations retrieves all locations from stateful storage
func (r *MockLocationRepository) ListLocations(ctx context.Context, req *locationpb.ListLocationsRequest) (*locationpb.ListLocationsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of locations
	items := make([]*locationpb.Location, 0, len(r.locations))
	for _, location := range r.locations {
		items = append(items, location)
	}

	// Process list data with processor
	result, err := r.processor.ProcessListRequest(
		items,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process list data: %w", err)
	}

	// Convert result.Items back to protobuf type
	processed := make([]*locationpb.Location, len(result.Items))
	for i, item := range result.Items {
		if typed, ok := item.(*locationpb.Location); ok {
			processed[i] = typed
		}
	}

	return &locationpb.ListLocationsResponse{
		Data:    processed,
		Success: true,
	}, nil
}

// GetLocationListPageData retrieves locations with advanced filtering, sorting, searching, and pagination
func (r *MockLocationRepository) GetLocationListPageData(
	ctx context.Context,
	req *locationpb.GetLocationListPageDataRequest,
) (*locationpb.GetLocationListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get location list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of locations
	locations := make([]*locationpb.Location, 0, len(r.locations))
	for _, location := range r.locations {
		locations = append(locations, location)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		locations,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process location list data: %w", err)
	}

	// Convert processed items back to location protobuf format
	processedLocations := make([]*locationpb.Location, len(result.Items))
	for i, item := range result.Items {
		if location, ok := item.(*locationpb.Location); ok {
			processedLocations[i] = location
		} else {
			return nil, fmt.Errorf("failed to convert item to location type")
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

	return &locationpb.GetLocationListPageDataResponse{
		LocationList:  processedLocations,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// GetLocationItemPageData retrieves a single location with enhanced item page data
func (r *MockLocationRepository) GetLocationItemPageData(
	ctx context.Context,
	req *locationpb.GetLocationItemPageDataRequest,
) (*locationpb.GetLocationItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get location item page data request is required")
	}
	if req.LocationId == "" {
		return nil, fmt.Errorf("location ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	location, exists := r.locations[req.LocationId]
	if !exists {
		return nil, fmt.Errorf("location with ID '%s' not found", req.LocationId)
	}

	// In a real implementation, you might:
	// 1. Load related data (related entities, attributes)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &locationpb.GetLocationItemPageDataResponse{
		Location: location,
		Success:  true,
	}, nil
}

// mapToProtobufLocation converts raw mock data to protobuf Location
func (r *MockLocationRepository) mapToProtobufLocation(rawLocation map[string]any) (*locationpb.Location, error) {
	location := &locationpb.Location{}

	// Map required fields
	if id, ok := rawLocation["id"].(string); ok {
		location.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawLocation["name"].(string); ok {
		location.Name = name
	} else {
		return nil, fmt.Errorf("missing or invalid name field")
	}

	if address, ok := rawLocation["address"].(string); ok {
		location.Address = address
	} else {
		return nil, fmt.Errorf("missing or invalid address field")
	}

	// Map optional fields
	if description, ok := rawLocation["description"].(string); ok {
		location.Description = &description
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawLocation["dateCreated"].(string); ok {
		location.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			location.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawLocation["dateModified"].(string); ok {
		location.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			location.DateModified = &timestamp
		}
	}

	if active, ok := rawLocation["active"].(bool); ok {
		location.Active = active
	}

	return location, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockLocationRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewLocationRepository creates a new location repository - Provider interface compatibility
func NewLocationRepository(businessType string) locationpb.LocationDomainServiceServer {
	return NewMockLocationRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock", "location", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockLocationRepository(businessType), nil
	})
}
