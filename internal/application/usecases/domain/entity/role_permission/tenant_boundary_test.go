package role_permission

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/shared/tenantguard"
	"github.com/erniealice/espyna-golang/shared/identity"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
	rolepermissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role_permission"
)

// Tenant-boundary tests (plan 20260927-tenant-boundary-hardening Wave 0, U-03).

type tbAllowAuthorizer struct{}

func (tbAllowAuthorizer) HasPermission(context.Context, string, string) (bool, error) {
	return true, nil
}
func (tbAllowAuthorizer) IsEnabled() bool { return true }

type tbRPRepo struct {
	rolepermissionpb.RolePermissionDomainServiceServer
	rows    map[string]*rolepermissionpb.RolePermission
	deletes int
}

func (r *tbRPRepo) ReadRolePermission(_ context.Context, req *rolepermissionpb.ReadRolePermissionRequest) (*rolepermissionpb.ReadRolePermissionResponse, error) {
	if row, ok := r.rows[req.GetData().GetId()]; ok {
		return &rolepermissionpb.ReadRolePermissionResponse{Data: []*rolepermissionpb.RolePermission{row}}, nil
	}
	return &rolepermissionpb.ReadRolePermissionResponse{}, nil
}

func (r *tbRPRepo) DeleteRolePermission(context.Context, *rolepermissionpb.DeleteRolePermissionRequest) (*rolepermissionpb.DeleteRolePermissionResponse, error) {
	r.deletes++
	return &rolepermissionpb.DeleteRolePermissionResponse{Success: true}, nil
}

type tbRoles struct {
	rolepb.RoleDomainServiceServer
	rows map[string]*rolepb.Role
}

func (r tbRoles) ReadRole(_ context.Context, req *rolepb.ReadRoleRequest) (*rolepb.ReadRoleResponse, error) {
	if row, ok := r.rows[req.GetData().GetId()]; ok {
		return &rolepb.ReadRoleResponse{Data: []*rolepb.Role{row}}, nil
	}
	return &rolepb.ReadRoleResponse{}, nil
}

func TestDeleteRolePermission_TenantBoundary(t *testing.T) {
	wsA, wsB := "ws-a", "ws-b"
	repo := &tbRPRepo{rows: map[string]*rolepermissionpb.RolePermission{
		"rp-a": {Id: "rp-a", RoleId: "role-a", PermissionId: "perm-x"},
		"rp-b": {Id: "rp-b", RoleId: "role-b", PermissionId: "perm-x"},
	}}
	roles := tbRoles{rows: map[string]*rolepb.Role{
		"role-a": {Id: "role-a", WorkspaceId: &wsA},
		"role-b": {Id: "role-b", WorkspaceId: &wsB},
	}}
	uc := NewDeleteRolePermissionUseCase(
		DeleteRolePermissionRepositories{RolePermission: repo, Role: roles},
		DeleteRolePermissionServices{ActionGatekeeper: actiongate.NewActionGatekeeper(tbAllowAuthorizer{}, nil)},
	)
	ctxA := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "admin-a", WorkspaceID: "ws-a"})

	denied := []*rolepermissionpb.RolePermission{
		{Id: "rp-b"},                   // grant on another tenant's role
		{Id: "rp-a", RoleId: "role-b"}, // URL parent disagrees with the stored parent
		{Id: "rp-missing"},             // unknown id
	}
	for _, d := range denied {
		if _, err := uc.Execute(ctxA, &rolepermissionpb.DeleteRolePermissionRequest{Data: d}); !errors.Is(err, tenantguard.ErrOutsideTenant) {
			t.Fatalf("delete %+v: want ErrOutsideTenant, got %v", d, err)
		}
	}
	if repo.deletes != 0 {
		t.Fatalf("denied deletes must not write, got %d", repo.deletes)
	}
	if _, err := uc.Execute(ctxA, &rolepermissionpb.DeleteRolePermissionRequest{Data: &rolepermissionpb.RolePermission{Id: "rp-a", RoleId: "role-a"}}); err != nil {
		t.Fatalf("own grant delete must succeed: %v", err)
	}
	if repo.deletes != 1 {
		t.Fatalf("want 1 delete, got %d", repo.deletes)
	}
}
