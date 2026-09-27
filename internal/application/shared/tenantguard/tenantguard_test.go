package tenantguard

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"

	"github.com/erniealice/espyna-golang/shared/identity"
)

// fakeWorkspaceUsers ignores request filters on purpose: the guard must
// re-check every returned row in memory (an adapter in AUTHZ shadow mode, or
// one that drops a filter, must not widen the result).
type fakeWorkspaceUsers struct {
	rows []*workspaceuserpb.WorkspaceUser
}

func (f fakeWorkspaceUsers) ListWorkspaceUsers(context.Context, *workspaceuserpb.ListWorkspaceUsersRequest) (*workspaceuserpb.ListWorkspaceUsersResponse, error) {
	return &workspaceuserpb.ListWorkspaceUsersResponse{Data: f.rows}, nil
}

func (f fakeWorkspaceUsers) ReadWorkspaceUser(_ context.Context, req *workspaceuserpb.ReadWorkspaceUserRequest) (*workspaceuserpb.ReadWorkspaceUserResponse, error) {
	for _, r := range f.rows {
		if r.GetId() == req.GetData().GetId() {
			return &workspaceuserpb.ReadWorkspaceUserResponse{Data: []*workspaceuserpb.WorkspaceUser{r}}, nil
		}
	}
	return &workspaceuserpb.ReadWorkspaceUserResponse{}, nil
}

type fakeRoles struct{ rows []*rolepb.Role }

func (f fakeRoles) ReadRole(_ context.Context, req *rolepb.ReadRoleRequest) (*rolepb.ReadRoleResponse, error) {
	for _, r := range f.rows {
		if r.GetId() == req.GetData().GetId() {
			return &rolepb.ReadRoleResponse{Data: []*rolepb.Role{r}}, nil
		}
	}
	return &rolepb.ReadRoleResponse{}, nil
}

func ctxIn(ws string) context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "actor", WorkspaceID: ws})
}

func strp(s string) *string { return &s }

var members = fakeWorkspaceUsers{rows: []*workspaceuserpb.WorkspaceUser{
	{Id: "wu-a1", WorkspaceId: "ws-a", UserId: "u1", Active: true},
	{Id: "wu-b2", WorkspaceId: "ws-b", UserId: "u2", Active: true},
	{Id: "wu-a3", WorkspaceId: "ws-a", UserId: "u3", Active: false},
}}

func TestRequireActiveMember_TwoWorkspaceMatrix(t *testing.T) {
	cases := []struct {
		name   string
		ctx    context.Context
		userID string
		ok     bool
	}{
		{"member of actor workspace", ctxIn("ws-a"), "u1", true},
		{"member of another workspace only", ctxIn("ws-a"), "u2", false},
		{"inactive membership", ctxIn("ws-a"), "u3", false},
		{"unknown user", ctxIn("ws-a"), "nobody", false},
		{"no workspace selected", ctxIn(""), "u1", false},
		{"no identity", context.Background(), "u1", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := RequireActiveMember(tc.ctx, members, tc.userID)
			if (err == nil) != tc.ok {
				t.Fatalf("RequireActiveMember(%q) err=%v, want ok=%v", tc.userID, err, tc.ok)
			}
		})
	}
	if err := RequireActiveMember(ctxIn("ws-a"), nil, "u1"); err == nil {
		t.Fatal("nil repository must deny")
	}
}

func TestRequireWorkspaceUserInTenant(t *testing.T) {
	if wu, err := RequireWorkspaceUserInTenant(ctxIn("ws-a"), members, "wu-a1"); err != nil || wu.GetId() != "wu-a1" {
		t.Fatalf("own binding: wu=%v err=%v", wu, err)
	}
	if _, err := RequireWorkspaceUserInTenant(ctxIn("ws-a"), members, "wu-b2"); !errors.Is(err, ErrOutsideTenant) {
		t.Fatalf("foreign binding must be ErrOutsideTenant, got %v", err)
	}
	if _, err := RequireWorkspaceUserInTenant(ctxIn("ws-a"), members, "missing"); !errors.Is(err, ErrOutsideTenant) {
		t.Fatalf("missing binding must be ErrOutsideTenant, got %v", err)
	}
}

