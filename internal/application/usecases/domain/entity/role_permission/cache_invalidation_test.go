package role_permission

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	clientportalgrantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_portal_grant"
	delegateclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_client"
	delegatesupplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_supplier"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
	rolepermissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role_permission"
	supplierportalgrantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_portal_grant"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

// ─── invalidation fakes ──────────────────────────────────────────────────────

// fakeWURRepo overrides only ListWorkspaceUserRoles and records the role filter.
type fakeWURRepo struct {
	workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer
	resp          *workspaceuserrolepb.ListWorkspaceUserRolesResponse
	err           error
	sawFilterRole string
}

func (f *fakeWURRepo) ListWorkspaceUserRoles(_ context.Context, req *workspaceuserrolepb.ListWorkspaceUserRolesRequest) (*workspaceuserrolepb.ListWorkspaceUserRolesResponse, error) {
	if req != nil && req.Filters != nil {
		for _, tf := range req.Filters.Filters {
			if tf.GetField() == "role_id" {
				f.sawFilterRole = tf.GetStringFilter().GetValue()
			}
		}
	}
	return f.resp, f.err
}

// fakeWURepo resolves workspace_user id → login user id (for the staff facet).
type fakeWURepo struct {
	workspaceuserpb.WorkspaceUserDomainServiceServer
	byID map[string]string // workspace_user_id → user_id
}

func (f *fakeWURepo) ReadWorkspaceUser(_ context.Context, req *workspaceuserpb.ReadWorkspaceUserRequest) (*workspaceuserpb.ReadWorkspaceUserResponse, error) {
	id := req.GetData().GetId()
	uid, ok := f.byID[id]
	if !ok {
		return &workspaceuserpb.ReadWorkspaceUserResponse{Success: false}, nil
	}
	return &workspaceuserpb.ReadWorkspaceUserResponse{
		Success: true,
		Data:    []*workspaceuserpb.WorkspaceUser{{Id: id, UserId: uid, Active: true}},
	}, nil
}

type fakeCPGRepo struct {
	clientportalgrantpb.ClientPortalGrantDomainServiceServer
	rows []*clientportalgrantpb.ClientPortalGrant
}

func (f *fakeCPGRepo) ListClientPortalGrants(_ context.Context, _ *clientportalgrantpb.ListClientPortalGrantsRequest) (*clientportalgrantpb.ListClientPortalGrantsResponse, error) {
	return &clientportalgrantpb.ListClientPortalGrantsResponse{Data: f.rows}, nil
}

type fakeSPGRepo struct {
	supplierportalgrantpb.SupplierPortalGrantDomainServiceServer
	rows []*supplierportalgrantpb.SupplierPortalGrant
}

func (f *fakeSPGRepo) ListSupplierPortalGrants(_ context.Context, _ *supplierportalgrantpb.ListSupplierPortalGrantsRequest) (*supplierportalgrantpb.ListSupplierPortalGrantsResponse, error) {
	return &supplierportalgrantpb.ListSupplierPortalGrantsResponse{Data: f.rows}, nil
}

type fakeDCRepo struct {
	delegateclientpb.DelegateClientDomainServiceServer
	rows []*delegateclientpb.DelegateClient
}

func (f *fakeDCRepo) ListDelegateClients(_ context.Context, _ *delegateclientpb.ListDelegateClientsRequest) (*delegateclientpb.ListDelegateClientsResponse, error) {
	return &delegateclientpb.ListDelegateClientsResponse{Data: f.rows}, nil
}

type fakeDSRepo struct {
	delegatesupplierpb.DelegateSupplierDomainServiceServer
	rows []*delegatesupplierpb.DelegateSupplier
}

func (f *fakeDSRepo) ListDelegateSuppliers(_ context.Context, _ *delegatesupplierpb.ListDelegateSuppliersRequest) (*delegatesupplierpb.ListDelegateSuppliersResponse, error) {
	return &delegatesupplierpb.ListDelegateSuppliersResponse{Data: f.rows}, nil
}

