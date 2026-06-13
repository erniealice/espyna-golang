package asset

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
)

// SetAssetActiveRepositories groups all repository dependencies
type SetAssetActiveRepositories struct {
	Asset assetpb.AssetDomainServiceServer // Primary entity repository
}

// SetAssetActiveServices groups all business service dependencies
type SetAssetActiveServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// SetAssetActiveUseCase handles toggling the active flag on an asset.
// It mirrors the UpdateAsset use case shape but operates on a minimal
// partial-field request (asset_id + active) to avoid the proto3 zero-bool
// ambiguity that would occur if UpdateAsset were used to set active=false.
// See docs/plan/20260503-asset-typed-stack-buildout/plan.md §"Why SetAssetActive".
type SetAssetActiveUseCase struct {
	repositories SetAssetActiveRepositories
	services     SetAssetActiveServices
}

// NewSetAssetActiveUseCase creates use case with grouped dependencies
func NewSetAssetActiveUseCase(
	repositories SetAssetActiveRepositories,
	services SetAssetActiveServices,
) *SetAssetActiveUseCase {
	return &SetAssetActiveUseCase{
		repositories: repositories,
		services:     services,
	}
}

func (uc *SetAssetActiveUseCase) Execute(ctx context.Context, req *assetpb.SetAssetActiveRequest) (*assetpb.SetAssetActiveResponse, error) {
	// Authorization check — toggling active is semantically an Update action.
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityAsset,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "asset.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository — read-merge-update is handled inside the adapter
	// (see SetAssetActive in contrib/postgres/internal/adapter/asset/asset.go).
	resp, err := uc.repositories.Asset.SetAssetActive(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "asset.errors.set_active_failed", "[ERR-DEFAULT] Asset set active failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request for SetAssetActive.
func (uc *SetAssetActiveUseCase) validateInput(ctx context.Context, req *assetpb.SetAssetActiveRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "asset.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.AssetId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "asset.validation.id_required", "[ERR-DEFAULT] Asset ID is required"))
	}
	return nil
}
