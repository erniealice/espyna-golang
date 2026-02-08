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
	locationattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_attribute"
)

// MockLocationAttributeRepository implements entity.LocationAttributeRepository using stateful mock data
type MockLocationAttributeRepository struct {
	locationattributepb.UnimplementedLocationAttributeDomainServiceServer
	businessType       string
	locationAttributes map[string]*locationattributepb.LocationAttribute // Persistent in-memory store
	mutex              sync.RWMutex                                      // Thread-safe concurrent access
	initialized        bool                                              // Prevent double initialization
	processor          *listdata.ListDataProcessor                       // List data processing
}

// LocationAttributeRepositoryOption allows configuration of repository behavior
type LocationAttributeRepositoryOption func(*MockLocationAttributeRepository)

// WithLocationAttributeTestOptimizations enables test-specific optimizations
func WithLocationAttributeTestOptimizations(enabled bool) LocationAttributeRepositoryOption {
	return func(r *MockLocationAttributeRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockLocationAttributeRepository creates a new mock location attribute repository
func NewMockLocationAttributeRepository(businessType string, options ...LocationAttributeRepositoryOption) locationattributepb.LocationAttributeDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockLocationAttributeRepository{
		businessType:       businessType,
		locationAttributes: make(map[string]*locationattributepb.LocationAttribute),
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
func (r *MockLocationAttributeRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawLocationAttributes, err := datamock.LoadLocationAttributes(r.businessType)
	if err != nil {
		return fmt.Errorf("failed to load initial location attributes: %w", err)
	}

	// Convert and store each location attribute
	for _, rawLocationAttribute := range rawLocationAttributes {
		if locationAttribute, err := r.mapToProtobufLocationAttribute(rawLocationAttribute); err == nil {
			// Use proper primary ID from protobuf model
			if locationAttribute.Id != "" {
				r.locationAttributes[locationAttribute.Id] = locationAttribute
			}
		}
	}

	r.initialized = true
	return nil
}

// CreateLocationAttribute creates a new location attribute with stateful storage
func (r *MockLocationAttributeRepository) CreateLocationAttribute(ctx context.Context, req *locationattributepb.CreateLocationAttributeRequest) (*locationattributepb.CreateLocationAttributeResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create location attribute request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("location attribute data is required")
	}
	if req.Data.LocationId == "" {
		return nil, fmt.Errorf("location ID is required")
	}
	if req.Data.AttributeId == "" {
		return nil, fmt.Errorf("attribute ID is required")
	}
	if req.Data.Value == "" {
		return nil, fmt.Errorf("value is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate ID if not provided
	id := req.Data.Id
	if id == "" {
		id = fmt.Sprintf("mock-location-attr-%d", time.Now().UnixNano())
	}

	// Create new location attribute with proper timestamps
	now := time.Now()
	newLocationAttribute := &locationattributepb.LocationAttribute{
		Id:                 id,
		LocationId:         req.Data.LocationId,
		AttributeId:        req.Data.AttributeId,
		Value:              req.Data.Value,
		Location:           req.Data.Location,
		Attribute:          req.Data.Attribute,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
	}

	// Store in persistent map using proper primary ID
	r.locationAttributes[newLocationAttribute.Id] = newLocationAttribute

	return &locationattributepb.CreateLocationAttributeResponse{
		Data:    []*locationattributepb.LocationAttribute{newLocationAttribute},
		Success: true,
	}, nil
}

// ReadLocationAttribute retrieves a location attribute by primary ID from stateful storage
func (r *MockLocationAttributeRepository) ReadLocationAttribute(ctx context.Context, req *locationattributepb.ReadLocationAttributeRequest) (*locationattributepb.ReadLocationAttributeResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read location attribute request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location attribute ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Use proper primary ID
	id := req.Data.Id
	if locationAttribute, exists := r.locationAttributes[id]; exists {
		return &locationattributepb.ReadLocationAttributeResponse{
			Data:    []*locationattributepb.LocationAttribute{locationAttribute},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("location attribute with ID '%s' not found", id)
}

// UpdateLocationAttribute updates an existing location attribute in stateful storage
func (r *MockLocationAttributeRepository) UpdateLocationAttribute(ctx context.Context, req *locationattributepb.UpdateLocationAttributeRequest) (*locationattributepb.UpdateLocationAttributeResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update location attribute request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location attribute ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Use proper primary ID
	id := req.Data.Id
	existingLocationAttribute, exists := r.locationAttributes[id]
	if !exists {
		return nil, fmt.Errorf("location attribute with ID '%s' not found", id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedLocationAttribute := &locationattributepb.LocationAttribute{
		Id:                 id,
		LocationId:         req.Data.LocationId,
		AttributeId:        req.Data.AttributeId,
		Value:              req.Data.Value,
		Location:           req.Data.Location,
		Attribute:          req.Data.Attribute,
		DateCreated:        existingLocationAttribute.DateCreated,       // Preserve original
		DateCreatedString:  existingLocationAttribute.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],                // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
	}

	// Update in persistent store
	r.locationAttributes[id] = updatedLocationAttribute

	return &locationattributepb.UpdateLocationAttributeResponse{
		Data:    []*locationattributepb.LocationAttribute{updatedLocationAttribute},
		Success: true,
	}, nil
}

// DeleteLocationAttribute deletes a location attribute from stateful storage
func (r *MockLocationAttributeRepository) DeleteLocationAttribute(ctx context.Context, req *locationattributepb.DeleteLocationAttributeRequest) (*locationattributepb.DeleteLocationAttributeResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete location attribute request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("location attribute ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Use proper primary ID
	id := req.Data.Id
	if _, exists := r.locationAttributes[id]; !exists {
		return nil, fmt.Errorf("location attribute with ID '%s' not found", id)
	}

	// Perform actual deletion from persistent store
	delete(r.locationAttributes, id)

	return &locationattributepb.DeleteLocationAttributeResponse{
		Success: true,
	}, nil
}

// ListLocationAttributes retrieves all location attributes from stateful storage
func (r *MockLocationAttributeRepository) ListLocationAttributes(ctx context.Context, req *locationattributepb.ListLocationAttributesRequest) (*locationattributepb.ListLocationAttributesResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of location attributes
	locationAttributes := make([]*locationattributepb.LocationAttribute, 0, len(r.locationAttributes))
	for _, locationAttribute := range r.locationAttributes {
		locationAttributes = append(locationAttributes, locationAttribute)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		locationAttributes,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process location attribute list data: %w", err)
	}

	// Convert processed items back to location attribute protobuf format
	processedLocationAttributes := make([]*locationattributepb.LocationAttribute, len(result.Items))
	for i, item := range result.Items {
		if locationAttribute, ok := item.(*locationattributepb.LocationAttribute); ok {
			processedLocationAttributes[i] = locationAttribute
		} else {
			return nil, fmt.Errorf("failed to convert item to location attribute type")
		}
	}

	return &locationattributepb.ListLocationAttributesResponse{
		Data:    processedLocationAttributes,
		Success: true,
	}, nil
}

// GetLocationAttributeListPageData retrieves location attributes with advanced filtering, sorting, searching, and pagination
func (r *MockLocationAttributeRepository) GetLocationAttributeListPageData(
	ctx context.Context,
	req *locationattributepb.GetLocationAttributeListPageDataRequest,
) (*locationattributepb.GetLocationAttributeListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get location attribute list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of location attributes
	locationAttributes := make([]*locationattributepb.LocationAttribute, 0, len(r.locationAttributes))
	for _, locationAttribute := range r.locationAttributes {
		locationAttributes = append(locationAttributes, locationAttribute)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		locationAttributes,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process location attribute list data: %w", err)
	}

	// Convert processed items back to location attribute protobuf format
	processedLocationAttributes := make([]*locationattributepb.LocationAttribute, len(result.Items))
	for i, item := range result.Items {
		if locationAttribute, ok := item.(*locationattributepb.LocationAttribute); ok {
			processedLocationAttributes[i] = locationAttribute
		} else {
			return nil, fmt.Errorf("failed to convert item to location attribute type")
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

	return &locationattributepb.GetLocationAttributeListPageDataResponse{
		LocationAttributeList: processedLocationAttributes,
		Pagination:            result.PaginationResponse,
		SearchResults:         searchResults,
		Success:               true,
	}, nil
}

// GetLocationAttributeItemPageData retrieves a single location attribute with enhanced item page data
func (r *MockLocationAttributeRepository) GetLocationAttributeItemPageData(
	ctx context.Context,
	req *locationattributepb.GetLocationAttributeItemPageDataRequest,
) (*locationattributepb.GetLocationAttributeItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get location attribute item page data request is required")
	}
	if req.LocationAttributeId == "" {
		return nil, fmt.Errorf("location attribute ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	locationAttribute, exists := r.locationAttributes[req.LocationAttributeId]
	if !exists {
		return nil, fmt.Errorf("location attribute with ID '%s' not found", req.LocationAttributeId)
	}

	// In a real implementation, you might:
	// 1. Load related data (location details, attribute details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &locationattributepb.GetLocationAttributeItemPageDataResponse{
		LocationAttribute: locationAttribute,
		Success:           true,
	}, nil
}

// mapToProtobufLocationAttribute converts raw mock data to protobuf LocationAttribute
func (r *MockLocationAttributeRepository) mapToProtobufLocationAttribute(rawLocationAttribute map[string]any) (*locationattributepb.LocationAttribute, error) {
	locationAttribute := &locationattributepb.LocationAttribute{}

	// Map ID field (generate if missing)
	if id, ok := rawLocationAttribute["id"].(string); ok {
		locationAttribute.Id = id
	} else {
		// Generate ID if missing from mock data
		locationAttribute.Id = fmt.Sprintf("mock-location-attr-%d", time.Now().UnixNano())
	}

	// Map required fields
	if locationId, ok := rawLocationAttribute["locationId"].(string); ok {
		locationAttribute.LocationId = locationId
	} else {
		return nil, fmt.Errorf("missing or invalid locationId field")
	}

	if attributeId, ok := rawLocationAttribute["attributeId"].(string); ok {
		locationAttribute.AttributeId = attributeId
	} else {
		return nil, fmt.Errorf("missing or invalid attributeId field")
	}

	if value, ok := rawLocationAttribute["value"].(string); ok {
		locationAttribute.Value = value
	} else {
		return nil, fmt.Errorf("missing or invalid value field")
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawLocationAttribute["dateCreated"].(string); ok {
		locationAttribute.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			locationAttribute.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawLocationAttribute["dateModified"].(string); ok {
		locationAttribute.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			locationAttribute.DateModified = &timestamp
		}
	}

	return locationAttribute, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockLocationAttributeRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewLocationAttributeRepository creates a new location attribute repository - Provider interface compatibility
func NewLocationAttributeRepository(businessType string) locationattributepb.LocationAttributeDomainServiceServer {
	return NewMockLocationAttributeRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock", "location_attribute", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockLocationAttributeRepository(businessType), nil
	})
}
