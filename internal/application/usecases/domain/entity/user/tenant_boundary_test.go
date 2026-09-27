package user

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/shared/tenantguard"
	"github.com/erniealice/espyna-golang/shared/identity"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
)

// Tenant-boundary tests (plan 20260927-tenant-boundary-hardening Wave 0, U-01).
// user has no workspace_id: RBAC alone must never let a workspace admin act on
// a user outside the actor's workspace.

type tbAllowAuthorizer struct{}

func (tbAllowAuthorizer) HasPermission(context.Context, string, string) (bool, error) {
	return true, nil
}
func (tbAllowAuthorizer) IsEnabled() bool { return true }

type tbUserRepo struct {
	userpb.UserDomainServiceServer
	users   map[string]*userpb.User
	updates int
	deletes int
}

func (r *tbUserRepo) ReadUser(_ context.Context, req *userpb.ReadUserRequest) (*userpb.ReadUserResponse, error) {
	if u, ok := r.users[req.GetData().GetId()]; ok {
		return &userpb.ReadUserResponse{Data: []*userpb.User{u}, Success: true}, nil
	}
	return &userpb.ReadUserResponse{}, nil
}

func (r *tbUserRepo) UpdateUser(_ context.Context, req *userpb.UpdateUserRequest) (*userpb.UpdateUserResponse, error) {
	r.updates++
	return &userpb.UpdateUserResponse{Data: []*userpb.User{req.GetData()}, Success: true}, nil
}

func (r *tbUserRepo) DeleteUser(context.Context, *userpb.DeleteUserRequest) (*userpb.DeleteUserResponse, error) {
	r.deletes++
	return &userpb.DeleteUserResponse{Success: true}, nil
}

func (r *tbUserRepo) GetUserItemPageData(_ context.Context, req *userpb.GetUserItemPageDataRequest) (*userpb.GetUserItemPageDataResponse, error) {
	return &userpb.GetUserItemPageDataResponse{User: r.users[req.GetUserId()], Success: true}, nil
}

type tbMembers struct {
	workspaceuserpb.WorkspaceUserDomainServiceServer
	rows []*workspaceuserpb.WorkspaceUser
}

func (m tbMembers) ListWorkspaceUsers(context.Context, *workspaceuserpb.ListWorkspaceUsersRequest) (*workspaceuserpb.ListWorkspaceUsersResponse, error) {
	return &workspaceuserpb.ListWorkspaceUsersResponse{Data: m.rows}, nil // ignores filters on purpose
}

func tbFixture() (*tbUserRepo, tbMembers) {
	repo := &tbUserRepo{users: map[string]*userpb.User{
		"u-a": {Id: "u-a", EmailAddress: "a@example.test", FirstName: "A", Active: true},
		"u-b": {Id: "u-b", EmailAddress: "b@example.test", FirstName: "B", Active: true},
	}}
	members := tbMembers{rows: []*workspaceuserpb.WorkspaceUser{
		{Id: "wu-a", WorkspaceId: "ws-a", UserId: "u-a", Active: true},
		{Id: "wu-b", WorkspaceId: "ws-b", UserId: "u-b", Active: true},
	}}
	return repo, members
}

func tbCtx(ws string) context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "admin-a", WorkspaceID: ws})
}

func tbGate() *actiongate.ActionGatekeeper {
	return actiongate.NewActionGatekeeper(tbAllowAuthorizer{}, nil)
}

func TestUpdateUser_TenantBoundary(t *testing.T) {
	repo, members := tbFixture()
	uc := NewUpdateUserUseCase(UpdateUserRepositories{User: repo, WorkspaceUser: members}, UpdateUserServices{ActionGatekeeper: tbGate()})

	// Foreign user (member of ws-b only): denied, zero writes.
	_, err := uc.Execute(tbCtx("ws-a"), &userpb.UpdateUserRequest{Data: &userpb.User{Id: "u-b", EmailAddress: "b@example.test", FirstName: "Renamed", Active: true}})
	if !errors.Is(err, tenantguard.ErrOutsideTenant) {
		t.Fatalf("foreign user update: want ErrOutsideTenant, got %v", err)
	}
	// Own member, email change: control-plane only.
	_, err = uc.Execute(tbCtx("ws-a"), &userpb.UpdateUserRequest{Data: &userpb.User{Id: "u-a", EmailAddress: "attacker@example.test", FirstName: "A", Active: true}})
	if !errors.Is(err, tenantguard.ErrPlatformOperation) {
		t.Fatalf("email change: want ErrPlatformOperation, got %v", err)
	}
	// Own member, global deactivate via edit form: control-plane only.
	_, err = uc.Execute(tbCtx("ws-a"), &userpb.UpdateUserRequest{Data: &userpb.User{Id: "u-a", EmailAddress: "a@example.test", FirstName: "A", Active: false}})
	if !errors.Is(err, tenantguard.ErrPlatformOperation) {
		t.Fatalf("active change: want ErrPlatformOperation, got %v", err)
	}
	// No workspace selected: fail closed.
	if _, err = uc.Execute(tbCtx(""), &userpb.UpdateUserRequest{Data: &userpb.User{Id: "u-a", EmailAddress: "a@example.test", FirstName: "A", Active: true}}); err == nil {
		t.Fatal("no workspace selected must deny")
	}
	if repo.updates != 0 {
		t.Fatalf("denied updates must not write; got %d writes", repo.updates)
	}
	// Own member, profile-only edit: allowed.
	if _, err = uc.Execute(tbCtx("ws-a"), &userpb.UpdateUserRequest{Data: &userpb.User{Id: "u-a", EmailAddress: "a@example.test", FirstName: "Alice", Active: true}}); err != nil {
		t.Fatalf("own member name edit must succeed: %v", err)
	}
	if repo.updates != 1 {
		t.Fatalf("want 1 write for the allowed edit, got %d", repo.updates)
	}
}

