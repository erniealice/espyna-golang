package asset

import (
	"context"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

type assignAuth struct{ allowed map[string]bool }

func (a assignAuth) HasPermission(_ context.Context, _ string, p string) (bool, error) {
	return a.allowed[p], nil
}
func (a assignAuth) IsEnabled() bool { return true }

type assignAssetRepo struct {
	assetpb.UnimplementedAssetDomainServiceServer
	asset    *assetpb.Asset
	assigned string
}

func (r *assignAssetRepo) ReadAsset(context.Context, *assetpb.ReadAssetRequest) (*assetpb.ReadAssetResponse, error) {
	return &assetpb.ReadAssetResponse{Data: []*assetpb.Asset{r.asset}}, nil
}
func (r *assignAssetRepo) AssignAssetProduct(_ context.Context, _, productID string) error {
	r.assigned = productID
	return nil
}

type assignProductRepo struct {
	productpb.UnimplementedProductDomainServiceServer
	product *productpb.Product
}

func (r *assignProductRepo) ReadProduct(context.Context, *productpb.ReadProductRequest) (*productpb.ReadProductResponse, error) {
	return &productpb.ReadProductResponse{Data: []*productpb.Product{r.product}}, nil
}

func assignCtx(ws string) context.Context {
	return identity.WithRequestIdentity(contextutil.WithUserID(context.Background(), "u1"), &identity.RequestIdentity{UserID: "u1", WorkspaceID: ws, PrincipalType: 7, PrincipalID: "staff1"})
}
func assignUC(a *assignAssetRepo, p *assignProductRepo, allowed map[string]bool, gateNil bool) *AssignProductUseCase {
	var gate *actiongate.ActionGatekeeper
	if !gateNil {
		gate = actiongate.NewActionGatekeeper(assignAuth{allowed: allowed}, ports.NewNoOpTranslator())
	}
	return NewAssignProductUseCase(AssignProductRepositories{Asset: a, Product: p}, AssignProductServices{ActionGatekeeper: gate, Translator: ports.NewNoOpTranslator()})
}
func allAssignPerms() map[string]bool {
	return map[string]bool{entityid.EntityPermission(entityid.Asset, entityid.ActionUpdate): true, entityid.EntityPermission(entityid.Asset, entityid.ActionRead): true, entityid.EntityPermission(entityid.Product, entityid.ActionRead): true}
}

func TestAssignProductExecute_SuccessPassesExactIDs(t *testing.T) {
	a := &assignAssetRepo{asset: &assetpb.Asset{Id: "a1", WorkspaceId: strp("ws1")}}
	p := &assignProductRepo{product: &productpb.Product{Id: "p1", WorkspaceId: strp("ws1")}}
	if err := assignUC(a, p, allAssignPerms(), false).Execute(assignCtx("ws1"), "a1", "p1"); err != nil {
		t.Fatal(err)
	}
	if a.assigned != "p1" {
		t.Fatalf("assigned product = %q", a.assigned)
	}
}
func TestAssignProductExecute_RejectsCases(t *testing.T) {
	base := allAssignPerms()
	cases := []struct {
		name, ws, aws, pws, current string
		perms                       map[string]bool
		nilGate                     bool
		want                        string
	}{
		{"nil gate", "ws1", "ws1", "ws1", "", base, true, "unavailable"},
		{"missing workspace", "", "ws1", "ws1", "", base, false, "unavailable"},
		{"asset mismatch", "ws1", "ws2", "ws1", "", base, false, "unavailable"},
		{"product mismatch", "ws1", "ws1", "ws2", "", base, false, "unavailable"},
		{"already assigned", "ws1", "ws1", "ws1", "other", base, false, "already"},
		{"auth denied", "ws1", "ws1", "ws1", "", map[string]bool{}, false, "denied"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := assignUC(&assignAssetRepo{asset: &assetpb.Asset{Id: "a1", WorkspaceId: strp(tc.aws), ProductId: strptr(tc.current)}}, &assignProductRepo{product: &productpb.Product{Id: "p1", WorkspaceId: strp(tc.pws)}}, tc.perms, tc.nilGate).Execute(assignCtx(tc.ws), "a1", "p1")
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("error=%v, want %q", err, tc.want)
			}
		})
	}
}
func strp(s string) *string { return &s }
func strptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func TestAssignProductRejectsMissingTypedBinding(t *testing.T) {
	a := &assignAssetRepo{asset: &assetpb.Asset{Id: "a1", WorkspaceId: strp("ws1")}}
	p := &assignProductRepo{product: &productpb.Product{Id: "p1", WorkspaceId: strp("ws1")}}
	ctx := identity.WithRequestIdentity(contextutil.WithUserID(context.Background(), "u1"), &identity.RequestIdentity{UserID: "u1", WorkspaceID: "ws1"})
	if err := assignUC(a, p, allAssignPerms(), false).Execute(ctx, "a1", "p1"); err == nil || a.assigned != "" {
		t.Fatal("missing active principal allowed assignment")
	}
}

