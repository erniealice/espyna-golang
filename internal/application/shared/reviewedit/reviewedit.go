// Package reviewedit carries the per-request decision that the acting staff
// session may UPDATE an existing outcome recorded by someone else, inside its
// review scope (plan 20261105-reviewer-edit-grades), and — plan
// 20261106-review-editor-override — FILL a blank cell (recorded as the cell
// owner) or CLEAR (soft-delete) an outcome recorded by someone else. The
// decision is resolved once by the use case (fresh strict permission verdict
// per capability) and read by the postgres adapter; the zero value is
// "denied" (fail closed).
package reviewedit

import (
	"context"

	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/shared/identity"
)

// PermissionCode is the capability that grants review-scope updates.
const PermissionCode = "task_outcome:update_in_review_scope"

// CreateCode is the capability that grants review-scope fills of a blank cell
// (Decision.Fill). Separate from PermissionCode so it is revocable on its own
// (plan 20261106 O4 = A).
const CreateCode = "task_outcome:create_in_review_scope"

// DeleteCode is the capability that grants review-scope clears of an outcome
// recorded by someone else (Decision.Clear, a snapshot-checked soft delete).
const DeleteCode = "task_outcome:delete_in_review_scope"

// ClassOverrideCode is the capability that lets a staff member holding a PRIMARY class
// edge change any recorded grade in that class on an open sheet, whoever
// recorded it (Decision.ClassOverride; design-v2 rule 4, main-teacher override).
const ClassOverrideCode = "task_outcome:update_in_class_scope"

// principalTypeStaff mirrors esqyma domain.entity.v1.PrincipalType STAFF (7), the
// value the session binding stamps on identity.PrincipalType. No shared
// constant is importable from the application layer (same local mirror as
// job_phase/transitions.go).
const principalTypeStaff int32 = 7

// Decision is the resolved review-edit capability set. Allowed = update
// (PermissionCode), Fill = CreateCode, Clear = DeleteCode, each from its own
// fresh strict verdict. WorkspaceScope is only meaningful when Any() is true.
type Decision struct {
	Allowed        bool
	Fill           bool
	Clear          bool
	WorkspaceScope bool
	// ClassOverride = ClassOverrideCode (fresh strict verdict). It is NOT a review
	// capability: Any() ignores it and WorkspaceScope is never derived from it.
	ClassOverride bool
}

// Any reports whether at least one review capability is on.
func (d Decision) Any() bool { return d.Allowed || d.Fill || d.Clear }

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

// Resolve returns a non-zero Decision only for a staff session (kind 7) with a
// non-empty principal id and a user id. Each capability (Allowed / Fill /
// Clear) is its own fresh strict verdict of PermissionCode / CreateCode /
// DeleteCode; a checker error denies that capability only. WorkspaceScope
// mirrors identity.HasWorkspaceRowScope and is evaluated only when Any().
// A nil checker or any other session kind yields the zero Decision.
func Resolve(ctx context.Context, c FreshChecker) Decision {
	if c == nil {
		return Decision{}
	}
	id, ok := identity.FromContext(ctx)
	if !ok || id == nil || id.PrincipalID == "" || id.PrincipalType != principalTypeStaff {
		return Decision{}
	}
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil || userID == "" {
		return Decision{}
	}
	has := func(code string) bool {
		ok, err := c.HasPermissionStrictFresh(ctx, userID, code)
		return err == nil && ok
	}
	d := Decision{
		Allowed: has(PermissionCode),
		Fill:    has(CreateCode),
		Clear:   has(DeleteCode),
	}
	d.ClassOverride = has(ClassOverrideCode)
	if !d.Any() {
		if d.ClassOverride {
			return Decision{ClassOverride: true}
		}
		return Decision{}
	}
	d.WorkspaceScope = identity.HasWorkspaceRowScope(ctx)
	return d
}