// recorder records the ids invalidated at each grain.
type recorder struct {
	users    []string
	bindings []string
}

func (r *recorder) InvalidateUser(id string)    { r.users = append(r.users, id) }
func (r *recorder) InvalidateBinding(id string) { r.bindings = append(r.bindings, id) }

func (r *recorder) hasBinding(id string) bool { return contains(r.bindings, id) }
func (r *recorder) hasUser(id string) bool    { return contains(r.users, id) }

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

func wur(workspaceUserID, roleID string, active bool) *workspaceuserrolepb.WorkspaceUserRole {
	return &workspaceuserrolepb.WorkspaceUserRole{WorkspaceUserId: workspaceUserID, RoleId: roleID, Active: active}
}

// ─── invalidation tests ──────────────────────────────────────────────────────

// TestInvalidateRoleBindings_EvictsActiveMatchingBindings is the P10/D2 core:
// a role→permission grant/revoke must invalidate EXACTLY the active operator
// bindings assigned that role via workspace_user_role — not inactive rows, not
// rows carrying a different role, and never a whole-cache flush. With no
// workspace_user repo wired, only the InvalidateBinding (operator) grain fires.
func TestInvalidateRoleBindings_EvictsActiveMatchingBindings(t *testing.T) {
	repo := &fakeWURRepo{resp: &workspaceuserrolepb.ListWorkspaceUserRolesResponse{Data: []*workspaceuserrolepb.WorkspaceUserRole{
		wur("wu-1", "role-A", true),  // in
		wur("wu-2", "role-A", true),  // in
		wur("wu-3", "role-A", false), // out: inactive
		wur("wu-4", "role-B", true),  // out: different role (adapter ignored filter)
		wur("", "role-A", true),      // out: empty binding id
	}}}
	rec := &recorder{}

	invalidateRoleBindings(context.Background(), roleBindingInvalidationRepos{WorkspaceUserRole: repo}, rec, "role-A")

	if repo.sawFilterRole != "role-A" {
		t.Fatalf("list was not scoped to role-A, saw filter %q", repo.sawFilterRole)
	}
	if got, want := len(rec.bindings), 2; got != want {
		t.Fatalf("evicted %d bindings, want %d (%v)", got, want, rec.bindings)
	}
	if !rec.hasBinding("wu-1") || !rec.hasBinding("wu-2") {
		t.Fatalf("expected wu-1 and wu-2 evicted, got %v", rec.bindings)
	}
	if len(rec.users) != 0 {
		t.Fatalf("no workspace_user repo wired: expected no user evictions, got %v", rec.users)
	}
}

// TestInvalidateRoleBindings_StaffLoginUserResolved verifies the CF-1 staff
// (kind 7) coverage: when a workspace_user repo is wired, each active WUR
// binding's login user is resolved and evicted via InvalidateUser (staff's
// cache-key bindingID is staff.id, not workspace_user.id, so InvalidateBinding
// alone misses it). The operator InvalidateBinding still fires precisely.
func TestInvalidateRoleBindings_StaffLoginUserResolved(t *testing.T) {
	repo := &fakeWURRepo{resp: &workspaceuserrolepb.ListWorkspaceUserRolesResponse{Data: []*workspaceuserrolepb.WorkspaceUserRole{
		wur("wu-1", "role-A", true),
		wur("wu-9", "role-A", false), // inactive: not resolved
	}}}
	wu := &fakeWURepo{byID: map[string]string{"wu-1": "user-teacher", "wu-9": "user-x"}}
	rec := &recorder{}

	invalidateRoleBindings(context.Background(), roleBindingInvalidationRepos{WorkspaceUserRole: repo, WorkspaceUser: wu}, rec, "role-A")

	if !rec.hasBinding("wu-1") {
		t.Fatalf("operator binding wu-1 not evicted: %v", rec.bindings)
	}
	if !rec.hasUser("user-teacher") {
		t.Fatalf("staff login user of wu-1 not evicted: %v", rec.users)
	}
	if rec.hasUser("user-x") {
		t.Fatalf("inactive binding's login user must not be evicted: %v", rec.users)
	}
}

