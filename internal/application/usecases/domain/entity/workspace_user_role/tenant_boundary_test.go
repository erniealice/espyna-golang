package workspace_user_role

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/shared/tenantguard"
	"github.com/erniealice/espyna-golang/shared/identity"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

// Tenant-boundary tests (plan 20260927-tenant-boundary-hardening Wave 0, U-03).

type tbAllowAuthorizer struct{}

func (tbAllowAuthorizer) HasPermission(context.Context, string, string) (bool, error) {
	return true, nil
}
func (tbAllowAuthorizer) IsEnabled() bool { return true }

type tbWURRepo struct {
	workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer
	rows    map[string]*workspaceuserrolepb.WorkspaceUserRole
	deletes int
}

func (r *tbWURRepo) ReadWorkspaceUserRole(_ context.Context, req *workspaceuserrolepb.ReadWorkspaceUserRoleRequest) (*workspaceuserrolepb.ReadWorkspaceUserRoleResponse, error) {
	if row, ok := r.rows[req.GetData().GetId()]; ok {
		return &workspaceuserrolepb.ReadWorkspaceUserRoleResponse{Data: []*workspaceuserrolepb.WorkspaceUserRole{row}}, nil
	}
	return &workspaceuserrolepb.ReadWorkspaceUserRoleResponse{}, nil
}

func (r *tbWURRepo) DeleteWorkspaceUserRole(context.Context, *workspaceuserrolepb.DeleteWorkspaceUserRoleRequest) (*workspaceuserrolepb.DeleteWorkspaceUserRoleResponse, error) {
	r.deletes++
	return &workspaceuserrolepb.DeleteWorkspaceUserRoleResponse{Success: true}, nil
}

type tbWURMembers struct {
	workspaceuserpb.WorkspaceUserDomainServiceServer
	rows map[string]*workspaceuserpb.WorkspaceUser
}

func (m tbWURMembers) ReadWorkspaceUser(_ context.Context, req *workspaceuserpb.ReadWorkspaceUserRequest) (*workspaceuserpb.ReadWorkspaceUserResponse, error) {
	if row, ok := m.rows[req.GetData().GetId()]; ok {
		return &workspaceuserpb.ReadWorkspaceUserResponse{Data: []*workspaceuserpb.WorkspaceUser{row}}, nil
	}
	return &workspaceuserpb.ReadWorkspaceUserResponse{}, nil
}

func TestDeleteWorkspaceUserRole_TenantBoundary(t *testing.T) {
	repo := &tbWURRepo{rows: map[string]*workspaceuserrolepb.WorkspaceUserRole{
		"wur-a": {Id: "wur-a", WorkspaceUserId: "wu-a", RoleId: "role-a"},
		"wur-b": {Id: "wur-b", WorkspaceUserId: "wu-b", RoleId: "role-b"},
	}}
	members := tbWURMembers{rows: map[string]*workspaceuserpb.WorkspaceUser{
		"wu-a": {Id: "wu-a", WorkspaceId: "ws-a", UserId: "u-a", Active: true},
		"wu-b": {Id: "wu-b", WorkspaceId: "ws-b", UserId: "u-b", Active: true},
	}}
	uc := NewDeleteWorkspaceUserRoleUseCase(
		DeleteWorkspaceUserRoleRepositories{WorkspaceUserRole: repo, WorkspaceUser: members},
		DeleteWorkspaceUserRoleServices{ActionGatekeeper: actiongate.NewActionGatekeeper(tbAllowAuthorizer{}, nil)},
	)
	ctxA := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "admin-a", WorkspaceID: "ws-a"})

	denied := []*workspaceuserrolepb.WorkspaceUserRole{
		{Id: "wur-b"},                          // another tenant's binding
		{Id: "wur-a", WorkspaceUserId: "wu-b"}, // parent on the request disagrees with the stored parent
		{Id: "wur-missing"},                    // unknown id
	}
	for _, d := range denied {
		if _, err := uc.Execute(ctxA, &workspaceuserrolepb.DeleteWorkspaceUserRoleRequest{Data: d}); !errors.Is(err, tenantguard.ErrOutsideTenant) {
			t.Fatalf("delete %+v: want ErrOutsideTenant, got %v", d, err)
		}
	}
	if repo.deletes != 0 {
		t.Fatalf("denied deletes must not write, got %d", repo.deletes)
	}
	if _, err := uc.Execute(ctxA, &workspaceuserrolepb.DeleteWorkspaceUserRoleRequest{Data: &workspaceuserrolepb.WorkspaceUserRole{Id: "wur-a"}}); err != nil {
		t.Fatalf("own binding delete must succeed: %v", err)
	}
	if repo.deletes != 1 {
		t.Fatalf("want 1 delete, got %d", repo.deletes)
	}
}
