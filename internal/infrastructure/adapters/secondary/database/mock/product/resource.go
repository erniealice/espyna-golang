//go:build mock_db

package product

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	datamock "leapfor.xyz/copya/golang"
	"leapfor.xyz/espyna/internal/application/shared/listdata"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	resourcepb "leapfor.xyz/esqyma/golang/v1/domain/product/resource"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
)

// MockResourceRepository implements product.ResourceRepository using stateful mock data
type MockResourceRepository struct {
	resourcepb.UnimplementedResourceDomainServiceServer
	businessType string
	resources    map[string]*resourcepb.Resource // Persistent in-memory store
	mutex        sync.RWMutex                    // Thread-safe concurrent access
	initialized  bool                            // Prevent double initialization
	processor    *listdata.ListDataProcessor     // List data processing utilities
}

// ResourceRepositoryOption allows configuration of repository behavior
type ResourceRepositoryOption func(*MockResourceRepository)

// WithResourceTestOptimizations enables test-specific optimizations
func WithResourceTestOptimizations(enabled bool) ResourceRepositoryOption {
	return func(r *MockResourceRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockResourceRepository creates a new mock resource repository
func NewMockResourceRepository(businessType string, options ...ResourceRepositoryOption) resourcepb.ResourceDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockResourceRepository{
		businessType: businessType,
		resources:    make(map[string]*resourcepb.Resource),
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
func (r *MockResourceRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawResources, err := datamock.LoadBusinessTypeModule(r.businessType, "resource")
	if err != nil {
		return fmt.Errorf("failed to load initial resources: %w", err)
	}

	// Convert and store each resource
	for _, rawResource := range rawResources {
		if resource, err := r.mapToProtobufResource(rawResource); err == nil {
			r.resources[resource.Id] = resource
		}
	}

	r.initialized = true
	return nil
}

// CreateResource creates a new resource with stateful storage
func (r *MockResourceRepository) CreateResource(ctx context.Context, req *resourcepb.CreateResourceRequest) (*resourcepb.CreateResourceResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create resource request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("resource data is required")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("resource name is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	resourceID := fmt.Sprintf("resource-%d", now.UnixNano())

	resource := &resourcepb.Resource{
		Id:                 resourceID,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		ProductId:          req.Data.ProductId,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true,
	}

	// Store in persistent map
	r.resources[resourceID] = resource

	return &resourcepb.CreateResourceResponse{
		Data:    []*resourcepb.Resource{resource},
		Success: true,
	}, nil
}

// ReadResource retrieves a resource by ID from stateful storage
func (r *MockResourceRepository) ReadResource(ctx context.Context, req *resourcepb.ReadResourceRequest) (*resourcepb.ReadResourceResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("read resource request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("resource ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Retrieve from persistent storage
	resource, exists := r.resources[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("resource with ID '%s' not found", req.Data.Id)
	}

	return &resourcepb.ReadResourceResponse{
		Data:    []*resourcepb.Resource{resource},
		Success: true,
	}, nil
}

// UpdateResource updates an existing resource in stateful storage
func (r *MockResourceRepository) UpdateResource(ctx context.Context, req *resourcepb.UpdateResourceRequest) (*resourcepb.UpdateResourceResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("update resource request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("resource ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if resource exists
	existingResource, exists := r.resources[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("resource with ID '%s' not found", req.Data.Id)
	}

	// Update with new modification timestamp
	now := time.Now()
	updatedResource := &resourcepb.Resource{
		Id:                 req.Data.Id,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		ProductId:          req.Data.ProductId,
		DateCreated:        existingResource.DateCreated,       // Preserve original creation date
		DateCreatedString:  existingResource.DateCreatedString, // Preserve original creation date
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Store updated resource
	r.resources[req.Data.Id] = updatedResource

	return &resourcepb.UpdateResourceResponse{
		Data:    []*resourcepb.Resource{updatedResource},
		Success: true,
	}, nil
}

// DeleteResource deletes a resource from stateful storage
func (r *MockResourceRepository) DeleteResource(ctx context.Context, req *resourcepb.DeleteResourceRequest) (*resourcepb.DeleteResourceResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("delete resource request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("resource ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if resource exists
	_, exists := r.resources[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("resource with ID '%s' not found", req.Data.Id)
	}

	// Remove from persistent storage
	delete(r.resources, req.Data.Id)

	return &resourcepb.DeleteResourceResponse{
		Success: true,
	}, nil
}

// ListResources retrieves all resources from stateful storage
func (r *MockResourceRepository) ListResources(ctx context.Context, req *resourcepb.ListResourcesRequest) (*resourcepb.ListResourcesResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice
	resources := make([]*resourcepb.Resource, 0, len(r.resources))
	for _, resource := range r.resources {
		resources = append(resources, resource)
	}

	return &resourcepb.ListResourcesResponse{
		Data:    resources,
		Success: true,
	}, nil
}

// mapToProtobufResource converts raw mock data to protobuf Resource
func (r *MockResourceRepository) mapToProtobufResource(rawResource map[string]any) (*resourcepb.Resource, error) {
	resource := &resourcepb.Resource{}

	if id, ok := rawResource["id"].(string); ok {
		resource.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawResource["name"].(string); ok {
		resource.Name = name
	}

	if productId, ok := rawResource["productId"].(string); ok {
		resource.ProductId = productId
	}

	if description, ok := rawResource["description"].(string); ok {
		resource.Description = &description
	}

	if dateCreated, ok := rawResource["dateCreated"].(string); ok {
		resource.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			resource.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawResource["dateModified"].(string); ok {
		resource.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			resource.DateModified = &timestamp
		}
	}

	if active, ok := rawResource["active"].(bool); ok {
		resource.Active = active
	}

	return resource, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockResourceRepository) parseTimestamp(timestampStr string) (int64, error) {
	if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
		return timestamp, nil
	}

	if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
		return t.UnixMilli(), nil
	}

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

// GetResourceListPageData retrieves resources with advanced filtering, sorting, searching, and pagination
func (r *MockResourceRepository) GetResourceListPageData(
	ctx context.Context,
	req *resourcepb.GetResourceListPageDataRequest,
) (*resourcepb.GetResourceListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get resource list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of resources
	resources := make([]*resourcepb.Resource, 0, len(r.resources))
	for _, resource := range r.resources {
		resources = append(resources, resource)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		resources,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process resource list data: %w", err)
	}

	// Convert processed items back to resource protobuf format
	processedResources := make([]*resourcepb.Resource, len(result.Items))
	for i, item := range result.Items {
		if resource, ok := item.(*resourcepb.Resource); ok {
			processedResources[i] = resource
		} else {
			return nil, fmt.Errorf("failed to convert item to resource type")
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

	return &resourcepb.GetResourceListPageDataResponse{
		ResourceList:  processedResources,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// GetResourceItemPageData retrieves a single resource with enhanced item page data
func (r *MockResourceRepository) GetResourceItemPageData(
	ctx context.Context,
	req *resourcepb.GetResourceItemPageDataRequest,
) (*resourcepb.GetResourceItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get resource item page data request is required")
	}
	if req.ResourceId == "" {
		return nil, fmt.Errorf("resource ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	resource, exists := r.resources[req.ResourceId]
	if !exists {
		return nil, fmt.Errorf("resource with ID '%s' not found", req.ResourceId)
	}

	// In a real implementation, you might:
	// 1. Load related data (product details, resource usage stats)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &resourcepb.GetResourceItemPageDataResponse{
		Resource: resource,
		Success:  true,
	}, nil
}

// NewResourceRepository creates a new mock resource repository (registry constructor)
func NewResourceRepository(data map[string]*resourcepb.Resource) resourcepb.ResourceDomainServiceServer {
	repo := &MockResourceRepository{
		businessType: "education", // Default business type
		resources:    data,
		mutex:        sync.RWMutex{},
		processor:    listdata.NewListDataProcessor(),
	}
	if data == nil {
		repo.resources = make(map[string]*resourcepb.Resource)
		// Initialize with mock data
		if err := repo.loadInitialData(); err != nil {
			fmt.Printf("Warning: Failed to load initial mock data: %v\n", err)
		}
	}
	return repo
}

func init() {
	registry.RegisterRepositoryFactory("mock", "resource", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockResourceRepository(businessType), nil
	})
}
