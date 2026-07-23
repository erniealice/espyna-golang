package workspace_user_role

import (
	"context"
	"testing"

	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

type recorder struct {
	users    []string
	bindings []string
}

func (r *recorder) InvalidateUser(id string)    { r.users = append(r.users, id) }
func (r *recorder) InvalidateBinding(id string) { r.bindings = append(r.bindings, id) }

// fakeWURRepo satisfies WorkspaceUserRoleDomainServiceServer via interface
// embedding, overriding only ReadWorkspaceUserRole (used by delete's binding
// resolution).
type fakeWURRepo struct {
	workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer
	readResp  *workspaceuserrolepb.ReadWorkspaceUserRoleResponse
	readErr   error
	readCalls int
}

func (f *fakeWURRepo) ReadWorkspaceUserRole(_ context.Context, _ *workspaceuserrolepb.ReadWorkspaceUserRoleRequest) (*workspaceuserrolepb.ReadWorkspaceUserRoleResponse, error) {
	f.readCalls++
	return f.readResp, f.readErr
}

// TestCreateInvalidateBindingCache verifies assigning a role evicts exactly that
// binding, and unwired / empty inputs are no-ops (P10/D2).
func TestCreateInvalidateBindingCache(t *testing.T) {
	rec := &recorder{}
	uc := &CreateWorkspaceUserRoleUseCase{services: CreateWorkspaceUserRoleServices{PermissionCacheInvalidator: rec}}
	uc.invalidateBindingCache("wu-7")
	uc.invalidateBindingCache("") // no-op

	if len(rec.bindings) != 1 || rec.bindings[0] != "wu-7" {
		t.Fatalf("expected only wu-7 evicted, got %v", rec.bindings)
	}

	// nil invalidator: no panic, no eviction.
	nilUC := &CreateWorkspaceUserRoleUseCase{}
	nilUC.invalidateBindingCache("wu-7")
}

// TestUpdateInvalidateBindingCache mirrors the create path for role changes.
func TestUpdateInvalidateBindingCache(t *testing.T) {
	rec := &recorder{}
	uc := &UpdateWorkspaceUserRoleUseCase{services: UpdateWorkspaceUserRoleServices{PermissionCacheInvalidator: rec}}
	uc.invalidateBindingCache("wu-8")
	if len(rec.bindings) != 1 || rec.bindings[0] != "wu-8" {
		t.Fatalf("expected only wu-8 evicted, got %v", rec.bindings)
	}
}

// TestDeleteResolveBindingID_PrefersRequest verifies the delete path uses the
// binding id already on the request without a read.
func TestDeleteResolveBindingID_PrefersRequest(t *testing.T) {
	repo := &fakeWURRepo{}
	uc := &DeleteWorkspaceUserRoleUseCase{repositories: DeleteWorkspaceUserRoleRepositories{WorkspaceUserRole: repo}}
	got := uc.resolveBindingID(context.Background(), &workspaceuserrolepb.DeleteWorkspaceUserRoleRequest{
		Data: &workspaceuserrolepb.WorkspaceUserRole{Id: "wur-1", WorkspaceUserId: "wu-req"},
	})
	if got != "wu-req" {
		t.Fatalf("expected request-supplied wu-req, got %q", got)
	}
	if repo.readCalls != 0 {
		t.Fatalf("expected no read when request carries the binding id, got %d reads", repo.readCalls)
	}
}

// TestDeleteResolveBindingID_ReadsRow verifies the delete path reads the row to
// resolve the binding id when the request does not carry it.
func TestDeleteResolveBindingID_ReadsRow(t *testing.T) {
	repo := &fakeWURRepo{readResp: &workspaceuserrolepb.ReadWorkspaceUserRoleResponse{Data: []*workspaceuserrolepb.WorkspaceUserRole{
		{Id: "wur-1", WorkspaceUserId: "wu-read"},
	}}}
	uc := &DeleteWorkspaceUserRoleUseCase{repositories: DeleteWorkspaceUserRoleRepositories{WorkspaceUserRole: repo}}
	got := uc.resolveBindingID(context.Background(), &workspaceuserrolepb.DeleteWorkspaceUserRoleRequest{
		Data: &workspaceuserrolepb.WorkspaceUserRole{Id: "wur-1"},
	})
	if got != "wu-read" {
		t.Fatalf("expected resolved wu-read, got %q", got)
	}
	if repo.readCalls != 1 {
		t.Fatalf("expected exactly one read, got %d", repo.readCalls)
	}
}

// TestDeleteInvalidateBindingCache verifies the delete eviction helper is scoped
// and nil-safe.
func TestDeleteInvalidateBindingCache(t *testing.T) {
	rec := &recorder{}
	uc := &DeleteWorkspaceUserRoleUseCase{services: DeleteWorkspaceUserRoleServices{PermissionCacheInvalidator: rec}}
	uc.invalidateBindingCache("wu-9")
	uc.invalidateBindingCache("")
	if len(rec.bindings) != 1 || rec.bindings[0] != "wu-9" {
		t.Fatalf("expected only wu-9 evicted, got %v", rec.bindings)
	}
}
