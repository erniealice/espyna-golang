//go:build postgres

package database

import (
	"context"
	"database/sql"
)

// DatabaseAuthorizationService implements AuthorizationService using PostgreSQL
// with RBAC tables (workspace_user, workspace_user_role, role_permission, permission).
// It uses a DENY-wins strategy: if any role grants DENY for a permission,
// the user is denied regardless of any ALLOW grants.
type DatabaseAuthorizationService struct {
	db      *sql.DB
	enabled bool
}

// NewDatabaseAuthorizationService creates a new database-backed authorization service.
func NewDatabaseAuthorizationService(db *sql.DB) *DatabaseAuthorizationService {
	return &DatabaseAuthorizationService{
		db:      db,
		enabled: true,
	}
}

// HasPermission checks if a user has a specific permission across ALL workspaces.
// Uses DENY-wins: allowed = hasAllow && !hasDeny.
func (s *DatabaseAuthorizationService) HasPermission(ctx context.Context, userID, permission string) (bool, error) {
	query := `
		SELECT
			COALESCE(bool_or(rp.permission_type = 'PERMISSION_TYPE_ALLOW'), false) AS has_allow,
			COALESCE(bool_or(rp.permission_type = 'PERMISSION_TYPE_DENY'), false) AS has_deny
		FROM workspace_user wu
		JOIN workspace_user_role wur ON wur.workspace_user_id = wu.id AND wur.active = true
		JOIN role_permission rp ON rp.role_id = wur.role_id AND rp.active = true
		JOIN permission p ON p.id = rp.permission_id AND p.active = true
		WHERE wu.user_id = $1 AND wu.active = true AND p.permission_code = $2`

	var hasAllow, hasDeny bool
	err := s.db.QueryRowContext(ctx, query, userID, permission).Scan(&hasAllow, &hasDeny)
	if err != nil {
		return false, err
	}

	return hasAllow && !hasDeny, nil
}

// HasGlobalPermission checks if a user has a global/system-wide permission.
// Equivalent to HasPermission (checks across all workspaces).
func (s *DatabaseAuthorizationService) HasGlobalPermission(ctx context.Context, userID, permission string) (bool, error) {
	return s.HasPermission(ctx, userID, permission)
}

// HasPermissionInWorkspace checks if a user has a permission within a specific workspace.
// Uses the same DENY-wins strategy scoped to a single workspace.
func (s *DatabaseAuthorizationService) HasPermissionInWorkspace(ctx context.Context, userID, workspaceID, permission string) (bool, error) {
	query := `
		SELECT
			COALESCE(bool_or(rp.permission_type = 'PERMISSION_TYPE_ALLOW'), false) AS has_allow,
			COALESCE(bool_or(rp.permission_type = 'PERMISSION_TYPE_DENY'), false) AS has_deny
		FROM workspace_user wu
		JOIN workspace_user_role wur ON wur.workspace_user_id = wu.id AND wur.active = true
		JOIN role_permission rp ON rp.role_id = wur.role_id AND rp.active = true
		JOIN permission p ON p.id = rp.permission_id AND p.active = true
		WHERE wu.user_id = $1 AND wu.active = true AND p.permission_code = $2 AND wu.workspace_id = $3`

	var hasAllow, hasDeny bool
	err := s.db.QueryRowContext(ctx, query, userID, permission, workspaceID).Scan(&hasAllow, &hasDeny)
	if err != nil {
		return false, err
	}

	return hasAllow && !hasDeny, nil
}

// GetUserRoles returns all distinct role names assigned to a user across all workspaces.
func (s *DatabaseAuthorizationService) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	query := `
		SELECT DISTINCT r.name
		FROM workspace_user wu
		JOIN workspace_user_role wur ON wur.workspace_user_id = wu.id AND wur.active = true
		JOIN role r ON r.id = wur.role_id AND r.active = true
		WHERE wu.user_id = $1 AND wu.active = true`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	return roles, rows.Err()
}

// GetUserRolesInWorkspace returns role names assigned to a user in a specific workspace.
func (s *DatabaseAuthorizationService) GetUserRolesInWorkspace(ctx context.Context, userID, workspaceID string) ([]string, error) {
	query := `
		SELECT DISTINCT r.name
		FROM workspace_user wu
		JOIN workspace_user_role wur ON wur.workspace_user_id = wu.id AND wur.active = true
		JOIN role r ON r.id = wur.role_id AND r.active = true
		WHERE wu.user_id = $1 AND wu.active = true AND wu.workspace_id = $2`

	rows, err := s.db.QueryContext(ctx, query, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	return roles, rows.Err()
}

// GetUserWorkspaces returns all workspace IDs that a user has access to.
func (s *DatabaseAuthorizationService) GetUserWorkspaces(ctx context.Context, userID string) ([]string, error) {
	query := `
		SELECT DISTINCT wu.workspace_id
		FROM workspace_user wu
		WHERE wu.user_id = $1 AND wu.active = true`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workspaces []string
	for rows.Next() {
		var wsID string
		if err := rows.Scan(&wsID); err != nil {
			return nil, err
		}
		workspaces = append(workspaces, wsID)
	}

	return workspaces, rows.Err()
}

// GetUserPermissionCodes returns all effective permission codes for a user.
// Returns ALLOW'd codes minus any DENY'd codes (DENY-wins).
func (s *DatabaseAuthorizationService) GetUserPermissionCodes(ctx context.Context, userID string) ([]string, error) {
	query := `
		SELECT DISTINCT p.permission_code
		FROM workspace_user wu
		JOIN workspace_user_role wur ON wur.workspace_user_id = wu.id AND wur.active = true
		JOIN role_permission rp ON rp.role_id = wur.role_id AND rp.active = true
			AND rp.permission_type = 'PERMISSION_TYPE_ALLOW'
		JOIN permission p ON p.id = rp.permission_id AND p.active = true
		WHERE wu.user_id = $1 AND wu.active = true
			AND p.permission_code NOT IN (
				SELECT p2.permission_code
				FROM workspace_user wu2
				JOIN workspace_user_role wur2 ON wur2.workspace_user_id = wu2.id AND wur2.active = true
				JOIN role_permission rp2 ON rp2.role_id = wur2.role_id AND rp2.active = true
					AND rp2.permission_type = 'PERMISSION_TYPE_DENY'
				JOIN permission p2 ON p2.id = rp2.permission_id AND p2.active = true
				WHERE wu2.user_id = $1 AND wu2.active = true
			)`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}

	return codes, rows.Err()
}

// IsEnabled returns whether authorization is enabled (always true for database impl).
func (s *DatabaseAuthorizationService) IsEnabled() bool {
	return s.enabled
}
