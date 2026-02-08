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
	grouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/group"
)

// MockGroupRepository implements entity.GroupRepository using stateful mock data
type MockGroupRepository struct {
	grouppb.UnimplementedGroupDomainServiceServer
	businessType string
	groups       map[string]*grouppb.Group // Persistent in-memory store
	mutex        sync.RWMutex              // Thread-safe concurrent access
	initialized  bool                      // Prevent double initialization
	processor    *listdata.ListDataProcessor // List data processing capabilities
}

// GroupRepositoryOption allows configuration of repository behavior
type GroupRepositoryOption func(*MockGroupRepository)

// WithGroupTestOptimizations enables test-specific optimizations
func WithGroupTestOptimizations(enabled bool) GroupRepositoryOption {
	return func(r *MockGroupRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockGroupRepository creates a new mock group repository
func NewMockGroupRepository(businessType string, options ...GroupRepositoryOption) grouppb.GroupDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockGroupRepository{
		businessType: businessType,
		groups:       make(map[string]*grouppb.Group),
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
func (r *MockGroupRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawGroups, err := datamock.LoadGroups(r.businessType)
	if err != nil {
		return fmt.Errorf("failed to load initial groups: %w", err)
	}

	// Convert and store each group
	for _, rawGroup := range rawGroups {
		if group, err := r.mapToProtobufGroup(rawGroup); err == nil {
			r.groups[group.Id] = group
		}
	}

	r.initialized = true
	return nil
}

// CreateGroup creates a new group with stateful storage
func (r *MockGroupRepository) CreateGroup(ctx context.Context, req *grouppb.CreateGroupRequest) (*grouppb.CreateGroupResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create group request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("group data is required")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("group name is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	groupID := fmt.Sprintf("group-%d-%d", now.UnixNano(), len(r.groups))

	// Create new group with proper timestamps and defaults
	newGroup := &grouppb.Group{
		Id:                 groupID,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.groups[groupID] = newGroup

	return &grouppb.CreateGroupResponse{
		Data:    []*grouppb.Group{newGroup},
		Success: true,
	}, nil
}

// ReadGroup retrieves a group by ID from stateful storage
func (r *MockGroupRepository) ReadGroup(ctx context.Context, req *grouppb.ReadGroupRequest) (*grouppb.ReadGroupResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read group request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("group ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated groups)
	if group, exists := r.groups[req.Data.Id]; exists {
		return &grouppb.ReadGroupResponse{
			Data:    []*grouppb.Group{group},
			Success: true,
		}, nil
	}

	// Return empty result for not found (no error)
	return &grouppb.ReadGroupResponse{
		Data:    []*grouppb.Group{},
		Success: true,
	}, nil
}

// UpdateGroup updates an existing group in stateful storage
func (r *MockGroupRepository) UpdateGroup(ctx context.Context, req *grouppb.UpdateGroupRequest) (*grouppb.UpdateGroupResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update group request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("group ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify group exists
	existingGroup, exists := r.groups[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("group with ID '%s' does not exist", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	updatedGroup := &grouppb.Group{
		Id:                 req.Data.Id,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		DateCreated:        existingGroup.DateCreated,       // Preserve original
		DateCreatedString:  existingGroup.DateCreatedString, // Preserve original
		DateModified:       req.Data.DateModified,           // Use timestamp from use case
		DateModifiedString: req.Data.DateModifiedString,     // Use formatted time from use case
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.groups[req.Data.Id] = updatedGroup

	return &grouppb.UpdateGroupResponse{
		Data:    []*grouppb.Group{updatedGroup},
		Success: true,
	}, nil
}

// DeleteGroup deletes a group from stateful storage
func (r *MockGroupRepository) DeleteGroup(ctx context.Context, req *grouppb.DeleteGroupRequest) (*grouppb.DeleteGroupResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete group request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("group ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify group exists before deletion
	if _, exists := r.groups[req.Data.Id]; !exists {
		return nil, fmt.Errorf("group with ID '%s' does not exist", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.groups, req.Data.Id)

	return &grouppb.DeleteGroupResponse{
		Success: true,
	}, nil
}

// ListGroups retrieves all groups from stateful storage
func (r *MockGroupRepository) ListGroups(ctx context.Context, req *grouppb.ListGroupsRequest) (*grouppb.ListGroupsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of groups
	items := make([]*grouppb.Group, 0, len(r.groups))
	for _, group := range r.groups {
		items = append(items, group)
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
	processed := make([]*grouppb.Group, len(result.Items))
	for i, item := range result.Items {
		if typed, ok := item.(*grouppb.Group); ok {
			processed[i] = typed
		}
	}

	return &grouppb.ListGroupsResponse{
		Data:    processed,
		Success: true,
	}, nil
}

// GetGroupListPageData retrieves groups with advanced filtering, sorting, searching, and pagination
func (r *MockGroupRepository) GetGroupListPageData(
	ctx context.Context,
	req *grouppb.GetGroupListPageDataRequest,
) (*grouppb.GetGroupListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get group list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of groups
	groups := make([]*grouppb.Group, 0, len(r.groups))
	for _, group := range r.groups {
		groups = append(groups, group)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		groups,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process group list data: %w", err)
	}

	// Convert processed items back to group protobuf format
	processedGroups := make([]*grouppb.Group, len(result.Items))
	for i, item := range result.Items {
		if group, ok := item.(*grouppb.Group); ok {
			processedGroups[i] = group
		} else {
			return nil, fmt.Errorf("failed to convert item to group type")
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

	return &grouppb.GetGroupListPageDataResponse{
		GroupList:     processedGroups,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// GetGroupItemPageData retrieves a single group with enhanced item page data
func (r *MockGroupRepository) GetGroupItemPageData(
	ctx context.Context,
	req *grouppb.GetGroupItemPageDataRequest,
) (*grouppb.GetGroupItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get group item page data request is required")
	}
	if req.GroupId == "" {
		return nil, fmt.Errorf("group ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	group, exists := r.groups[req.GroupId]
	if !exists {
		return nil, fmt.Errorf("group with ID '%s' not found", req.GroupId)
	}

	// In a real implementation, you might:
	// 1. Load related data (related entities, attributes)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &grouppb.GetGroupItemPageDataResponse{
		Group:   group,
		Success: true,
	}, nil
}

// mapToProtobufGroup converts raw mock data to protobuf Group
func (r *MockGroupRepository) mapToProtobufGroup(rawGroup map[string]any) (*grouppb.Group, error) {
	group := &grouppb.Group{}

	// Map required fields
	if id, ok := rawGroup["id"].(string); ok {
		group.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawGroup["name"].(string); ok {
		group.Name = name
	} else {
		return nil, fmt.Errorf("missing or invalid name field")
	}

	// Map optional fields
	if description, ok := rawGroup["description"].(string); ok {
		group.Description = description
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawGroup["dateCreated"].(string); ok {
		group.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			group.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawGroup["dateModified"].(string); ok {
		group.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			group.DateModified = &timestamp
		}
	}

	if active, ok := rawGroup["active"].(bool); ok {
		group.Active = active
	}

	return group, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockGroupRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewGroupRepository creates a new group repository - Provider interface compatibility
func NewGroupRepository(businessType string) grouppb.GroupDomainServiceServer {
	return NewMockGroupRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock", "group", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockGroupRepository(businessType), nil
	})
}
