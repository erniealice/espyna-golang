package security

import "sync"

// PermissionCacheInvalidator is the application-layer port for evicting cached
// RBAC permission-code lookups. It exists so authorization-mutating use cases
// (role_permission grant/revoke, permission create/update, workspace_user_role
// assign/change/unassign) can force the permission loader's per-identity cache
// to reload IMMEDIATELY after a grant change — closing the window (P10/D2)
// where a new grant was invisible until the loader's TTL expired or the process
// restarted.
//
// Deps point INWARD: this interface lives in the application layer; the concrete
// cache (consumer/http.DBPermissionLoader) implements it structurally and is
// wired in at the composition layer. Use cases depend only on this port, never
// on consumer/http. Follows the AuditService cross-cutting-port precedent
// (ports/infrastructure/audit.go): a narrow interface with a NoOp default,
// implemented by the concrete adapter and injected by composition.
//
// Two eviction grains match the two cache-key dimensions the loader keys on
// (see consumer/http/loaders.go permissionCacheKey):
//   - InvalidateUser evicts every entry for a login user id (across all of that
//     user's workspaces and bindings) — used when a change names a User.
//   - InvalidateBinding evicts every entry for a single binding row id
//     (= WorkspaceUser.id for a staff principal, the value the session-scoped
//     permission loader stores as the cache-key bindingID) — used when a change
//     names a specific binding, which is more precise (it leaves the same
//     person's OTHER bindings untouched) and lets a role→permission grant
//     enumerate the affected bindings from workspace_user_role rows in a single
//     hop (no WorkspaceUser resolution).
//
// Implementations MUST NOT over-clear: an eviction affects only entries matching
// the given id and never flushes the whole cache (P10/D2 audit-gate requirement).
type PermissionCacheInvalidator interface {
	// InvalidateUser drops every cached permission entry for userID. Idempotent;
	// a no-op when userID is empty or unknown.
	InvalidateUser(userID string)

	// InvalidateBinding drops every cached permission entry whose binding id
	// matches bindingID (the workspace_user id used as the session-scoped cache
	// key's bindingID). Idempotent; a no-op when bindingID is empty or unknown.
	InvalidateBinding(bindingID string)
}

// NoOpPermissionCacheInvalidator is the safe default when no permission cache is
// wired (unit tests, mock boots, or any build whose permission loader is
// disabled). Every method does nothing.
type NoOpPermissionCacheInvalidator struct{}

// NewNoOpPermissionCacheInvalidator returns the do-nothing invalidator.
func NewNoOpPermissionCacheInvalidator() *NoOpPermissionCacheInvalidator {
	return &NoOpPermissionCacheInvalidator{}
}

func (NoOpPermissionCacheInvalidator) InvalidateUser(string)    {}
func (NoOpPermissionCacheInvalidator) InvalidateBinding(string) {}

// SettablePermissionCacheInvalidator is a late-bound indirection that resolves
// the construction-order inversion between the use cases and the loader: the
// permission loader wraps the security use case (it cannot be built until the
// use cases exist), yet the use cases need a reference to the loader to
// invalidate it. The Container builds ONE holder up front, injects it into the
// authorization-mutating use cases, and — once the loader is constructed at the
// composition layer — sets the holder's delegate to the loader.
//
// Until SetDelegate is called the holder is itself the NoOp default: every
// InvalidateUser/InvalidateBinding is a silent no-op, so a build with no
// permission loader (mock/dev) is unaffected. Thread-safe: SetDelegate may race
// with in-flight invalidations during boot.
type SettablePermissionCacheInvalidator struct {
	mu       sync.RWMutex
	delegate PermissionCacheInvalidator
}

// NewSettablePermissionCacheInvalidator returns an unset (no-op) holder.
func NewSettablePermissionCacheInvalidator() *SettablePermissionCacheInvalidator {
	return &SettablePermissionCacheInvalidator{}
}

// SetDelegate points the holder at the real invalidator (the permission loader).
// Passing nil reverts it to the no-op state.
func (s *SettablePermissionCacheInvalidator) SetDelegate(d PermissionCacheInvalidator) {
	s.mu.Lock()
	s.delegate = d
	s.mu.Unlock()
}

// InvalidateUser forwards to the delegate, or no-ops when unset.
func (s *SettablePermissionCacheInvalidator) InvalidateUser(userID string) {
	s.mu.RLock()
	d := s.delegate
	s.mu.RUnlock()
	if d != nil {
		d.InvalidateUser(userID)
	}
}

// InvalidateBinding forwards to the delegate, or no-ops when unset.
func (s *SettablePermissionCacheInvalidator) InvalidateBinding(bindingID string) {
	s.mu.RLock()
	d := s.delegate
	s.mu.RUnlock()
	if d != nil {
		d.InvalidateBinding(bindingID)
	}
}
