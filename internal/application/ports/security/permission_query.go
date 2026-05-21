package security

import "context"

// PermissionQuery is the narrow port for RBAC permission-code lookups.
//
// Separate from Authorizer: Authorizer answers
// "does user X have permission Y?" (boolean), while PermissionQuery returns
// the full set of ALLOW codes net of DENYs so callers can cache or filter
// in bulk (e.g. for sidebar building, batch authorization, permission
// hydration on session cookie).
//
// DENY-wins semantics: if the user holds both an ALLOW and a DENY grant
// for a given permission, it is treated as denied and omitted from the
// returned set.
type PermissionQuery interface {
	// GetUserPermissionCodes returns all effective ALLOW permission codes
	// for the given user within the given workspace.
	// Returns an empty slice (not nil) when the user has no permissions.
	GetUserPermissionCodes(ctx context.Context, userID, workspaceID string) ([]string, error)
}
