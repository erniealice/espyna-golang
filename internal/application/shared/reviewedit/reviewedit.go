// Package reviewedit carries the per-request decision that the acting staff
// session may UPDATE an existing outcome recorded by someone else, inside its
// review scope (plan 20261105-reviewer-edit-grades). The decision is resolved
// once by the use case (fresh strict permission verdict) and read by the
// postgres adapter; the zero value is "denied" (fail closed).
package reviewedit

import (
	"context"

	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/shared/identity"
)

// PermissionCode is the capability that grants review-scope updates.
const PermissionCode = "task_outcome:update_in_review_scope"

// principalTypeStaff mirrors esqyma domain.entity.v1.PrincipalType STAFF (7), the
// value the session binding stamps on identity.PrincipalType. No shared
// constant is importable from the application layer (same local mirror as
// job_phase/transitions.go).
const principalTypeStaff int32 = 7

// Decision is the resolved review-edit capability. WorkspaceScope is only
// meaningful when Allowed.
type Decision struct {
	Allowed        bool
	WorkspaceScope bool
}

type contextKey struct{}

// WithDecision stores d on ctx.
func WithDecision(ctx context.Context, d Decision) context.Context {
	return context.WithValue(ctx, contextKey{}, d)
}

// DecisionFromContext returns the stored Decision; a missing decision yields the
// zero (denied) value and ok=false.
func DecisionFromContext(ctx context.Context) (Decision, bool) {
	d, ok := ctx.Value(contextKey{}).(Decision)
	return d, ok
}

// FreshChecker is the authoritative deny-capable permission verdict (bypasses
// the TTL cache, reads on the ambient transaction executor).
type FreshChecker interface {
	HasPermissionStrictFresh(ctx context.Context, userID, code string) (bool, error)
}

// Resolve returns Allowed only for a staff session (kind 7) with a non-empty
// principal id and a fresh strict PermissionCode verdict. WorkspaceScope mirrors
// identity.HasWorkspaceRowScope and is evaluated only when Allowed. Any error,
// nil checker or other kind yields the zero Decision.
func Resolve(ctx context.Context, c FreshChecker) Decision {
	if c == nil {
		return Decision{}
	}
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil || id.PrincipalID == "" || id.PrincipalType != principalTypeStaff {
		if ok && id != nil {
		} else {
		}
		return Decision{}
	}
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil || userID == "" {
		return Decision{}
	}
	has, err := c.HasPermissionStrictFresh(ctx, userID, PermissionCode)
	if err != nil || !has {
		return Decision{}
	}
	return Decision{Allowed: true, WorkspaceScope: identity.HasWorkspaceRowScope(ctx)}
}