func TestRequireRoleInTenant(t *testing.T) {
	roles := fakeRoles{rows: []*rolepb.Role{
		{Id: "r-a", WorkspaceId: strp("ws-a")},
		{Id: "r-b", WorkspaceId: strp("ws-b")},
		{Id: "r-global"},
	}}
	if err := RequireRoleInTenant(ctxIn("ws-a"), roles, "r-a"); err != nil {
		t.Fatalf("own role: %v", err)
	}
	for _, id := range []string{"r-b", "r-global", "missing"} {
		if err := RequireRoleInTenant(ctxIn("ws-a"), roles, id); !errors.Is(err, ErrOutsideTenant) {
			t.Fatalf("role %s must be ErrOutsideTenant, got %v", id, err)
		}
	}
}

func TestRequireActorWorkspace(t *testing.T) {
	if err := RequireActorWorkspace(ctxIn("ws-a"), "ws-a"); err != nil {
		t.Fatalf("own workspace: %v", err)
	}
	for _, id := range []string{"ws-b", ""} {
		if err := RequireActorWorkspace(ctxIn("ws-a"), id); err == nil {
			t.Fatalf("workspace %q must be denied", id)
		}
	}
	if err := RequireActorWorkspace(ctxIn(""), ""); err == nil {
		t.Fatal("no selected workspace must be denied")
	}
}

func TestRequirePlatformOperator_DeniesTenantPath(t *testing.T) {
	if err := RequirePlatformOperator(ctxIn("ws-a")); !errors.Is(err, ErrPlatformOperation) {
		t.Fatalf("want ErrPlatformOperation, got %v", err)
	}
}

func TestPlatformOperatorMarker(t *testing.T) {
	if IsPlatformOperator(context.Background()) {
		t.Fatal("plain context must not be a platform operator")
	}
	for _, c := range [][2]string{{"", "reason"}, {"op", ""}} {
		if IsPlatformOperator(WithPlatformOperator(context.Background(), c[0], c[1])) {
			t.Fatalf("marker without operator and reason must not apply: %v", c)
		}
	}
	ctx := WithPlatformOperator(ctxIn("ws-a"), "ops@example.test", "ticket-123")
	if err := RequirePlatformOperator(ctx); err != nil {
		t.Fatalf("marked context must pass: %v", err)
	}
}

// TestNoProductionCallerOfWithPlatformOperator keeps the control-plane marker
// out of the request path. Wave 0 has no production caller at all; the P1
// control-plane CLI (cmd/platform-ops) is the only path that may be added here.
func TestNoProductionCallerOfWithPlatformOperator(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	allowed := map[string]bool{}
	var offenders []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == ".git" || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if strings.Contains(filepath.ToSlash(path), "internal/application/shared/tenantguard/") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), "WithPlatformOperator(") && !allowed[filepath.ToSlash(path)] {
			offenders = append(offenders, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(offenders) > 0 {
		t.Fatalf("WithPlatformOperator must not be called from production code: %v", offenders)
	}
}

func TestRequireReadableUser(t *testing.T) {
	self := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "u2", WorkspaceID: "ws-a"})
	if err := RequireReadableUser(self, members, "u2"); err != nil {
		t.Fatalf("self read must pass even outside the workspace: %v", err)
	}
	if err := RequireReadableUser(ctxIn("ws-a"), members, "u1"); err != nil {
		t.Fatalf("member read must pass: %v", err)
	}
	if err := RequireReadableUser(ctxIn("ws-a"), members, "u2"); !errors.Is(err, ErrOutsideTenant) {
		t.Fatalf("foreign read must be ErrOutsideTenant, got %v", err)
	}
	if err := RequireReadableUser(WithPlatformOperator(ctxIn("ws-a"), "ops", "ticket"), members, "u2"); err != nil {
		t.Fatalf("operator read must pass: %v", err)
	}
}