// TestInvalidateRoleBindings_CoversPortalAndDelegate verifies the CF-1 portal +
// delegate coverage: a role held through a CLIENT/SUPPLIER portal grant or a
// CLIENT/SUPPLIER delegate is evicted on its own cache-key bindingID
// (portal_grant.id / delegate.id) — the representations InvalidateBinding on a
// workspace_user id could never match. Inactive rows and foreign-role rows
// (adapter ignored the filter) are skipped.
func TestInvalidateRoleBindings_CoversPortalAndDelegate(t *testing.T) {
	cpg := &fakeCPGRepo{rows: []*clientportalgrantpb.ClientPortalGrant{
		{Id: "cpg-1", RoleId: "role-A", Active: true},  // in
		{Id: "cpg-2", RoleId: "role-A", Active: false}, // out: inactive
		{Id: "cpg-3", RoleId: "role-B", Active: true},  // out: different role
	}}
	spg := &fakeSPGRepo{rows: []*supplierportalgrantpb.SupplierPortalGrant{
		{Id: "spg-1", RoleId: "role-A", Active: true},
	}}
	dc := &fakeDCRepo{rows: []*delegateclientpb.DelegateClient{
		{Id: "dc-1", DelegateId: "del-1", RoleId: strptr("role-A"), Active: true},
		{Id: "dc-2", DelegateId: "del-9", RoleId: strptr("role-A"), Active: false}, // out: inactive
	}}
	ds := &fakeDSRepo{rows: []*delegatesupplierpb.DelegateSupplier{
		{Id: "ds-1", DelegateId: "del-2", RoleId: "role-A", Active: true},
	}}
	rec := &recorder{}

	invalidateRoleBindings(context.Background(), roleBindingInvalidationRepos{
		ClientPortalGrant:   cpg,
		SupplierPortalGrant: spg,
		DelegateClient:      dc,
		DelegateSupplier:    ds,
	}, rec, "role-A")

	for _, want := range []string{"cpg-1", "spg-1", "del-1", "del-2"} {
		if !rec.hasBinding(want) {
			t.Fatalf("expected binding %q evicted, got %v", want, rec.bindings)
		}
	}
	for _, deny := range []string{"cpg-2", "cpg-3", "del-9"} {
		if rec.hasBinding(deny) {
			t.Fatalf("binding %q must NOT be evicted (inactive/foreign role), got %v", deny, rec.bindings)
		}
	}
	if got := len(rec.bindings); got != 4 {
		t.Fatalf("expected exactly 4 bindings evicted, got %d (%v)", got, rec.bindings)
	}
}

// TestInvalidateRoleBindings_NilSafe verifies every unwired / empty / failing
// input path is a silent no-op — invalidation is best-effort and must never
// break or widen the grant it follows.
func TestInvalidateRoleBindings_NilSafe(t *testing.T) {
	rec := &recorder{}
	okRepo := &fakeWURRepo{resp: &workspaceuserrolepb.ListWorkspaceUserRolesResponse{Data: []*workspaceuserrolepb.WorkspaceUserRole{wur("wu-1", "role-A", true)}}}

	// entirely empty repos
	invalidateRoleBindings(context.Background(), roleBindingInvalidationRepos{}, rec, "role-A")
	// nil invalidator
	invalidateRoleBindings(context.Background(), roleBindingInvalidationRepos{WorkspaceUserRole: okRepo}, nil, "role-A")
	// empty role id
	invalidateRoleBindings(context.Background(), roleBindingInvalidationRepos{WorkspaceUserRole: okRepo}, rec, "")
	// list error
	invalidateRoleBindings(context.Background(), roleBindingInvalidationRepos{WorkspaceUserRole: &fakeWURRepo{err: errors.New("boom")}}, rec, "role-A")
	// nil response
	invalidateRoleBindings(context.Background(), roleBindingInvalidationRepos{WorkspaceUserRole: &fakeWURRepo{}}, rec, "role-A")

	if len(rec.bindings) != 0 || len(rec.users) != 0 {
		t.Fatalf("expected no evictions on nil-safe paths, got bindings=%v users=%v", rec.bindings, rec.users)
	}
}

