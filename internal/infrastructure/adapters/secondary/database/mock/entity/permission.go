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
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
)

// MockPermissionRepository implements entity.PermissionRepository using stateful mock data
type MockPermissionRepository struct {
	permissionpb.UnimplementedPermissionDomainServiceServer
	businessType string
	permissions  map[string]*permissionpb.Permission // Persistent in-memory store
	mutex        sync.RWMutex                        // Thread-safe concurrent access
	initialized  bool                                // Prevent double initialization
	processor    *listdata.ListDataProcessor         // List data processing
}

// PermissionRepositoryOption allows configuration of repository behavior
type PermissionRepositoryOption func(*MockPermissionRepository)

// WithPermissionTestOptimizations enables test-specific optimizations
func WithPermissionTestOptimizations(enabled bool) PermissionRepositoryOption {
	return func(r *MockPermissionRepository) {
		// Test optimizations could include faster ID generation, logging, etc.
		// For now, this is a placeholder for future optimizations
	}
}

// NewMockPermissionRepository creates a new mock permission repository
func NewMockPermissionRepository(businessType string, options ...PermissionRepositoryOption) permissionpb.PermissionDomainServiceServer {
	if businessType == "" {
		businessType = "education" // Default fallback
	}

	repo := &MockPermissionRepository{
		businessType: businessType,
		permissions:  make(map[string]*permissionpb.Permission),
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
func (r *MockPermissionRepository) loadInitialData() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.initialized {
		return nil // Prevent double initialization
	}

	// Load from @leapfor/copya package
	rawPermissions, err := datamock.LoadPermissions(r.businessType)
	if err != nil {
		return fmt.Errorf("failed to load initial permissions: %w", err)
	}

	// Convert and store each permission
	for _, rawPermission := range rawPermissions {
		if permission, err := r.mapToProtobufPermission(rawPermission); err == nil {
			r.permissions[permission.Id] = permission
		}
	}

	r.initialized = true
	return nil
}

// CreatePermission creates a new permission with stateful storage
func (r *MockPermissionRepository) CreatePermission(ctx context.Context, req *permissionpb.CreatePermissionRequest) (*permissionpb.CreatePermissionResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("create permission request is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("permission data is required")
	}
	if req.Data.WorkspaceId == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}
	if req.Data.UserId == "" {
		return nil, fmt.Errorf("user ID is required")
	}
	if req.Data.PermissionCode == "" {
		return nil, fmt.Errorf("permission code is required")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Generate unique ID with timestamp
	now := time.Now()
	permissionID := fmt.Sprintf("permission-%d-%d", now.UnixNano(), len(r.permissions))

	// Create new permission with proper timestamps and defaults
	newPermission := &permissionpb.Permission{
		Id:                 permissionID,
		WorkspaceId:        req.Data.WorkspaceId,
		UserId:             req.Data.UserId,
		GrantedByUserId:    req.Data.GrantedByUserId,
		PermissionCode:     req.Data.PermissionCode,
		PermissionType:     req.Data.PermissionType,
		DateCreated:        &[]int64{now.UnixMilli()}[0],
		DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
		DateModified:       &[]int64{now.UnixMilli()}[0],
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             true, // Default to active
	}

	// Store in persistent map
	r.permissions[permissionID] = newPermission

	return &permissionpb.CreatePermissionResponse{
		Data:    []*permissionpb.Permission{newPermission},
		Success: true,
	}, nil
}

// ReadPermission retrieves a permission by ID from stateful storage
func (r *MockPermissionRepository) ReadPermission(ctx context.Context, req *permissionpb.ReadPermissionRequest) (*permissionpb.ReadPermissionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("read permission request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("permission ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store first (includes created/updated permissions)
	if permission, exists := r.permissions[req.Data.Id]; exists {
		return &permissionpb.ReadPermissionResponse{
			Data:    []*permissionpb.Permission{permission},
			Success: true,
		}, nil
	}

	return &permissionpb.ReadPermissionResponse{
		Data:    []*permissionpb.Permission{},
		Success: false,
	}, nil
}

// UpdatePermission updates an existing permission in stateful storage
func (r *MockPermissionRepository) UpdatePermission(ctx context.Context, req *permissionpb.UpdatePermissionRequest) (*permissionpb.UpdatePermissionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update permission request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("permission ID is required for update")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify permission exists
	existingPermission, exists := r.permissions[req.Data.Id]
	if !exists {
		return nil, fmt.Errorf("permission with ID '%s' not found", req.Data.Id)
	}

	// Update only specified fields (preserve others)
	now := time.Now()
	updatedPermission := &permissionpb.Permission{
		Id:                 req.Data.Id,
		WorkspaceId:        req.Data.WorkspaceId,
		UserId:             req.Data.UserId,
		GrantedByUserId:    req.Data.GrantedByUserId,
		PermissionCode:     req.Data.PermissionCode,
		PermissionType:     req.Data.PermissionType,
		DateCreated:        existingPermission.DateCreated,       // Preserve original
		DateCreatedString:  existingPermission.DateCreatedString, // Preserve original
		DateModified:       &[]int64{now.UnixMilli()}[0],         // Update modification time
		DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		Active:             req.Data.Active,
	}

	// Update in persistent store
	r.permissions[req.Data.Id] = updatedPermission

	return &permissionpb.UpdatePermissionResponse{
		Data:    []*permissionpb.Permission{updatedPermission},
		Success: true,
	}, nil
}

// DeletePermission deletes a permission from stateful storage
func (r *MockPermissionRepository) DeletePermission(ctx context.Context, req *permissionpb.DeletePermissionRequest) (*permissionpb.DeletePermissionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete permission request is required")
	}
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("permission ID is required for deletion")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Verify permission exists before deletion
	if _, exists := r.permissions[req.Data.Id]; !exists {
		return nil, fmt.Errorf("permission with ID '%s' not found", req.Data.Id)
	}

	// Perform actual deletion from persistent store
	delete(r.permissions, req.Data.Id)

	return &permissionpb.DeletePermissionResponse{
		Success: true,
	}, nil
}

// ListPermissions retrieves all permissions from stateful storage
func (r *MockPermissionRepository) ListPermissions(ctx context.Context, req *permissionpb.ListPermissionsRequest) (*permissionpb.ListPermissionsResponse, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of permissions
	permissions := make([]*permissionpb.Permission, 0, len(r.permissions))
	for _, permission := range r.permissions {
		permissions = append(permissions, permission)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		permissions,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process permission list data: %w", err)
	}

	// Convert processed items back to permission protobuf format
	processedPermissions := make([]*permissionpb.Permission, len(result.Items))
	for i, item := range result.Items {
		if permission, ok := item.(*permissionpb.Permission); ok {
			processedPermissions[i] = permission
		} else {
			return nil, fmt.Errorf("failed to convert item to permission type")
		}
	}

	return &permissionpb.ListPermissionsResponse{
		Data:    processedPermissions,
		Success: true,
	}, nil
}

// GetPermissionListPageData retrieves permissions with advanced filtering, sorting, searching, and pagination
func (r *MockPermissionRepository) GetPermissionListPageData(
	ctx context.Context,
	req *permissionpb.GetPermissionListPageDataRequest,
) (*permissionpb.GetPermissionListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get permission list page data request is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Convert map to slice of permissions
	permissions := make([]*permissionpb.Permission, 0, len(r.permissions))
	for _, permission := range r.permissions {
		permissions = append(permissions, permission)
	}

	// Use the list data processor to handle filtering, sorting, searching, and pagination
	result, err := r.processor.ProcessListRequest(
		permissions,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process permission list data: %w", err)
	}

	// Convert processed items back to permission protobuf format
	processedPermissions := make([]*permissionpb.Permission, len(result.Items))
	for i, item := range result.Items {
		if permission, ok := item.(*permissionpb.Permission); ok {
			processedPermissions[i] = permission
		} else {
			return nil, fmt.Errorf("failed to convert item to permission type")
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

	return &permissionpb.GetPermissionListPageDataResponse{
		PermissionList: processedPermissions,
		Pagination:     result.PaginationResponse,
		SearchResults:  searchResults,
		Success:        true,
	}, nil
}

// GetPermissionItemPageData retrieves a single permission with enhanced item page data
func (r *MockPermissionRepository) GetPermissionItemPageData(
	ctx context.Context,
	req *permissionpb.GetPermissionItemPageDataRequest,
) (*permissionpb.GetPermissionItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get permission item page data request is required")
	}
	if req.PermissionId == "" {
		return nil, fmt.Errorf("permission ID is required")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check in-memory store
	permission, exists := r.permissions[req.PermissionId]
	if !exists {
		return nil, fmt.Errorf("permission with ID '%s' not found", req.PermissionId)
	}

	// In a real implementation, you might:
	// 1. Load related data (workspace details, user details, granting user details)
	// 2. Apply access control checks
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging

	return &permissionpb.GetPermissionItemPageDataResponse{
		Permission: permission,
		Success:    true,
	}, nil
}

// mapToProtobufPermission converts raw mock data to protobuf Permission
func (r *MockPermissionRepository) mapToProtobufPermission(rawPermission map[string]any) (*permissionpb.Permission, error) {
	permission := &permissionpb.Permission{}

	// Map required fields
	if id, ok := rawPermission["id"].(string); ok {
		permission.Id = id
	} else {
		return nil, fmt.Errorf("missing or invalid id field")
	}

	// Mock data contains permission definitions, not user assignments
	// Map to protobuf structure with default values for testing
	permission.WorkspaceId = "workspace-default"
	permission.UserId = "user-system"
	permission.GrantedByUserId = "admin-system"

	// Map permissionCode from id field (e.g., "client.list" -> "client.list")
	if id, ok := rawPermission["id"].(string); ok {
		permission.PermissionCode = id
	} else {
		return nil, fmt.Errorf("missing or invalid permissionCode field")
	}

	// Map permission type - handle both string and numeric representations
	if permissionTypeRaw, ok := rawPermission["permissionType"]; ok {
		switch v := permissionTypeRaw.(type) {
		case string:
			if permissionTypeValue, exists := permissionpb.PermissionType_value[v]; exists {
				permission.PermissionType = permissionpb.PermissionType(permissionTypeValue)
			} else {
				permission.PermissionType = permissionpb.PermissionType_PERMISSION_TYPE_UNSPECIFIED
			}
		case float64:
			permission.PermissionType = permissionpb.PermissionType(int32(v))
		case int32:
			permission.PermissionType = permissionpb.PermissionType(v)
		default:
			permission.PermissionType = permissionpb.PermissionType_PERMISSION_TYPE_UNSPECIFIED
		}
	}

	// Handle date fields - convert string timestamps to Unix timestamps
	if dateCreated, ok := rawPermission["dateCreated"].(string); ok {
		permission.DateCreatedString = &dateCreated
		if timestamp, err := r.parseTimestamp(dateCreated); err == nil {
			permission.DateCreated = &timestamp
		}
	}

	if dateModified, ok := rawPermission["dateModified"].(string); ok {
		permission.DateModifiedString = &dateModified
		if timestamp, err := r.parseTimestamp(dateModified); err == nil {
			permission.DateModified = &timestamp
		}
	}

	if active, ok := rawPermission["active"].(bool); ok {
		permission.Active = active
	}

	return permission, nil
}

// parseTimestamp converts string timestamp to Unix timestamp
func (r *MockPermissionRepository) parseTimestamp(timestampStr string) (int64, error) {
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

// NewPermissionRepository creates a new permission repository - Provider interface compatibility
func NewPermissionRepository(businessType string) permissionpb.PermissionDomainServiceServer {
	return NewMockPermissionRepository(businessType)
}

func init() {
	registry.RegisterRepositoryFactory("mock", "permission", func(conn any, tableName string) (any, error) {
		businessType, _ := conn.(string)
		if businessType == "" {
			businessType = "education"
		}
		return NewMockPermissionRepository(businessType), nil
	})
}
