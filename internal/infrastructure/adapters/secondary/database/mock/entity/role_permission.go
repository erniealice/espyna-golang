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
	permissionpb "leapfor.xyz/esqyma/golang/v1/domain/entity/permission"
	rolepermissionpb "leapfor.xyz/esqyma/golang/v1/domain/entity/role_permission"
)

// MockRolePermissionRepository implements entity.RolePermissionRepository using stateful mock data
type MockRolePermissionRepository struct {
	rolepermissionpb.UnimplementedRolePermissionDomainServiceServer
	businessType    string
	rolePermissions map[string]*rolepermissionpb.RolePermission // Persistent in-memory store
	mutex           sync.RWMutex                                // Thread-safe concurrent access
	initialized     bool                                        // Prevent double initialization
	processor       *listdata.ListDataProcessor                 // List data processing for pagination, filtering, sorting
}

// RolePermissionRepositoryOption allows configuration of repository behavior
type RolePermissionRepositoryOption func(*MockRolePermissionRepository)

// WithRolePermissionTestOptimizations enables test-specific optimizations
func WithRolePermissionTestOptimizations(enabled bool) RolePermissionRepositoryOption {
	return func(r *MockRolePermissionRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockRolePermissionRepository creates a new mock role permission repository
func NewMockRolePermissionRepository(businessType string, options ...RolePermissionRepositoryOption) rolepermissionpb.RolePermissionDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockRolePermissionRepository{
		businessType:    businessType,
		rolePermissions: make(map[string]*rolepermissionpb.RolePermission),
		processor:       listdata.NewListDataProcessor(),
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
func (r *MockRolePermissionRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawRolePermissions, err := datamock.LoadRolePermissions(r.businessType)
	if err != nil {
		return fmt.Errorf("failed to load initial role permissions: %w", err)
	}

	// Convert and store each role permission
	for _, rawRolePermission := range rawRolePermissions {
		if rolePermission, err := r.mapToProtobufRolePermission(rawRolePermission); err == nil {
			r.rolePermissions[rolePermission.Id] = rolePermission
		}
	}

	r.initialized = true
	return nil
}

// CreateRolePermission creates a new role permission relationship with stateful storage
func (r *MockRolePermissionRepository) CreateRolePermission(ctx context.Context, req *rolepermissionpb.CreateRolePermissionRequest) (*rolepermissionpb.CreateRolePermissionResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create role permission request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("role permission data is required")
	}
	if req.Data.RoleId == "" {
		return nil, fmt.Errorf("role ID is required")
	}
	if req.Data.PermissionId == "" {
		return nil, fmt.Errorf("permission ID is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	rolePermissionID := fmt.Sprintf("role-permission-%d-%d", now.UnixNano(), len(r.rolePermissions))

	// Create new role permission with proper timestamps and defaults
	newRolePermission := &rolepermissionpb.RolePermission{
		Id:                 rolePermissionID,
		RoleId:             req.Data.RoleId,
		PermissionId:       req.Data.PermissionId,
		Permission:         req.Data.Permission,
		PermissionType:     req.Data.PermissionType,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.rolePermissions[rolePermissionID] = newRolePermission

	return &rolepermissionpb.CreateRolePermissionResponse{
		Data:    []*rolepermissionpb.RolePermission{newRolePermission},
		Success: true,
	}, nil
}

// ReadRolePermission retrieves a role permission relationship by ID from stateful storage
func (r *MockRolePermissionRepository) ReadRolePermission(ctx context.Context, req *rolepermissionpb.ReadRolePermissionRequest) (*rolepermissionpb.ReadRolePermissionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read role permission request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role permission ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated role permissions)
	if rolePermission, exists := r.rolePermissions[req.Data.Id]; exists {
		return &rolepermissionpb.ReadRolePermissionResponse{
			Data:    []*rolepermissionpb.RolePermission{rolePermission},
			Success: true,
		}, nil
	}

	return nil, fmt.Errorf("role permission with ID '%s' not found", req.Data.Id)
}

// UpdateRolePermission updates an existing role permission relationship in stateful storage
func (r *MockRolePermissionRepository) UpdateRolePermission(ctx context.Context, req *rolepermissionpb.UpdateRolePermissionRequest) (*rolepermissionpb.UpdateRolePermissionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update role permission request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role permission ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify role permission exists
	existingRolePermission, exists := r.rolePermissions[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("role permission with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedRolePermission := &rolepermissionpb.RolePermission{
		Id:                 req.Data.Id,
		RoleId:             req.Data.RoleId,
		PermissionId:       req.Data.PermissionId,
		Permission:         req.Data.Permission,
		PermissionType:     req.Data.PermissionType,
		DateCreated:        existingRolePermission.DateCreated,       // Preserve original
		DateCreatedString:  existingRolePermission.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],             // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.rolePermissions[req.Data.Id] = updatedRolePermission

	return &rolepermissionpb.UpdateRolePermissionResponse{
		Data:    []*rolepermissionpb.RolePermission{updatedRolePermission},
		Success: true,
	}, nil
}

// DeleteRolePermission deletes a role permission relationship from stateful storage
func (r *MockRolePermissionRepository) DeleteRolePermission(ctx context.Context, req *rolepermissionpb.DeleteRolePermissionRequest) (*rolepermissionpb.DeleteRolePermissionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete role permission request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("role permission ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify role permission exists before deletion
	if _, exists := r.rolePermissions[req.Data.Id]; !exists {
		return nil, fmt.Errorf("role permission with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.rolePermissions, req.Data.Id)

	return &rolepermissionpb.DeleteRolePermissionResponse{
		Success: true,
	}, nil
}

// ListRolePermissions retrieves all role permission relationships from stateful storage
func (r *MockRolePermissionRepository) ListRolePermissions(ctx context.Context, req *rolepermissionpb.ListRolePermissionsRequest) (*rolepermissionpb.ListRolePermissionsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of role permissions
	rolePermissions := make([]*rolepermissionpb.RolePermission, 0, len(r.rolePermissions))
	for _, rolePermission := range r.rolePermissions {
		rolePermissions = append(rolePermissions, rolePermission)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		rolePermissions,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process role permission list data: %w", err)
	}

	// Convert processed items back to role permission protobuf format
	processedRolePermissions := make([]*rolepermissionpb.RolePermission, len(result.Items))
	for i, item := range result.Items {
		if rolePermission, ok := item.(*rolepermissionpb.RolePermission); ok {
			processedRolePermissions[i] = rolePermission
		} else {
			return nil, fmt.Errorf("failed to convert item to role permission type")
		}
	}

	return &rolepermissionpb.ListRolePermissionsResponse{
		Data:    processedRolePermissions,
		Success: true,
	}, nil
}

// GetRolePermissionListPageData retrieves role permissions with advanced filtering, sorting, searching, and pagination
func (r *MockRolePermissionRepository) GetRolePermissionListPageData(
	ctx context.Context,
	req *rolepermissionpb.GetRolePermissionListPageDataRequest,
) (*rolepermissionpb.GetRolePermissionListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get role permission list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of role permissions
	rolepermissions := make([]*rolepermissionpb.RolePermission, 0, len(r.rolePermissions))
	for _, rolepermission := range r.rolePermissions {
		rolepermissions = append(rolepermissions, rolepermission)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		rolepermissions,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process role permission list data: %w", err)
	}

	// Convert processed items back to role permission protobuf format
	processedRolePermissions := make([]*rolepermissionpb.RolePermission, len(result.Items))
	for i, item := range result.Items {
		if rolepermission, ok := item.(*rolepermissionpb.RolePermission); ok {
			processedRolePermissions[i] = rolepermission
		} else {
			return nil, fmt.Errorf("failed to convert item to role permission type")
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

	return &rolepermissionpb.GetRolePermissionListPageDataResponse{
		RolePermissionList: processedRolePermissions,
		Pagination:         result.PaginationResponse,
		SearchResults:      searchResults,
		Success:            true,
	}, nil
}

// GetRolePermissionItemPageData retrieves a single role permission with enhanced item page data
func (r *MockRolePermissionRepository) GetRolePermissionItemPageData(
	ctx context.Context,
	req *rolepermissionpb.GetRolePermissionItemPageDataRequest,
) (*rolepermissionpb.GetRolePermissionItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get role permission item page data request is required")
	}
	if req.RolePermissionId == "" {
		return nil, fmt.Errorf("role permission ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	rolepermission, exists := r.rolePermissions[req.RolePermissionId]
	if !exists {
		return nil, fmt.Errorf("role permission with ID '%s' not found", req.RolePermissionId)
	}

	// In a real implementation, you might:
	// 1. Load related data (role details, permission details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &rolepermissionpb.GetRolePermissionItemPageDataResponse{
		RolePermission: rolepermission,
		Success:        true,
	}, nil
}

// mapToProtobufRolePermission converts raw mock data to protobuf RolePermission
func (r *MockRolePermissionRepository) mapToProtobufRolePermission(rawRolePermission map[string]any) (*rolepermissionpb.RolePermission, error) {
	rolePermission := &rolepermissionpb.RolePermission{}

	// Map required fields
	if id, ok := rawRolePermission["id"].(string); ok {
		rolePermission.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	if roleId, ok := rawRolePermission["roleId"].(string); ok {
		rolePermission.RoleId = roleId
	} else {
		return nil, fmt.Errorf("missing or invalid roleId field")
	}

	if permissionId, ok := rawRolePermission["permissionId"].(string); ok {
		rolePermission.PermissionId = permissionId
	} else {
		return nil, fmt.Errorf("missing or invalid permissionId field")
	}

	// Map permission type - handle both string and numeric representations
	if permissionTypeRaw, ok := rawRolePermission["permissionType"]; ok {
		switch v := permissionTypeRaw.(type) {
		case string:
			if permissionTypeValue, exists := permissionpb.PermissionType_value[v]; exists {
				rolePermission.PermissionType = permissionpb.PermissionType(permissionTypeValue)
			} else {
				rolePermission.PermissionType = permissionpb.PermissionType_PERMISSION_TYPE_UNSPECIFIED
			}
		case float64:
			rolePermission.PermissionType = permissionpb.PermissionType(int32(v))
		case int32:
			rolePermission.PermissionType = permissionpb.PermissionType(v)
		default:
			rolePermission.PermissionType = permissionpb.PermissionType_PERMISSION_TYPE_UNSPECIFIED
		}
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawRolePermission["dateCreated"].(string); ok {
		rolePermission.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			rolePermission.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawRolePermission["dateModified"].(string); ok {
		rolePermission.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			rolePermission.DateModified = &timestamp
		}
	}

	if active, ok := rawRolePermission["active"].(bool); ok {
		rolePermission.Active = active
	}

	return rolePermission, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockRolePermissionRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewRolePermissionRepository creates a new role permission repository - Provider interface compatibility
func NewRolePermissionRepository(businessType string) rolepermissionpb.RolePermissionDomainServiceServer {
	return NewMockRolePermissionRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock", "role_permission", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockRolePermissionRepository(businessType), nil
	})
}
