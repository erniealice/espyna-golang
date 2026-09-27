package workspace

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/shared/tenantguard"
	"github.com/erniealice/espyna-golang/shared/identity"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

// Tenant-boundary tests (plan 20260927-tenant-boundary-hardening Wave 0, U-02).

type tbAllowAuthorizer struct{}

func (tbAllowAuthorizer) HasPermission(context.Context, string, string) (bool, error) {
	return true, nil
}
func (tbAllowAuthorizer) IsEnabled() bool { return true }

type tbWorkspaceRepo struct {
	workspacepb.WorkspaceDomainServiceServer
	writes int
}

func (r *tbWorkspaceRepo) UpdateWorkspace(_ context.Context, req *workspacepb.UpdateWorkspaceRequest) (*workspacepb.UpdateWorkspaceResponse, error) {
	r.writes++
	return &workspacepb.UpdateWorkspaceResponse{Data: []*workspacepb.Workspace{req.GetData()}, Success: true}, nil
}

func (r *tbWorkspaceRepo) DeleteWorkspace(context.Context, *workspacepb.DeleteWorkspaceRequest) (*workspacepb.DeleteWorkspaceResponse, error) {
	r.writes++
	return &workspacepb.DeleteWorkspaceResponse{Success: true}, nil
}

func (r *tbWorkspaceRepo) CreateWorkspace(context.Context, *workspacepb.CreateWorkspaceRequest) (*workspacepb.CreateWorkspaceResponse, error) {
	r.writes++
	return &workspacepb.CreateWorkspaceResponse{Success: true}, nil
}

func tbCtx(ws string) context.Context {
	return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "admin-a", WorkspaceID: ws})
}

func tbGate() *actiongate.ActionGatekeeper {
	return actiongate.NewActionGatekeeper(tbAllowAuthorizer{}, nil)
}

func TestUpdateWorkspace_OnlyActorWorkspace(t *testing.T) {
	repo := &tbWorkspaceRepo{}
	uc := NewUpdateWorkspaceUseCase(UpdateWorkspaceRepositories{Workspace: repo}, UpdateWorkspaceServices{ActionGatekeeper: tbGate()})
	if _, err := uc.Execute(tbCtx("ws-a"), &workspacepb.UpdateWorkspaceRequest{Data: &workspacepb.Workspace{Id: "ws-b", Name: "Hijacked"}}); !errors.Is(err, tenantguard.ErrOutsideTenant) {
		t.Fatalf("other workspace: want ErrOutsideTenant, got %v", err)
	}
	if _, err := uc.Execute(tbCtx(""), &workspacepb.UpdateWorkspaceRequest{Data: &workspacepb.Workspace{Id: "ws-a", Name: "Renamed"}}); err == nil {
		t.Fatal("no workspace selected must deny")
	}
	if repo.writes != 0 {
		t.Fatalf("denied updates must not write, got %d", repo.writes)
	}
	if _, err := uc.Execute(tbCtx("ws-a"), &workspacepb.UpdateWorkspaceRequest{Data: &workspacepb.Workspace{Id: "ws-a", Name: "Renamed"}}); err != nil {
		t.Fatalf("own workspace update must succeed: %v", err)
	}
}

func TestWorkspaceCreateDelete_DeniedFromTenantPath(t *testing.T) {
	repo := &tbWorkspaceRepo{}
	del := NewDeleteWorkspaceUseCase(DeleteWorkspaceRepositories{Workspace: repo}, DeleteWorkspaceServices{ActionGatekeeper: tbGate()})
	if _, err := del.Execute(tbCtx("ws-a"), &workspacepb.DeleteWorkspaceRequest{Data: &workspacepb.Workspace{Id: "ws-b"}}); !errors.Is(err, tenantguard.ErrPlatformOperation) {
		t.Fatalf("delete workspace: want ErrPlatformOperation, got %v", err)
	}
	cr := NewCreateWorkspaceUseCase(CreateWorkspaceRepositories{Workspace: repo}, CreateWorkspaceServices{ActionGatekeeper: tbGate()})
	if _, err := cr.Execute(tbCtx("ws-a"), &workspacepb.CreateWorkspaceRequest{Data: &workspacepb.Workspace{Name: "New tenant"}}); !errors.Is(err, tenantguard.ErrPlatformOperation) {
		t.Fatalf("create workspace: want ErrPlatformOperation, got %v", err)
	}
	if repo.writes != 0 {
		t.Fatalf("denied ops must not write, got %d", repo.writes)
	}
}
