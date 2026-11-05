package task_outcome

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/reviewedit"
	"github.com/erniealice/espyna-golang/shared/identity"
)

// strictAuthz embeds a nil ports.Authorizer (never called) and adds the fresh
// strict capability.
type strictAuthz struct {
	ports.Authorizer
	perms map[string]bool
	err   error
}

func (s strictAuthz) HasPermissionStrictFresh(_ context.Context, _ string, code string) (bool, error) {
	return s.perms[code], s.err
}

func staffCtx(kind int32, principalID string, wsScope bool) context.Context {
	ctx := contextutil.WithUserID(context.Background(), "user-1")
	ctx = identity.WithSessionBinding(ctx, kind, principalID, "", "")
	if wsScope {
		ctx = identity.WithWorkspaceRowScope(ctx)
	}
	return ctx
}

var withPerm = map[string]bool{reviewedit.PermissionCode: true}

// AC-0
func TestResolveReviewEdit_StaffWithPermAllowed(t *testing.T) {
	got := resolveReviewEdit(staffCtx(7, "staff-1", false), strictAuthz{perms: withPerm})
	if got != (reviewedit.Decision{Allowed: true}) {
		t.Fatalf("got %+v", got)
	}
}

// AC-0
func TestResolveReviewEdit_Denied(t *testing.T) {
	cases := map[string]struct {
		ctx context.Context
		a   ports.Authorizer
	}{
		"kind 2":          {staffCtx(2, "op-1", false), strictAuthz{perms: withPerm}},
		"no staff id":     {staffCtx(7, "", false), strictAuthz{perms: withPerm}},
		"authz error":     {staffCtx(7, "staff-1", false), strictAuthz{perms: withPerm, err: errors.New("x")}},
		"no perm":         {staffCtx(7, "staff-1", false), strictAuthz{perms: map[string]bool{}}},
		"nil authorizer":  {staffCtx(7, "staff-1", false), nil},
		"no fresh method": {staffCtx(7, "staff-1", false), ports.Authorizer(nil)},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := resolveReviewEdit(tc.ctx, tc.a); got != (reviewedit.Decision{}) {
				t.Fatalf("got %+v", got)
			}
		})
	}
}

// AC-16
func TestResolveReviewEdit_WorkspaceScopeWithPerm(t *testing.T) {
	got := resolveReviewEdit(staffCtx(7, "staff-1", true), strictAuthz{perms: withPerm})
	if got != (reviewedit.Decision{Allowed: true, WorkspaceScope: true}) {
		t.Fatalf("got %+v", got)
	}
}

// AC-16: approval_scope:workspace WITHOUT the new permission grants nothing.
func TestResolveReviewEdit_WorkspaceScopeWithoutPerm(t *testing.T) {
	got := resolveReviewEdit(staffCtx(7, "staff-1", true), strictAuthz{perms: map[string]bool{}})
	if got != (reviewedit.Decision{}) {
		t.Fatalf("got %+v", got)
	}
}

// withReviewEdit stores the decision on the returned ctx.
func TestWithReviewEdit_StoresDecision(t *testing.T) {
	ctx := withReviewEdit(staffCtx(7, "staff-1", true), strictAuthz{perms: withPerm})
	d, ok := reviewedit.DecisionFromContext(ctx)
	if !ok || !d.Allowed || !d.WorkspaceScope {
		t.Fatalf("got %+v ok=%v", d, ok)
	}
	d, ok = reviewedit.DecisionFromContext(withReviewEdit(staffCtx(7, "staff-1", true), nil))
	if !ok || d.Allowed {
		t.Fatalf("nil authz must store zero decision, got %+v ok=%v", d, ok)
	}
}
