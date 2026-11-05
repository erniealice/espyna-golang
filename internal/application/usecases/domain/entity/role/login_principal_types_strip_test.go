package role

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	principaltypepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/principal_type"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
)

// captureRoleRepo records what the use case hands the repository.
type captureRoleRepo struct {
	rolepb.UnimplementedRoleDomainServiceServer
	created *rolepb.Role
	updated *rolepb.Role
}

func (c *captureRoleRepo) CreateRole(_ context.Context, req *rolepb.CreateRoleRequest) (*rolepb.CreateRoleResponse, error) {
	c.created = req.Data
	return &rolepb.CreateRoleResponse{Data: []*rolepb.Role{req.Data}, Success: true}, nil
}

func (c *captureRoleRepo) UpdateRole(_ context.Context, req *rolepb.UpdateRoleRequest) (*rolepb.UpdateRoleResponse, error) {
	c.updated = req.Data
	return &rolepb.UpdateRoleResponse{Data: []*rolepb.Role{req.Data}, Success: true}, nil
}

// TestRoleUseCases_StripLoginPrincipalTypes: login_principal_types (which
// logins load a role) is written by logged release SQL only. The create and
// update use cases never pass it to the repository (login-scoped roles,
// 2026-11-05, red-team L5).
func TestRoleUseCases_StripLoginPrincipalTypes(t *testing.T) {
	ctx := context.Background()
	scoped := []principaltypepb.PrincipalType{principaltypepb.PrincipalType(7)}
	repo := &captureRoleRepo{}

	create := NewCreateRoleUseCase(CreateRoleRepositories{Role: repo}, CreateRoleServices{
		Translator: ports.NewNoOpTranslator(), IDGenerator: ports.NewNoOpIDGenerator(),
	})
	if _, err := create.executeCore(ctx, &rolepb.CreateRoleRequest{Data: &rolepb.Role{
		Name: "Scoped Probe", Description: "probe", Color: "#000000", Active: true,
		LoginPrincipalTypes: scoped,
	}}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if repo.created == nil || len(repo.created.GetLoginPrincipalTypes()) != 0 {
		t.Fatalf("create passed login_principal_types to the repository: %v", repo.created.GetLoginPrincipalTypes())
	}

	update := NewUpdateRoleUseCase(UpdateRoleRepositories{Role: repo}, UpdateRoleServices{
		Translator: ports.NewNoOpTranslator(),
	})
	if _, err := update.executeCore(ctx, &rolepb.UpdateRoleRequest{Data: &rolepb.Role{
		Id: "role-x", Name: "Scoped Probe", Color: "#000000", LoginPrincipalTypes: scoped,
	}}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if repo.updated == nil || len(repo.updated.GetLoginPrincipalTypes()) != 0 {
		t.Fatalf("update passed login_principal_types to the repository: %v", repo.updated.GetLoginPrincipalTypes())
	}
}
