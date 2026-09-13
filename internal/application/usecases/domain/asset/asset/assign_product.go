package asset

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// AssetProductAssigner is the narrow persistence capability for a one-way assignment.
type AssetProductAssigner interface {
	AssignAssetProduct(ctx context.Context, assetID, productID string) error
}

type AssignProductRepositories struct {
	Asset   assetpb.AssetDomainServiceServer
	Product productpb.ProductDomainServiceServer
}

type AssignProductServices struct {
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type AssignProductUseCase struct {
	repositories AssignProductRepositories
	services     AssignProductServices
}

func NewAssignProductUseCase(repositories AssignProductRepositories, services AssignProductServices) *AssignProductUseCase {
	return &AssignProductUseCase{repositories: repositories, services: services}
}

// Execute assigns an unassigned asset to an existing product in the active workspace.
func (uc *AssignProductUseCase) Execute(ctx context.Context, assetID, productID string) error {
	if uc.services.ActionGatekeeper == nil {
		return errors.New(uc.message(ctx, "asset.validation.assignment_unavailable", "Asset assignment unavailable"))
	}
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.Asset, Action: entityid.ActionUpdate}); err != nil {
		return err
	}
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.Asset, Action: entityid.ActionRead}); err != nil {
		return err
	}
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.Product, Action: entityid.ActionRead}); err != nil {
		return err
	}
	assetID, productID = strings.TrimSpace(assetID), strings.TrimSpace(productID)
	if assetID == "" || productID == "" {
		return errors.New(uc.message(ctx, "asset.validation.product_required", "Asset and product are required"))
	}
	id, err := identity.RequireWorkspace(ctx)
	if err != nil || id.WorkspaceID == "" || id.PrincipalType == 0 || id.PrincipalID == "" {
		return errors.New(uc.message(ctx, "asset.validation.assignment_unavailable", "Asset assignment unavailable"))
	}
	if uc.repositories.Asset == nil || uc.repositories.Product == nil {
		return errors.New(uc.message(ctx, "asset.validation.assignment_unavailable", "Asset assignment unavailable"))
	}
	a, err := uc.repositories.Asset.ReadAsset(ctx, &assetpb.ReadAssetRequest{Data: &assetpb.Asset{Id: assetID}})
	if err != nil || a == nil || len(a.Data) == 0 {
		return errors.New(uc.message(ctx, "asset.validation.assignment_unavailable", "Asset assignment unavailable"))
	}
	asset := a.Data[0]
	if asset.GetWorkspaceId() != id.WorkspaceID {
		return errors.New(uc.message(ctx, "asset.validation.assignment_unavailable", "Asset assignment unavailable"))
	}
	if asset.GetProductId() != "" {
		return errors.New(uc.message(ctx, "asset.validation.product_already_assigned", "Asset already has a product"))
	}
	p, err := uc.repositories.Product.ReadProduct(ctx, &productpb.ReadProductRequest{Data: &productpb.Product{Id: productID}})
	if err != nil || p == nil || len(p.Data) == 0 || p.Data[0].GetWorkspaceId() != id.WorkspaceID {
		return errors.New(uc.message(ctx, "asset.validation.product_unavailable", "Product unavailable"))
	}
	assigner, ok := uc.repositories.Asset.(AssetProductAssigner)
	if !ok {
		return errors.New(uc.message(ctx, "asset.validation.assignment_unavailable", "Asset assignment unavailable"))
	}
	if err := assigner.AssignAssetProduct(ctx, assetID, productID); err != nil {
		return fmt.Errorf("%s: %w", uc.message(ctx, "asset.validation.assignment_unavailable", "Asset assignment unavailable"), err)
	}
	return nil
}

func (uc *AssignProductUseCase) message(ctx context.Context, key, fallback string) string {
	return contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, key, fallback)
}