// ─── C14 reactivate-on-create roundtrip ──────────────────────────────────────

// fakeRolePermissionRepo mirrors the LIVE postgres adapter semantics that the
// C14 fix depends on (verified against contrib/postgres adapter/core/operations.go):
//   - CreateRolePermission enforces uq_role_permission_1 (role_id, permission_id)
//     with NO active filter, so a soft-deleted pair still blocks a plain INSERT
//     (this is the 422 the fix must avoid).
//   - ListRolePermissions DEFAULTS to active=true unless an explicit active
//     BooleanFilter is supplied, in which case it honours the requested value
//     (so soft-deleted rows are only visible with active=false — the C10 lesson).
//   - UpdateRolePermission re-activates by id and preserves the row identity.
//   - DeleteRolePermission soft-deletes (active=false).
type fakeRolePermissionRepo struct {
	rolepermissionpb.RolePermissionDomainServiceServer
	rows        []*rolepermissionpb.RolePermission
	createCalls int
	updateCalls int
}

func (f *fakeRolePermissionRepo) find(roleID, permID string) *rolepermissionpb.RolePermission {
	for _, r := range f.rows {
		if r.GetRoleId() == roleID && r.GetPermissionId() == permID {
			return r
		}
	}
	return nil
}

func (f *fakeRolePermissionRepo) CreateRolePermission(_ context.Context, req *rolepermissionpb.CreateRolePermissionRequest) (*rolepermissionpb.CreateRolePermissionResponse, error) {
	f.createCalls++
	d := req.GetData()
	if f.find(d.GetRoleId(), d.GetPermissionId()) != nil {
		// uq_role_permission_1 has no active filter — active OR inactive collides.
		return nil, errors.New(`duplicate key value violates unique constraint "uq_role_permission_1"`)
	}
	row := &rolepermissionpb.RolePermission{
		Id:             d.GetId(),
		RoleId:         d.GetRoleId(),
		PermissionId:   d.GetPermissionId(),
		PermissionType: d.GetPermissionType(),
		Active:         d.GetActive(),
	}
	f.rows = append(f.rows, row)
	return &rolepermissionpb.CreateRolePermissionResponse{Success: true, Data: []*rolepermissionpb.RolePermission{row}}, nil
}

func (f *fakeRolePermissionRepo) ListRolePermissions(_ context.Context, req *rolepermissionpb.ListRolePermissionsRequest) (*rolepermissionpb.ListRolePermissionsResponse, error) {
	hasActiveFilter, wantActive := false, true
	roleID := ""
	if req.GetFilters() != nil {
		for _, tf := range req.GetFilters().GetFilters() {
			switch tf.GetField() {
			case "active":
				if bf := tf.GetBooleanFilter(); bf != nil {
					hasActiveFilter, wantActive = true, bf.GetValue()
				}
			case "role_id":
				if sf := tf.GetStringFilter(); sf != nil {
					roleID = sf.GetValue()
				}
			}
		}
	}
	out := []*rolepermissionpb.RolePermission{}
	for _, r := range f.rows {
		if roleID != "" && r.GetRoleId() != roleID {
			continue
		}
		if hasActiveFilter {
			if r.GetActive() != wantActive {
				continue
			}
		} else if !r.GetActive() {
			continue // default: active = true only
		}
		out = append(out, r)
	}
	return &rolepermissionpb.ListRolePermissionsResponse{Success: true, Data: out}, nil
}