func TestCreateAssetProductRequiresReadPermissionAndScope(t *testing.T) {
	for _, tc := range []struct {
		name    string
		allowed bool
		ws      string
	}{{"permission", false, "ws1"}, {"scope", true, "other"}} {
		t.Run(tc.name, func(t *testing.T) {
			gate := actiongate.NewActionGatekeeper(assignAuth{allowed: map[string]bool{"asset:create": true, "product:read": tc.allowed}}, ports.NewNoOpTranslator())
			uc := NewCreateAssetUseCase(CreateAssetRepositories{Product: &assignProductRepo{product: &productpb.Product{Id: "p1", WorkspaceId: strp(tc.ws)}}}, CreateAssetServices{ActionGatekeeper: gate, Translator: ports.NewNoOpTranslator()})
			err := uc.validateInput(assignCtx("ws1"), &assetpb.CreateAssetRequest{Data: &assetpb.Asset{Name: "Unit", AcquisitionCost: 100, AssetCategoryId: "category", ProductId: strp("p1")}})
			if err == nil {
				t.Fatal("unauthorized product link accepted")
			}
		})
	}
}

func TestUpdateAssetProductAcknowledgementScope(t *testing.T) {
	for _, tc := range []struct {
		name, workspace, current, requested string
		binding                             bool
		allowRead                           bool
		wantOK                              bool
	}{
		{"same link", "ws1", "p1", "p1", true, true, true},
		{"change link", "ws1", "p1", "p2", true, true, false},
		{"cross workspace", "other", "p1", "p1", true, true, false},
		{"no binding", "ws1", "p1", "p1", false, true, false},
		{"no read permission", "ws1", "p1", "p1", true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := assignCtx("ws1")
			if !tc.binding {
				ctx = identity.WithRequestIdentity(contextutil.WithUserID(context.Background(), "u1"), &identity.RequestIdentity{UserID: "u1", WorkspaceID: "ws1"})
			}
			gate := actiongate.NewActionGatekeeper(assignAuth{allowed: map[string]bool{"asset:read": tc.allowRead}}, ports.NewNoOpTranslator())
			uc := NewUpdateAssetUseCase(UpdateAssetRepositories{Asset: &assignAssetRepo{asset: &assetpb.Asset{Id: "a1", WorkspaceId: strp(tc.workspace), ProductId: strp(tc.current)}}}, UpdateAssetServices{ActionGatekeeper: gate, Translator: ports.NewNoOpTranslator()})
			err := uc.validateInput(ctx, &assetpb.UpdateAssetRequest{Data: &assetpb.Asset{Id: "a1", Name: "Unit", AcquisitionCost: 100, AssetCategoryId: "category", ProductId: strp(tc.requested)}})
			if (err == nil) != tc.wantOK {
				t.Fatalf("error=%v wantOK=%v", err, tc.wantOK)
			}
		})
	}
}
