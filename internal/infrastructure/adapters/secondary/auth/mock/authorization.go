//go:build mock_auth

package mock

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
)

// MockAuthorizationService provides a configurable mock for testing authorization scenarios
type MockAuthorizationService struct {
	enabled bool

	// Configuration for different test scenarios
	allowAll           bool
	denyAll            bool
	allowedPermissions map[string]bool
	userRoles          map[string][]string
	userWorkspaces     map[string][]string

	// Default test data
	defaultRoles      []string
	defaultWorkspaces []string
}

// NewMockAuthorizationService creates a new mock authorization service
func NewMockAuthorizationService(enabled bool) *MockAuthorizationService {
	return &MockAuthorizationService{
		enabled:            enabled,
		allowAll:           true, // Default to allowing all for simple tests
		denyAll:            false,
		allowedPermissions: make(map[string]bool),
		userRoles:          make(map[string][]string),
		userWorkspaces:     make(map[string][]string),
		defaultRoles:       []string{"user", "test-role"},
		defaultWorkspaces:  []string{"default-workspace", "test-workspace"},
	}
}

// Configuration methods for different test scenarios

// AllowAll configures the service to allow all permissions
func (m *MockAuthorizationService) AllowAll() *MockAuthorizationService {
	m.allowAll = true
	m.denyAll = false
	return m
}

// DenyAll configures the service to deny all permissions
func (m *MockAuthorizationService) DenyAll() *MockAuthorizationService {
	m.allowAll = false
	m.denyAll = true
	return m
}

// AllowPermissions configures specific permissions to be allowed
func (m *MockAuthorizationService) AllowPermissions(permissions ...string) *MockAuthorizationService {
	m.allowAll = false
	m.denyAll = false
	for _, perm := range permissions {
		m.allowedPermissions[perm] = true
	}
	return m
}

// SetUserRoles configures roles for a specific user
func (m *MockAuthorizationService) SetUserRoles(userID string, roles ...string) *MockAuthorizationService {
	m.userRoles[userID] = roles
	return m
}

// SetUserWorkspaces configures workspaces for a specific user
func (m *MockAuthorizationService) SetUserWorkspaces(userID string, workspaces ...string) *MockAuthorizationService {
	m.userWorkspaces[userID] = workspaces
	return m
}

// SetDefaults configures default roles and workspaces for unknown users
func (m *MockAuthorizationService) SetDefaults(roles []string, workspaces []string) *MockAuthorizationService {
	m.defaultRoles = roles
	m.defaultWorkspaces = workspaces
	return m
}

// AuthorizationService interface implementation

func (m *MockAuthorizationService) HasPermission(ctx context.Context, userID, permission string) (bool, error) {
	fmt.Printf("🔐 HasPermission called: userID=%s, permission=%s, enabled=%v, allowAll=%v, denyAll=%v\n", userID, permission, m.enabled, m.allowAll, m.denyAll)

	if !m.enabled {
		return false, fmt.Errorf("authorization service is disabled")
	}

	// Handle different test scenarios
	if m.denyAll {
		fmt.Printf("🔐 DenyAll: returning false\n")
		return false, nil
	}

	if m.allowAll {
		fmt.Printf("🔐 AllowAll: returning true\n")
		return true, nil
	}

	// Check specific allowed permissions
	result := m.allowedPermissions[permission]
	fmt.Printf("🔐 Specific permission check: permission=%s, allowed=%v\n", permission, result)
	return result, nil
}

func (m *MockAuthorizationService) HasGlobalPermission(ctx context.Context, userID, permission string) (bool, error) {
	// For mock purposes, global permissions work the same as regular permissions
	return m.HasPermission(ctx, userID, permission)
}

