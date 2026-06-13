package asset

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
)

// ReadAssetRepositories groups all repository dependencies
type ReadAssetRepositories struct {
	Asset assetpb.AssetDomainServiceServer // Primary entity repository
}

// ReadAssetServices groups all business service dependencies
type ReadAssetServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadAssetUseCase handles the business logic for reading assets
type ReadAssetUseCase struct {
	repositories ReadAssetRepositories
	services     ReadAssetServices
}

// NewReadAssetUseCase creates use case with grouped dependencies
func NewReadAssetUseCase(
	repositories ReadAssetRepositories,
	services ReadAssetServices,
) *ReadAssetUseCase {
	return &ReadAssetUseCase{
		repositories: repositories,
		services:     services,
	}
}

func (uc *ReadAssetUseCase) Execute(ctx context.Context, req *assetpb.ReadAssetRequest) (*assetpb.ReadAssetResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityAsset,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.Asset.ReadAsset(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return response as-is (even if empty data for not found case)
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadAssetUseCase) validateInput(ctx context.Context, req *assetpb.ReadAssetRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "asset.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "asset.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "asset.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}
	return nil
}
