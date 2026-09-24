package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erniealice/espyna-golang/shared/identity"
)

// Plan 20260924-approval-role-workflow D3: the permission installer marks the
// request with the workspace row scope ONLY when the loaded (binding-narrowed)
// codes include approval_scope:workspace.

type codesLoader struct{ codes []string }

func (l codesLoader) GetUserPermissionCodes(context.Context, string, string, PrincipalType, string, string, string) ([]string, error) {
	return l.codes, nil
}
func (l codesLoader) IsEnabled() bool { return true }

func injectWithCodes(t *testing.T, codes []string, hint *PermissionBindingHint) context.Context {
	t.Helper()
	a := NewViewAdapter(nil, "v-test", nil, nil, nil, nil, nil, nil, codesLoader{codes: codes}, nil, nil, "", "", "", nil)
	if hint != nil {
		h := *hint
		a.SetPrincipalLookup(func(*http.Request) PermissionBindingHint { return h })
	}
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "u1", WorkspaceID: "ws1"})
	req := httptest.NewRequest(http.MethodGet, "/demo", nil).WithContext(ctx)
	return a.injectRequestContext(req).Context()
}

func TestInjectRequestContext_WorkspaceRowScopeMarker(t *testing.T) {
	scope := []string{"job_phase:verify", identity.WorkspaceRowScopePermission}
	staff := &PermissionBindingHint{Kind: PrincipalType(7), BindingID: "staff-1"}
	if !identity.HasWorkspaceRowScope(injectWithCodes(t, scope, staff)) {
		t.Fatal("a STAFF binding whose binding-scoped codes include approval_scope:workspace must mark the request")
	}
	if identity.HasWorkspaceRowScope(injectWithCodes(t, []string{"job_phase:verify", "job_phase:publish"}, staff)) {
		t.Fatal("without approval_scope:workspace the request must stay unmarked (fail closed)")
	}
	if identity.HasWorkspaceRowScope(injectWithCodes(t, scope, nil)) {
		t.Fatal("the legacy union path (no principal lookup) must never mint the marker")
	}
	operator := &PermissionBindingHint{Kind: PrincipalType(2), BindingID: "wu-1"}
	if identity.HasWorkspaceRowScope(injectWithCodes(t, scope, operator)) {
		t.Fatal("only STAFF bindings are marked (operators are not row-scoped anyway)")
	}
	if identity.HasWorkspaceRowScope(injectWithCodes(t, scope, &PermissionBindingHint{})) {
		t.Fatal("an empty binding hint installs empty perms and must stay unmarked")
	}
}
