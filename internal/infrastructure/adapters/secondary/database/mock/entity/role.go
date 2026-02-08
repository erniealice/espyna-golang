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
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
)

// MockRoleRepository implements entity.RoleRepository using stateful mock data
type MockRoleRepository struct {
	rolepb.UnimplementedRoleDomainServiceServer
	businessType string
	roles        map[string]*rolepb.Role     // Persistent in-memory store
	mutex        sync.RWMutex                // Thread-safe concurrent access
	initialized  bool                        // Prevent double initialization
	processor    *listdata.ListDataProcessor // List data processing
}

// RoleRepositoryOption allows configuration of repository behavior
type RoleRepositoryOption func(*MockRoleRepository)

// WithRoleTestOptimizations enables test-specific optimizations
func WithRoleTestOptimizations(enabled bool) RoleRepositoryOption {
	return func(r *MockRoleRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockRoleRepository creates a new mock role repository
func NewMockRoleRepository(businessType string, options ...RoleRepositoryOption) rolepb.RoleDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockRoleRepository{
		businessType: businessType,
		roles:        make(map[string]*rolepb.Role),
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
func (r *MockRoleRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawRoles, err := datamock.LoadRoles(r.businessType)
	if err != nil {
		return fmt.Errorf("failed to load initial roles: %w", err)
	}

	// Convert and store each role
	for _, rawRole := range rawRoles {
		if role, err := r.mapToProtobufRole(rawRole); err == nil {
			r.roles[role.Id] = role
		}
	}

	r.initialized = true
	return nil
}

// CreateRole creates a new role with stateful storage
func (r *MockRoleRepository) CreateRole(ctx context.Context, req *rolepb.CreateRoleRequest) (*rolepb.CreateRoleResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create role request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("role data is required")
	}
	if req.Data.Name == "" {
		return nil, fmt.Errorf("role name is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	roleID := fmt.Sprintf("role-%d-%d", now.UnixNano(), len(r.roles))

	// Create new role with proper timestamps and defaults
	newRole := &rolepb.Role{
		Id:                 roleID,
		WorkspaceId:        req.Data.WorkspaceId,
		Workspace:          req.Data.Workspace,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		Color:              req.Data.Color,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.roles[roleID] = newRole

	return &rolepb.CreateRoleResponse{
		Data:    []*rolepb.Role{newRole},
		Success: true,
	}, nil
}

// ReadRole retrieves a role by ID from stateful storage
func (r *MockRoleRepository) ReadRole(ctx context.Context, req *rolepb.ReadRoleRequest) (*rolepb.ReadRoleResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read role request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated roles)
	if role, exists := r.roles[req.Data.Id]; exists {
		return &rolepb.ReadRoleResponse{
			Data:    []*rolepb.Role{role},
			Success: true,
		}, nil
	}

	return &rolepb.ReadRoleResponse{
		Data:    []*rolepb.Role{},
		Success: false,
	}, nil
}

// UpdateRole updates an existing role in stateful storage
func (r *MockRoleRepository) UpdateRole(ctx context.Context, req *rolepb.UpdateRoleRequest) (*rolepb.UpdateRoleResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update role request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify role exists
	existingRole, exists := r.roles[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("role with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedRole := &rolepb.Role{
		Id:                 req.Data.Id,
		WorkspaceId:        req.Data.WorkspaceId,
		Workspace:          req.Data.Workspace,
		Name:               req.Data.Name,
		Description:        req.Data.Description,
		Color:              req.Data.Color,
		DateCreated:        existingRole.DateCreated,       // Preserve original
		DateCreatedString:  existingRole.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],   // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.roles[req.Data.Id] = updatedRole

	return &rolepb.UpdateRoleResponse{
		Data:    []*rolepb.Role{updatedRole},
		Success: true,
	}, nil
}

// DeleteRole deletes a role from stateful storage
func (r *MockRoleRepository) DeleteRole(ctx context.Context, req *rolepb.DeleteRoleRequest) (*rolepb.DeleteRoleResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete role request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify role exists before deletion
	if _, exists := r.roles[req.Data.Id]; !exists {
		return nil, fmt.Errorf("role with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.roles, req.Data.Id)

	return &rolepb.DeleteRoleResponse{
		Success: true,
	}, nil
}

// ListRoles retrieves all roles from stateful storage
func (r *MockRoleRepository) ListRoles(ctx context.Context, req *rolepb.ListRolesRequest) (*rolepb.ListRolesResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of roles
	roles := make([]*rolepb.Role, 0, len(r.roles))
	for _, role := range r.roles {
		roles = append(roles, role)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		roles,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process role list data: %w", err)
	}

	// Convert processed items back to role protobuf format
	processedRoles := make([]*rolepb.Role, len(result.Items))
	for i, item := range result.Items {
		if role, ok := item.(*rolepb.Role); ok {
			processedRoles[i] = role
		} else {
			return nil, fmt.Errorf("failed to convert item to role type")
		}
	}

	return &rolepb.ListRolesResponse{
		Data:    processedRoles,
		Success: true,
	}, nil
}

// GetRoleListPageData retrieves roles with advanced filtering, sorting, searching, and pagination
func (r *MockRoleRepository) GetRoleListPageData(
	ctx context.Context,
	req *rolepb.GetRoleListPageDataRequest,
) (*rolepb.GetRoleListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get role list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of roles
	roles := make([]*rolepb.Role, 0, len(r.roles))
	for _, role := range r.roles {
		roles = append(roles, role)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		roles,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process role list data: %w", err)
	}

	// Convert processed items back to role protobuf format
	processedRoles := make([]*rolepb.Role, len(result.Items))
	for i, item := range result.Items {
		if role, ok := item.(*rolepb.Role); ok {
			processedRoles[i] = role
		} else {
			return nil, fmt.Errorf("failed to convert item to role type")
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

	return &rolepb.GetRoleListPageDataResponse{
		RoleList:      processedRoles,
		Pagination:    result.PaginationResponse,
		SearchResults: searchResults,
		Success:       true,
	}, nil
}

// GetRoleItemPageData retrieves a single role with enhanced item page data
func (r *MockRoleRepository) GetRoleItemPageData(
	ctx context.Context,
	req *rolepb.GetRoleItemPageDataRequest,
) (*rolepb.GetRoleItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get role item page data request is required")
	}
	if req.RoleId == "" {
		return nil, fmt.Errorf("role ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	role, exists := r.roles[req.RoleId]
	if !exists {
		return nil, fmt.Errorf("role with ID '%s' not found", req.RoleId)
	}

	// In a real implementation, you might:
	// 1. Load related data (workspace details, permission details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &rolepb.GetRoleItemPageDataResponse{
		Role:    role,
		Success: true,
	}, nil
}

// mapToProtobufRole converts raw mock data to protobuf Role
func (r *MockRoleRepository) mapToProtobufRole(rawRole map[string]any) (*rolepb.Role, error) {
	role := &rolepb.Role{}

	// Map required fields
	if id, ok := rawRole["id"].(string); ok {
		role.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if name, ok := rawRole["name"].(string); ok {
		role.Name = name
	} else {
		return nil, fmt.Errorf("missing or invalid name field")
	}

	// Map optional fields
	if workspaceId, ok := rawRole["workspaceId"].(string); ok {
		role.WorkspaceId = &workspaceId
	}

	if description, ok := rawRole["description"].(string); ok {
		role.Description = description
	}

	if color, ok := rawRole["color"].(string); ok {
		role.Color = color
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawRole["dateCreated"].(string); ok {
		role.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			role.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawRole["dateModified"].(string); ok {
		role.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			role.DateModified = &timestamp
		}
	}

	if active, ok := rawRole["active"].(bool); ok {
		role.Active = active
	}

	return role, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockRoleRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewRoleRepository creates a new role repository - Provider interface compatibility
func NewRoleRepository(businessType string) rolepb.RoleDomainServiceServer {
	return NewMockRoleRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock", "role", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockRoleRepository(businessType), nil
	})
}