func (f *fakeRolePermissionRepo) UpdateRolePermission(_ context.Context, req *rolepermissionpb.UpdateRolePermissionRequest) (*rolepermissionpb.UpdateRolePermissionResponse, error) {
	f.updateCalls++
	d := req.GetData()
	for _, r := range f.rows {
		if r.GetId() == d.GetId() {
			r.Active = d.GetActive()
			r.PermissionType = d.GetPermissionType()
			return &rolepermissionpb.UpdateRolePermissionResponse{Success: true, Data: []*rolepermissionpb.RolePermission{r}}, nil
		}
	}
	return nil, errors.New("record not found")
}

func (f *fakeRolePermissionRepo) DeleteRolePermission(_ context.Context, req *rolepermissionpb.DeleteRolePermissionRequest) (*rolepermissionpb.DeleteRolePermissionResponse, error) {
	for _, r := range f.rows {
		if r.GetId() == req.GetData().GetId() {
			r.Active = false
			return &rolepermissionpb.DeleteRolePermissionResponse{Success: true}, nil
		}
	}
	return nil, errors.New("record not found")
}

func (f *fakeRolePermissionRepo) activeCount(roleID, permID string) int {
	n := 0
	for _, r := range f.rows {
		if r.GetRoleId() == roleID && r.GetPermissionId() == permID && r.GetActive() {
			n++
		}
	}
	return n
}

// reference fakes for validateEntityReferences (always active).
type fakeRoleRepo struct{ rolepb.RoleDomainServiceServer }

func (fakeRoleRepo) ReadRole(_ context.Context, req *rolepb.ReadRoleRequest) (*rolepb.ReadRoleResponse, error) {
	return &rolepb.ReadRoleResponse{Success: true, Data: []*rolepb.Role{{Id: req.GetData().GetId(), Active: true}}}, nil
}

type fakePermRepo struct{ permissionpb.PermissionDomainServiceServer }

func (fakePermRepo) ReadPermission(_ context.Context, req *permissionpb.ReadPermissionRequest) (*permissionpb.ReadPermissionResponse, error) {
	return &permissionpb.ReadPermissionResponse{Success: true, Data: []*permissionpb.Permission{{Id: req.GetData().GetId(), Active: true}}}, nil
}

// stubAuthorizer satisfies actiongate.Authorizer; IsEnabled=false short-circuits
// the gatekeeper to "allow" without needing a real permission backend.
type stubAuthorizer struct{}

func (stubAuthorizer) HasPermission(context.Context, string, string) (bool, error) { return true, nil }
func (stubAuthorizer) IsEnabled() bool                                             { return false }

type seqIDGen struct{ n int }

func (g *seqIDGen) GenerateID() string                   { g.n++; return fmt.Sprintf("rp-%d", g.n) }
func (g *seqIDGen) GenerateIDWithPrefix(p string) string { g.n++; return fmt.Sprintf("%s-%d", p, g.n) }
func (g *seqIDGen) IsEnabled() bool                      { return true }
func (g *seqIDGen) GetProviderInfo() string              { return "seq" }

func strptr(s string) *string { return &s }

func newRoundtripUseCases(repo *fakeRolePermissionRepo) (*CreateRolePermissionUseCase, *DeleteRolePermissionUseCase) {
	gate := actiongate.NewActionGatekeeper(stubAuthorizer{}, ports.NewNoOpTranslator())
	create := NewCreateRolePermissionUseCase(
		CreateRolePermissionRepositories{RolePermission: repo, Role: fakeRoleRepo{}, Permission: fakePermRepo{}},
		CreateRolePermissionServices{
			Transactor:       ports.NewNoOpTransactor(),
			Translator:       ports.NewNoOpTranslator(),
			ActionGatekeeper: gate,
			IDGenerator:      &seqIDGen{},
		},
	)
	del := NewDeleteRolePermissionUseCase(
		DeleteRolePermissionRepositories{RolePermission: repo},
		DeleteRolePermissionServices{
			Transactor:       ports.NewNoOpTransactor(),
			Translator:       ports.NewNoOpTranslator(),
			ActionGatekeeper: gate,
		},
	)
	return create, del
}