func (m *MockAuthorizationService) HasPermissionInWorkspace(ctx context.Context, userID, workspaceID, permission string) (bool, error) {
	if !m.enabled {
		return false, fmt.Errorf("authorization service is disabled")
	}

	// Check if user has access to this workspace first
	userWorkspaces, _ := m.GetUserWorkspaces(ctx, userID)
	hasWorkspaceAccess := false
	for _, ws := range userWorkspaces {
		if ws == workspaceID {
			hasWorkspaceAccess = true
			break
		}
	}

	if !hasWorkspaceAccess {
		return false, nil
	}

	// Then check permission (using same logic as HasPermission)
	workspacePermission := fmt.Sprintf("%s:%s", workspaceID, permission)

	if m.denyAll {
		return false, nil
	}

	if m.allowAll {
		return true, nil
	}

	// Check both the workspace-specific permission and the general permission
	return m.allowedPermissions[workspacePermission] || m.allowedPermissions[permission], nil
}

func (m *MockAuthorizationService) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	if !m.enabled {
		return nil, fmt.Errorf("authorization service is disabled")
	}

	// Return user-specific roles if configured
	if roles, exists := m.userRoles[userID]; exists {
		return roles, nil
	}

	// Return default roles
	return m.defaultRoles, nil
}

func (m *MockAuthorizationService) GetUserRolesInWorkspace(ctx context.Context, userID, workspaceID string) ([]string, error) {
	if !m.enabled {
		return nil, fmt.Errorf("authorization service is disabled")
	}

	// For mock purposes, return the same roles with workspace prefix
	roles, err := m.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Add workspace context to role names
	workspaceRoles := make([]string, len(roles))
	for i, role := range roles {
		workspaceRoles[i] = fmt.Sprintf("%s-%s", workspaceID, role)
	}

	return workspaceRoles, nil
}

func (m *MockAuthorizationService) GetUserWorkspaces(ctx context.Context, userID string) ([]string, error) {
	if !m.enabled {
		return nil, fmt.Errorf("authorization service is disabled")
	}

	// Return user-specific workspaces if configured
	if workspaces, exists := m.userWorkspaces[userID]; exists {
		return workspaces, nil
	}

	// Return default workspaces
	return m.defaultWorkspaces, nil
}

func (m *MockAuthorizationService) GetUserPermissionCodes(ctx context.Context, userID string) ([]string, error) {
	if !m.enabled {
		return nil, fmt.Errorf("authorization service is disabled")
	}

	// If allowAll, return all configured allowed permissions
	if m.allowAll {
		codes := make([]string, 0, len(m.allowedPermissions))
		for perm := range m.allowedPermissions {
			codes = append(codes, perm)
		}
		return codes, nil
	}

	if m.denyAll {
		return []string{}, nil
	}

	// Return specifically allowed permissions
	codes := make([]string, 0, len(m.allowedPermissions))
	for perm := range m.allowedPermissions {
		codes = append(codes, perm)
	}
	return codes, nil
}

func (m *MockAuthorizationService) IsEnabled() bool {
	return m.enabled
}

// Utility methods for testing specific scenarios

// NewAllowAllAuth creates a mock that allows all permissions
func NewAllowAllAuth() *MockAuthorizationService {
	return NewMockAuthorizationService(true).AllowAll()
}

// NewDenyAllAuth creates a mock that denies all permissions
func NewDenyAllAuth() *MockAuthorizationService {
	return NewMockAuthorizationService(true).DenyAll()
}

// NewRecordTestAuth creates a mock configured for record testing
func NewRecordTestAuth() *MockAuthorizationService {
	return NewMockAuthorizationService(true).
		AllowPermissions(
			ports.EntityPermission(ports.EntityRecord, ports.ActionCreate),
			ports.EntityPermission(ports.EntityRecord, ports.ActionRead),
			ports.EntityPermission(ports.EntityRecord, ports.ActionUpdate),
			ports.EntityPermission(ports.EntityRecord, ports.ActionDelete),
			ports.EntityPermission(ports.EntityRecord, ports.ActionList),
		).
		SetDefaults(
			[]string{"record-user", "test-user"},
			[]string{"education-workspace", "test-workspace"},
		)
}

// NewDisabledAuth creates a disabled auth service
func NewDisabledAuth() *MockAuthorizationService {
	return NewMockAuthorizationService(false)
}