func TestGetUserItemPageData_ForeignUserDenied(t *testing.T) {
	repo, members := tbFixture()
	uc := NewGetUserItemPageDataUseCase(GetUserItemPageDataRepositories{User: repo, WorkspaceUser: members}, GetUserItemPageDataServices{ActionGatekeeper: tbGate()})
	if _, err := uc.Execute(tbCtx("ws-a"), &userpb.GetUserItemPageDataRequest{UserId: "u-b"}); !errors.Is(err, tenantguard.ErrOutsideTenant) {
		t.Fatalf("foreign user detail: want ErrOutsideTenant, got %v", err)
	}
	if _, err := uc.Execute(tbCtx("ws-a"), &userpb.GetUserItemPageDataRequest{UserId: "u-a"}); err != nil {
		t.Fatalf("own member detail must succeed: %v", err)
	}
}

func TestGlobalUserLifecycle_DeniedFromTenantPath(t *testing.T) {
	repo, _ := tbFixture()
	del := NewDeleteUserUseCase(DeleteUserRepositories{User: repo}, DeleteUserServices{ActionGatekeeper: tbGate()})
	if _, err := del.Execute(tbCtx("ws-a"), &userpb.DeleteUserRequest{Data: &userpb.User{Id: "u-a"}}); !errors.Is(err, tenantguard.ErrPlatformOperation) {
		t.Fatalf("delete user: want ErrPlatformOperation, got %v", err)
	}
	dis := NewDisableUserUseCase(DisableUserRepositories{User: repo}, DisableUserServices{ActionGatekeeper: tbGate()})
	if _, err := dis.Execute(tbCtx("ws-a"), &userpb.DisableUserRequest{UserId: "u-b"}); !errors.Is(err, tenantguard.ErrPlatformOperation) {
		t.Fatalf("disable user: want ErrPlatformOperation, got %v", err)
	}
	en := NewEnableUserUseCase(EnableUserRepositories{User: repo}, EnableUserServices{ActionGatekeeper: tbGate()})
	if _, err := en.Execute(tbCtx("ws-a"), &userpb.EnableUserRequest{UserId: "u-b"}); !errors.Is(err, tenantguard.ErrPlatformOperation) {
		t.Fatalf("enable user: want ErrPlatformOperation, got %v", err)
	}
	if repo.deletes != 0 || repo.updates != 0 {
		t.Fatalf("denied lifecycle ops must not write (deletes=%d updates=%d)", repo.deletes, repo.updates)
	}
}

func (r *tbUserRepo) tbReadUseCase(members tbMembers) *ReadUserUseCase {
	return NewReadUserUseCase(ReadUserRepositories{User: r, WorkspaceUser: members}, ReadUserServices{ActionGatekeeper: tbGate()})
}

func TestReadUser_TenantBoundary(t *testing.T) {
	repo, members := tbFixture()
	uc := repo.tbReadUseCase(members)
	if _, err := uc.Execute(tbCtx("ws-a"), &userpb.ReadUserRequest{Data: &userpb.User{Id: "u-b"}}); !errors.Is(err, tenantguard.ErrOutsideTenant) {
		t.Fatalf("foreign read: want ErrOutsideTenant, got %v", err)
	}
	if _, err := uc.Execute(tbCtx("ws-a"), &userpb.ReadUserRequest{Data: &userpb.User{Id: "u-a"}}); err != nil {
		t.Fatalf("member read must succeed: %v", err)
	}
	// Self read (sidebar profile, timezone lookup) works before a workspace is chosen.
	selfCtx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "u-b"})
	if _, err := uc.Execute(selfCtx, &userpb.ReadUserRequest{Data: &userpb.User{Id: "u-b"}}); err != nil {
		t.Fatalf("self read must succeed: %v", err)
	}
}
