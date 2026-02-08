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
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

// MockWorkspaceUserRoleRepository implements entity.WorkspaceUserRoleRepository using stateful mock data
type MockWorkspaceUserRoleRepository struct {
	workspaceuserrolepb.UnimplementedWorkspaceUserRoleDomainServiceServer
	businessType       string
	workspaceUserRoles map[string]*workspaceuserrolepb.WorkspaceUserRole // Persistent in-memory store
	mutex              sync.RWMutex                                      // Thread-safe concurrent access
	initialized        bool                                              // Prevent double initialization
	processor          *listdata.ListDataProcessor                       // List data processing utilities
}

// WorkspaceUserRoleRepositoryOption allows configuration of repository behavior
type WorkspaceUserRoleRepositoryOption func(*MockWorkspaceUserRoleRepository)

// WithWorkspaceUserRoleTestOptimizations enables test-specific optimizations
func WithWorkspaceUserRoleTestOptimizations(enabled bool) WorkspaceUserRoleRepositoryOption {
	return func(r *MockWorkspaceUserRoleRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockWorkspaceUserRoleRepository creates a new mock workspace user role repository
func NewMockWorkspaceUserRoleRepository(businessType string, options ...WorkspaceUserRoleRepositoryOption) workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockWorkspaceUserRoleRepository{
		businessType:       businessType,
		workspaceUserRoles: make(map[string]*workspaceuserrolepb.WorkspaceUserRole),
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
func (r *MockWorkspaceUserRoleRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawWorkspaceUserRoles, err := datamock.LoadWorkspaceUserRoles(r.businessType)
	if err != nil {
		return fmt.Errorf("failed to load initial workspace user roles: %w", err)
	}

	// Convert and store each workspace user role
	for _, rawWorkspaceUserRole := range rawWorkspaceUserRoles {
		if workspaceUserRole, err := r.mapToProtobufWorkspaceUserRole(rawWorkspaceUserRole); err == nil {
			r.workspaceUserRoles[workspaceUserRole.Id] = workspaceUserRole
		}
	}

	r.initialized = true
	return nil
}

// CreateWorkspaceUserRole creates a new workspace user role relationship with stateful storage
func (r *MockWorkspaceUserRoleRepository) CreateWorkspaceUserRole(ctx context.Context, req *workspaceuserrolepb.CreateWorkspaceUserRoleRequest) (*workspaceuserrolepb.CreateWorkspaceUserRoleResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create workspace user role request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("workspace user role data is required")
	}
	if req.Data.WorkspaceUserId == "" {
		return nil, fmt.Errorf("workspace user ID is required")
	}
	if req.Data.RoleId == "" {
		return nil, fmt.Errorf("role ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	workspaceUserRoleID := fmt.Sprintf("workspace-user-role-%d-%d", now.UnixNano(), len(r.workspaceUserRoles))

	// Create new workspace user role with proper timestamps and defaults
	newWorkspaceUserRole := &workspaceuserrolepb.WorkspaceUserRole{
		Id:                 workspaceUserRoleID,
		WorkspaceUserId:    req.Data.WorkspaceUserId,
		RoleId:             req.Data.RoleId,
		Role:               req.Data.Role,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.workspaceUserRoles[workspaceUserRoleID] = newWorkspaceUserRole

	return &workspaceuserrolepb.CreateWorkspaceUserRoleResponse{
		Data:    []*workspaceuserrolepb.WorkspaceUserRole{newWorkspaceUserRole},
		Success: true,
	}, nil
}

// ReadWorkspaceUserRole retrieves a workspace user role relationship by ID from stateful storage
func (r *MockWorkspaceUserRoleRepository) ReadWorkspaceUserRole(ctx context.Context, req *workspaceuserrolepb.ReadWorkspaceUserRoleRequest) (*workspaceuserrolepb.ReadWorkspaceUserRoleResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read workspace user role request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user role ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated workspace user roles)
	if workspaceUserRole, exists := r.workspaceUserRoles[req.Data.Id]; exists {
		return &workspaceuserrolepb.ReadWorkspaceUserRoleResponse{
			Data:    []*workspaceuserrolepb.WorkspaceUserRole{workspaceUserRole},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("workspace user role with ID '%s' not found", req.Data.Id)
}

// UpdateWorkspaceUserRole updates an existing workspace user role relationship in stateful storage
func (r *MockWorkspaceUserRoleRepository) UpdateWorkspaceUserRole(ctx context.Context, req *workspaceuserrolepb.UpdateWorkspaceUserRoleRequest) (*workspaceuserrolepb.UpdateWorkspaceUserRoleResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update workspace user role request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user role ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify workspace user role exists
	existingWorkspaceUserRole, exists := r.workspaceUserRoles[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("workspace user role with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedWorkspaceUserRole := &workspaceuserrolepb.WorkspaceUserRole{
		Id:                 req.Data.Id,
		WorkspaceUserId:    req.Data.WorkspaceUserId,
		RoleId:             req.Data.RoleId,
		Role:               req.Data.Role,
		DateCreated:        existingWorkspaceUserRole.DateCreated,       // Preserve original
		DateCreatedString:  existingWorkspaceUserRole.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],                // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.workspaceUserRoles[req.Data.Id] = updatedWorkspaceUserRole

	return &workspaceuserrolepb.UpdateWorkspaceUserRoleResponse{
		Data:    []*workspaceuserrolepb.WorkspaceUserRole{updatedWorkspaceUserRole},
		Success: true,
	}, nil
}

// DeleteWorkspaceUserRole deletes a workspace user role relationship from stateful storage
func (r *MockWorkspaceUserRoleRepository) DeleteWorkspaceUserRole(ctx context.Context, req *workspaceuserrolepb.DeleteWorkspaceUserRoleRequest) (*workspaceuserrolepb.DeleteWorkspaceUserRoleResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete workspace user role request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("workspace user role ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify workspace user role exists before deletion
	if _, exists := r.workspaceUserRoles[req.Data.Id]; !exists {
		return nil, fmt.Errorf("workspace user role with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.workspaceUserRoles, req.Data.Id)

	return &workspaceuserrolepb.DeleteWorkspaceUserRoleResponse{
		Success: true,
	}, nil
}

// ListWorkspaceUserRoles retrieves all workspace user role relationships from stateful storage
func (r *MockWorkspaceUserRoleRepository) ListWorkspaceUserRoles(ctx context.Context, req *workspaceuserrolepb.ListWorkspaceUserRolesRequest) (*workspaceuserrolepb.ListWorkspaceUserRolesResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of workspace user roles
	workspaceUserRoles := make([]*workspaceuserrolepb.WorkspaceUserRole, 0, len(r.workspaceUserRoles))
	for _, workspaceUserRole := range r.workspaceUserRoles {
		workspaceUserRoles = append(workspaceUserRoles, workspaceUserRole)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		workspaceUserRoles,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process workspace user role list data: %w", err)
	}

	// Convert processed items back to workspace user role protobuf format
	processedWorkspaceUserRoles := make([]*workspaceuserrolepb.WorkspaceUserRole, len(result.Items))
	for i, item := range result.Items {
		if workspaceUserRole, ok := item.(*workspaceuserrolepb.WorkspaceUserRole); ok {
			processedWorkspaceUserRoles[i] = workspaceUserRole
		} else {
			return nil, fmt.Errorf("failed to convert item to workspace user role type")
		}
	}

	return &workspaceuserrolepb.ListWorkspaceUserRolesResponse{
		Data:    processedWorkspaceUserRoles,
		Success: true,
	}, nil
}

// GetWorkspaceUserRoleListPageData retrieves workspace user roles with advanced filtering, sorting, searching, and pagination
func (r *MockWorkspaceUserRoleRepository) GetWorkspaceUserRoleListPageData(
	ctx context.Context,
	req *workspaceuserrolepb.GetWorkspaceUserRoleListPageDataRequest,
) (*workspaceuserrolepb.GetWorkspaceUserRoleListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get workspace user role list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of workspace user roles
	workspaceUserRoles := make([]*workspaceuserrolepb.WorkspaceUserRole, 0, len(r.workspaceUserRoles))
	for _, workspaceUserRole := range r.workspaceUserRoles {
		workspaceUserRoles = append(workspaceUserRoles, workspaceUserRole)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		workspaceUserRoles,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process workspace user role list data: %w", err)
	}

	// Convert processed items back to workspace user role protobuf format
	processedWorkspaceUserRoles := make([]*workspaceuserrolepb.WorkspaceUserRole, len(result.Items))
	for i, item := range result.Items {
		if workspaceUserRole, ok := item.(*workspaceuserrolepb.WorkspaceUserRole); ok {
			processedWorkspaceUserRoles[i] = workspaceUserRole
		} else {
			return nil, fmt.Errorf("failed to convert item to workspace user role type")
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

	return &workspaceuserrolepb.GetWorkspaceUserRoleListPageDataResponse{
		WorkspaceUserRoleList: processedWorkspaceUserRoles,
		Pagination:            result.PaginationResponse,
		SearchResults:         searchResults,
		Success:               true,
	}, nil
}

// GetWorkspaceUserRoleItemPageData retrieves a single workspace user role with enhanced item page data
func (r *MockWorkspaceUserRoleRepository) GetWorkspaceUserRoleItemPageData(
	ctx context.Context,
	req *workspaceuserrolepb.GetWorkspaceUserRoleItemPageDataRequest,
) (*workspaceuserrolepb.GetWorkspaceUserRoleItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get workspace user role item page data request is required")
	}
	if req.WorkspaceUserRoleId == "" {
		return nil, fmt.Errorf("workspace user role ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	workspaceUserRole, exists := r.workspaceUserRoles[req.WorkspaceUserRoleId]
	if !exists {
		return nil, fmt.Errorf("workspace user role with ID '%s' not found", req.WorkspaceUserRoleId)
	}

	// In a real implementation, you might:
	// 1. Load related data (workspace user details, role details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &workspaceuserrolepb.GetWorkspaceUserRoleItemPageDataResponse{
		WorkspaceUserRole: workspaceUserRole,
		Success:           true,
	}, nil
}

// mapToProtobufWorkspaceUserRole converts raw mock data to protobuf WorkspaceUserRole
func (r *MockWorkspaceUserRoleRepository) mapToProtobufWorkspaceUserRole(rawWorkspaceUserRole map[string]any) (*workspaceuserrolepb.WorkspaceUserRole, error) {
	workspaceUserRole := &workspaceuserrolepb.WorkspaceUserRole{}

	// Map required fields
	if id, ok := rawWorkspaceUserRole["id"].(string); ok {
		workspaceUserRole.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if workspaceUserId, ok := rawWorkspaceUserRole["workspaceUserId"].(string); ok {
		workspaceUserRole.WorkspaceUserId = workspaceUserId
	} else {
		return nil, fmt.Errorf("missing or invalid workspaceUserId field")
	}

	if roleId, ok := rawWorkspaceUserRole["roleId"].(string); ok {
		workspaceUserRole.RoleId = roleId
	} else {
		return nil, fmt.Errorf("missing or invalid roleId field")
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawWorkspaceUserRole["dateCreated"].(string); ok {
		workspaceUserRole.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			workspaceUserRole.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawWorkspaceUserRole["dateModified"].(string); ok {
		workspaceUserRole.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			workspaceUserRole.DateModified = &timestamp
		}
	}

	if active, ok := rawWorkspaceUserRole["active"].(bool); ok {
		workspaceUserRole.Active = active
	}

	return workspaceUserRole, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockWorkspaceUserRoleRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewWorkspaceUserRoleRepository creates a new workspace user role repository - Provider interface compatibility
func NewWorkspaceUserRoleRepository(businessType string) workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer {
	return NewMockWorkspaceUserRoleRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock", "workspace_user_role", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockWorkspaceUserRoleRepository(businessType), nil
	})
}