// TestCreateRolePermission_ReactivatesAfterRemove is the C14 roundtrip: grant →
// remove (soft-delete) → re-grant the SAME (role, permission). The re-grant must
// SUCCEED by reactivating the soft-deleted row (not INSERT a duplicate that
// collides on uq_role_permission_1 and 422s), leaving exactly one active row.
func TestCreateRolePermission_ReactivatesAfterRemove(t *testing.T) {
	repo := &fakeRolePermissionRepo{}
	create, del := newRoundtripUseCases(repo)
	ctx := context.Background()
	const roleID, permID = "role-admin", "perm-client-read"

	// 1) Grant.
	first, err := create.Execute(ctx, &rolepermissionpb.CreateRolePermissionRequest{
		Data: &rolepermissionpb.RolePermission{RoleId: roleID, PermissionId: permID},
	})
	if err != nil {
		t.Fatalf("initial grant failed: %v", err)
	}
	grantID := first.GetData()[0].GetId()
	if grantID == "" {
		t.Fatalf("expected a generated grant id, got empty")
	}
	if repo.activeCount(roleID, permID) != 1 {
		t.Fatalf("after grant: want 1 active row, got %d", repo.activeCount(roleID, permID))
	}

	// 2) Remove (soft-delete).
	if _, err := del.Execute(ctx, &rolepermissionpb.DeleteRolePermissionRequest{
		Data: &rolepermissionpb.RolePermission{Id: grantID, RoleId: roleID},
	}); err != nil {
		t.Fatalf("remove failed: %v", err)
	}
	if repo.activeCount(roleID, permID) != 0 {
		t.Fatalf("after remove: want 0 active rows, got %d", repo.activeCount(roleID, permID))
	}

	// 3) Re-grant the same pair — must reactivate, not 422.
	second, err := create.Execute(ctx, &rolepermissionpb.CreateRolePermissionRequest{
		Data: &rolepermissionpb.RolePermission{RoleId: roleID, PermissionId: permID},
	})
	if err != nil {
		t.Fatalf("re-grant after remove must reactivate, got error: %v", err)
	}
	if !second.GetSuccess() || len(second.GetData()) != 1 || !second.GetData()[0].GetActive() {
		t.Fatalf("re-grant response not an active row: %+v", second)
	}
	if got := second.GetData()[0].GetId(); got != grantID {
		t.Fatalf("re-grant must reuse the soft-deleted row id %q, got %q", grantID, got)
	}
	if repo.activeCount(roleID, permID) != 1 {
		t.Fatalf("after re-grant: want exactly 1 active row (no duplicate), got %d", repo.activeCount(roleID, permID))
	}
	if len(repo.rows) != 1 {
		t.Fatalf("re-grant must not INSERT a duplicate: want 1 row total, got %d", len(repo.rows))
	}
	if repo.updateCalls != 1 {
		t.Fatalf("re-grant must take the reactivate (Update) path exactly once, got %d update calls", repo.updateCalls)
	}
}

// TestCreateRolePermission_FreshGrantInserts confirms the non-reactivate path is
// unchanged: a first-ever grant plain-INSERTs (no spurious Update).
func TestCreateRolePermission_FreshGrantInserts(t *testing.T) {
	repo := &fakeRolePermissionRepo{}
	create, _ := newRoundtripUseCases(repo)

	resp, err := create.Execute(context.Background(), &rolepermissionpb.CreateRolePermissionRequest{
		Data: &rolepermissionpb.RolePermission{RoleId: "role-x", PermissionId: "perm-y"},
	})
	if err != nil {
		t.Fatalf("fresh grant failed: %v", err)
	}
	if !resp.GetData()[0].GetActive() {
		t.Fatalf("fresh grant should be active")
	}
	if repo.createCalls != 1 || repo.updateCalls != 0 {
		t.Fatalf("fresh grant must INSERT (create=1,update=0), got create=%d update=%d", repo.createCalls, repo.updateCalls)
	}
}
