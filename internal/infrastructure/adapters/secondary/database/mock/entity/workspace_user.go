//go:build mock_db

package entity

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
	workspaceuserpb "leapfor.xyz/esqyma/golang/v1/domain/entity/workspace_user"
)

// MockWorkspaceUserRepository implements entity.WorkspaceUserRepository using stateful mock data
type MockWorkspaceUserRepository struct {
	workspaceuserpb.UnimplementedWorkspaceUserDomainServiceServer
	businessType   string
	workspaceUsers map[string]*workspaceuserpb.WorkspaceUser // Persistent in-memory store
	mutex          sync.RWMutex                              // Thread-safe concurrent access
	initialized    bool                                      // Prevent double initialization
	processor      *listdata.ListDataProcessor               // List data processing utilities
}

// WorkspaceUserRepositoryOption allows configuration of repository behavior
type WorkspaceUserRepositoryOption func(*MockWorkspaceUserRepository)

// WithWorkspaceUserTestOptimizations enables test-specific optimizations
func WithWorkspaceUserTestOptimizations(enabled bool) WorkspaceUserRepositoryOption {
	return func(r *MockWorkspaceUserRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockWorkspaceUserRepository creates a new mock workspace user repository
func NewMockWorkspaceUserRepository(businessType string, options ...WorkspaceUserRepositoryOption) workspaceuserpb.WorkspaceUserDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockWorkspaceUserRepository{
		businessType:   businessType,
		workspaceUsers: make(map[string]*workspaceuserpb.WorkspaceUser),
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
func (r *MockWorkspaceUserRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawWorkspaceUsers, err := datamock.LoadWorkspaceUsers(r.businessType)
	if err != nil {
		return fmt.Errorf("failed to load initial workspace users: %w", err)
	}

	// Convert and store each workspace user
	for _, rawWorkspaceUser := range rawWorkspaceUsers {
		if workspaceUser, err := r.mapToProtobufWorkspaceUser(rawWorkspaceUser); err == nil {
			r.workspaceUsers[workspaceUser.Id] = workspaceUser
		}
	}

	r.initialized = true
	return nil
}

// CreateWorkspaceUser creates a new workspace user relationship with stateful storage
func (r *MockWorkspaceUserRepository) CreateWorkspaceUser(ctx context.Context, req *workspaceuserpb.CreateWorkspaceUserRequest) (*workspaceuserpb.CreateWorkspaceUserResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create workspace user request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("workspace user data is required")
	}
	if req.Data.WorkspaceId == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}
	if req.Data.UserId == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	workspaceUserID := fmt.Sprintf("workspace-user-%d-%d", now.UnixNano(), len(r.workspaceUsers))

	// Create new workspace user with proper timestamps and defaults
	newWorkspaceUser := &workspaceuserpb.WorkspaceUser{
		Id:                 workspaceUserID,
		WorkspaceId:        req.Data.WorkspaceId,
		UserId:             req.Data.UserId,
		Workspace:          req.Data.Workspace,
		User:               req.Data.User,
		WorkspaceUserRoles: req.Data.WorkspaceUserRoles,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.workspaceUsers[workspaceUserID] = newWorkspaceUser

	return &workspaceuserpb.CreateWorkspaceUserResponse{
		Data:    []*workspaceuserpb.WorkspaceUser{newWorkspaceUser},
		Success: true,
	}, nil
}

// ReadWorkspaceUser retrieves a workspace user relationship by ID from stateful storage
func (r *MockWorkspaceUserRepository) ReadWorkspaceUser(ctx context.Context, req *workspaceuserpb.ReadWorkspaceUserRequest) (*workspaceuserpb.ReadWorkspaceUserResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read workspace user request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated workspace users)
	if workspaceUser, exists := r.workspaceUsers[req.Data.Id]; exists {
		return &workspaceuserpb.ReadWorkspaceUserResponse{
			Data:    []*workspaceuserpb.WorkspaceUser{workspaceUser},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("workspace user with ID '%s' not found", req.Data.Id)
}

// UpdateWorkspaceUser updates an existing workspace user relationship in stateful storage
func (r *MockWorkspaceUserRepository) UpdateWorkspaceUser(ctx context.Context, req *workspaceuserpb.UpdateWorkspaceUserRequest) (*workspaceuserpb.UpdateWorkspaceUserResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update workspace user request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify workspace user exists
	existingWorkspaceUser, exists := r.workspaceUsers[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("workspace user with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedWorkspaceUser := &workspaceuserpb.WorkspaceUser{
		Id:                 req.Data.Id,
		WorkspaceId:        req.Data.WorkspaceId,
		UserId:             req.Data.UserId,
		Workspace:          req.Data.Workspace,
		User:               req.Data.User,
		WorkspaceUserRoles: req.Data.WorkspaceUserRoles,
		DateCreated:        existingWorkspaceUser.DateCreated,       // Preserve original
		DateCreatedString:  existingWorkspaceUser.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],            // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.workspaceUsers[req.Data.Id] = updatedWorkspaceUser

	return &workspaceuserpb.UpdateWorkspaceUserResponse{
		Data:    []*workspaceuserpb.WorkspaceUser{updatedWorkspaceUser},
		Success: true,
	}, nil
}

// DeleteWorkspaceUser deletes a workspace user relationship from stateful storage
func (r *MockWorkspaceUserRepository) DeleteWorkspaceUser(ctx context.Context, req *workspaceuserpb.DeleteWorkspaceUserRequest) (*workspaceuserpb.DeleteWorkspaceUserResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete workspace user request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify workspace user exists before deletion
	if _, exists := r.workspaceUsers[req.Data.Id]; !exists {
		return nil, fmt.Errorf("workspace user with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.workspaceUsers, req.Data.Id)

	return &workspaceuserpb.DeleteWorkspaceUserResponse{
		Success: true,
	}, nil
}

// ListWorkspaceUsers retrieves all workspace user relationships from stateful storage
func (r *MockWorkspaceUserRepository) ListWorkspaceUsers(ctx context.Context, req *workspaceuserpb.ListWorkspaceUsersRequest) (*workspaceuserpb.ListWorkspaceUsersResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of workspace users
	workspaceUsers := make([]*workspaceuserpb.WorkspaceUser, 0, len(r.workspaceUsers))
	for _, workspaceUser := range r.workspaceUsers {
		workspaceUsers = append(workspaceUsers, workspaceUser)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		workspaceUsers,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process workspace user list data: %w", err)
	}

	// Convert processed items back to workspace user protobuf format
	processedWorkspaceUsers := make([]*workspaceuserpb.WorkspaceUser, len(result.Items))
	for i, item := range result.Items {
		if workspaceUser, ok := item.(*workspaceuserpb.WorkspaceUser); ok {
			processedWorkspaceUsers[i] = workspaceUser
		} else {
			return nil, fmt.Errorf("failed to convert item to workspace user type")
		}
	}

	return &workspaceuserpb.ListWorkspaceUsersResponse{
		Data:    processedWorkspaceUsers,
		Success: true,
	}, nil
}

// GetWorkspaceUserListPageData retrieves workspace users with advanced filtering, sorting, searching, and pagination
func (r *MockWorkspaceUserRepository) GetWorkspaceUserListPageData(
	ctx context.Context,
	req *workspaceuserpb.GetWorkspaceUserListPageDataRequest,
) (*workspaceuserpb.GetWorkspaceUserListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get workspace user list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of workspace users
	workspaceUsers := make([]*workspaceuserpb.WorkspaceUser, 0, len(r.workspaceUsers))
	for _, workspaceUser := range r.workspaceUsers {
		workspaceUsers = append(workspaceUsers, workspaceUser)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		workspaceUsers,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process workspace user list data: %w", err)
	}

	// Convert processed items back to workspace user protobuf format
	processedWorkspaceUsers := make([]*workspaceuserpb.WorkspaceUser, len(result.Items))
	for i, item := range result.Items {
		if workspaceUser, ok := item.(*workspaceuserpb.WorkspaceUser); ok {
			processedWorkspaceUsers[i] = workspaceUser
		} else {
			return nil, fmt.Errorf("failed to convert item to workspace user type")
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

	return &workspaceuserpb.GetWorkspaceUserListPageDataResponse{
		WorkspaceUserList: processedWorkspaceUsers,
		Pagination:        result.PaginationResponse,
		SearchResults:     searchResults,
		Success:           true,
	}, nil
}

// GetWorkspaceUserItemPageData retrieves a single workspace user with enhanced item page data
func (r *MockWorkspaceUserRepository) GetWorkspaceUserItemPageData(
	ctx context.Context,
	req *workspaceuserpb.GetWorkspaceUserItemPageDataRequest,
) (*workspaceuserpb.GetWorkspaceUserItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get workspace user item page data request is required")
	}
	if req.WorkspaceUserId == "" {
		return nil, fmt.Errorf("workspace user ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	workspaceUser, exists := r.workspaceUsers[req.WorkspaceUserId]
	if !exists {
		return nil, fmt.Errorf("workspace user with ID '%s' not found", req.WorkspaceUserId)
	}

	// In a real implementation, you might:
	// 1. Load related data (workspace details, user details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &workspaceuserpb.GetWorkspaceUserItemPageDataResponse{
		WorkspaceUser: workspaceUser,
		Success:       true,
	}, nil
}

// mapToProtobufWorkspaceUser converts raw mock data to protobuf WorkspaceUser
func (r *MockWorkspaceUserRepository) mapToProtobufWorkspaceUser(rawWorkspaceUser map[string]any) (*workspaceuserpb.WorkspaceUser, error) {
	workspaceUser := &workspaceuserpb.WorkspaceUser{}

	// Map required fields
	if id, ok := rawWorkspaceUser["id"].(string); ok {
		workspaceUser.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if workspaceId, ok := rawWorkspaceUser["workspaceId"].(string); ok {
		workspaceUser.WorkspaceId = workspaceId
	} else {
		return nil, fmt.Errorf("missing or invalid workspaceId field")
	}

	if userId, ok := rawWorkspaceUser["userId"].(string); ok {
		workspaceUser.UserId = userId
	} else {
		return nil, fmt.Errorf("missing or invalid userId field")
	}

	// Note: Roles handling removed - WorkspaceUser now uses WorkspaceUserRoles junction table
	// Raw data conversion for roles should be handled separately if needed

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawWorkspaceUser["dateCreated"].(string); ok {
		workspaceUser.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			workspaceUser.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawWorkspaceUser["dateModified"].(string); ok {
		workspaceUser.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			workspaceUser.DateModified = &timestamp
		}
	}

	if active, ok := rawWorkspaceUser["active"].(bool); ok {
		workspaceUser.Active = active
	}

	return workspaceUser, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockWorkspaceUserRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewWorkspaceUserRepository creates a new workspace user repository - Provider interface compatibility
func NewWorkspaceUserRepository(businessType string) workspaceuserpb.WorkspaceUserDomainServiceServer {
	return NewMockWorkspaceUserRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock", "workspace_user", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockWorkspaceUserRepository(businessType), nil
	})
}
