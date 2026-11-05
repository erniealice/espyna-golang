package reviewedit

import (
	"context"
	"errors"
	"testing"

	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/shared/identity"
)

type fakeChecker struct {
	has   bool
	err   error
	calls int
	code  string
}

func (f *fakeChecker) HasPermissionStrictFresh(_ context.Context, _ string, code string) (bool, error) {
	f.calls++
	f.code = code
	return f.has, f.err
}

func sessionCtx(kind int32, principalID string, wsScope bool) context.Context {
	ctx := contextutil.WithUserID(context.Background(), "user-1")
	ctx = identity.WithSessionBinding(ctx, kind, principalID, "", "")
	if wsScope {
		ctx = identity.WithWorkspaceRowScope(ctx)
	}
	return ctx
}

// AC-0: zero value denied, round trip.
func TestDecision_ZeroDeniedAndRoundTrip(t *testing.T) {
	if d, ok := DecisionFromContext(context.Background()); ok || d.Allowed || d.WorkspaceScope {
		t.Fatalf("missing decision must be zero/denied, got %+v ok=%v", d, ok)
	}
	want := Decision{Allowed: true, WorkspaceScope: true}
	got, ok := DecisionFromContext(WithDecision(context.Background(), want))
	if !ok || got != want {
		t.Fatalf("round trip: got %+v ok=%v", got, ok)
	}
}

// AC-0 / AC-16 resolver cases.
func TestResolve(t *testing.T) {
	cases := []struct {
		name string
		ctx  context.Context
		c    FreshChecker
		want Decision
	}{
		{"staff with perm", sessionCtx(7, "staff-1", false), &fakeChecker{has: true}, Decision{Allowed: true}},
		{"staff with perm and workspace scope", sessionCtx(7, "staff-1", true), &fakeChecker{has: true}, Decision{Allowed: true, WorkspaceScope: true}},
		{"workspace scope without perm", sessionCtx(7, "staff-1", true), &fakeChecker{has: false}, Decision{}},
		{"kind 2", sessionCtx(2, "op-1", true), &fakeChecker{has: true}, Decision{}},
		{"kind 0 no binding", contextutil.WithUserID(context.Background(), "user-1"), &fakeChecker{has: true}, Decision{}},
		{"empty principal id", sessionCtx(7, "", true), &fakeChecker{has: true}, Decision{}},
		{"checker error", sessionCtx(7, "staff-1", true), &fakeChecker{has: true, err: errors.New("boom")}, Decision{}},
		{"nil checker", sessionCtx(7, "staff-1", true), nil, Decision{}},
		{"no user", identity.WithSessionBinding(context.Background(), 7, "staff-1", "", ""), &fakeChecker{has: true}, Decision{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Resolve(tc.ctx, tc.c); got != tc.want {
				t.Fatalf("got %+v want %+v", got, tc.want)
			}
		})
	}
}

func TestResolve_AsksForPermissionCode(t *testing.T) {
	c := &fakeChecker{has: true}
	Resolve(sessionCtx(7, "staff-1", false), c)
	if c.calls != 1 || c.code != PermissionCode || PermissionCode != "task_outcome:update_in_review_scope" {
		t.Fatalf("calls=%d code=%q", c.calls, c.code)
	}
}
