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
		// fakeChecker answers the same for every code, so all three capabilities follow.
		{"staff with perm", sessionCtx(7, "staff-1", false), &fakeChecker{has: true}, Decision{Allowed: true, Fill: true, Clear: true}},
		{"staff with perm and workspace scope", sessionCtx(7, "staff-1", true), &fakeChecker{has: true}, Decision{Allowed: true, Fill: true, Clear: true, WorkspaceScope: true}},
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

func TestResolve_AsksForEachCode(t *testing.T) {
	c := &codeChecker{perms: map[string]bool{PermissionCode: true}}
	Resolve(sessionCtx(7, "staff-1", false), c)
	want := []string{PermissionCode, CreateCode, DeleteCode}
	if len(c.asked) != len(want) {
		t.Fatalf("asked %v want %v", c.asked, want)
	}
	for i := range want {
		if c.asked[i] != want[i] {
			t.Fatalf("asked %v want %v", c.asked, want)
		}
	}
	if PermissionCode != "task_outcome:update_in_review_scope" ||
		CreateCode != "task_outcome:create_in_review_scope" ||
		DeleteCode != "task_outcome:delete_in_review_scope" {
		t.Fatalf("codes changed: %q %q %q", PermissionCode, CreateCode, DeleteCode)
	}
}

// codeChecker answers per code and records what was asked.
type codeChecker struct {
	perms map[string]bool
	errs  map[string]error
	asked []string
}

func (c *codeChecker) HasPermissionStrictFresh(_ context.Context, _ string, code string) (bool, error) {
	c.asked = append(c.asked, code)
	return c.perms[code], c.errs[code]
}

// AC-O1: Fill / Clear only with their own fresh codes, kind 7 only.
func TestResolve_FillClear(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		name  string
		ctx   context.Context
		perms map[string]bool
		errs  map[string]error
		want  Decision
	}{
		{"create only", sessionCtx(7, "s", false), map[string]bool{CreateCode: true}, nil, Decision{Fill: true}},
		{"delete only", sessionCtx(7, "s", false), map[string]bool{DeleteCode: true}, nil, Decision{Clear: true}},
		{"all three ws", sessionCtx(7, "s", true), map[string]bool{PermissionCode: true, CreateCode: true, DeleteCode: true}, nil, Decision{Allowed: true, Fill: true, Clear: true, WorkspaceScope: true}},
		{"create ws", sessionCtx(7, "s", true), map[string]bool{CreateCode: true}, nil, Decision{Fill: true, WorkspaceScope: true}},
		{"create errors, update kept", sessionCtx(7, "s", false), map[string]bool{PermissionCode: true, CreateCode: true}, map[string]error{CreateCode: boom}, Decision{Allowed: true}},
		{"delete errors only", sessionCtx(7, "s", true), map[string]bool{DeleteCode: true}, map[string]error{DeleteCode: boom}, Decision{}},
		{"update does not imply fill/clear", sessionCtx(7, "s", false), map[string]bool{PermissionCode: true}, nil, Decision{Allowed: true}},
		{"kind 2 with all", sessionCtx(2, "op", true), map[string]bool{PermissionCode: true, CreateCode: true, DeleteCode: true}, nil, Decision{}},
		{"kind 7 no principal", sessionCtx(7, "", true), map[string]bool{CreateCode: true, DeleteCode: true}, nil, Decision{}},
		{"unrelated code", sessionCtx(7, "s", true), map[string]bool{"task_outcome:create": true, "task_outcome:delete": true}, nil, Decision{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Resolve(tc.ctx, &codeChecker{perms: tc.perms, errs: tc.errs})
			if got != tc.want {
				t.Fatalf("got %+v want %+v", got, tc.want)
			}
		})
	}
}

func TestDecision_Any(t *testing.T) {
	if (Decision{WorkspaceScope: true}).Any() {
		t.Fatal("WorkspaceScope alone must not be a capability")
	}
	for _, d := range []Decision{{Allowed: true}, {Fill: true}, {Clear: true}} {
		if !d.Any() {
			t.Fatalf("%+v.Any() = false", d)
		}
	}
}
