package asset

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
)

// ListAssetsRepositories groups all repository dependencies
type ListAssetsRepositories struct {
	Asset assetpb.AssetDomainServiceServer // Primary entity repository
}

// ListAssetsServices groups all business service dependencies
type ListAssetsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListAssetsUseCase handles the business logic for listing assets
type ListAssetsUseCase struct {
	repositories ListAssetsRepositories
	services     ListAssetsServices
}

// NewListAssetsUseCase creates use case with grouped dependencies
func NewListAssetsUseCase(
	repositories ListAssetsRepositories,
	services ListAssetsServices,
) *ListAssetsUseCase {
	return &ListAssetsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list assets operation
func (uc *ListAssetsUseCase) Execute(ctx context.Context, req *assetpb.ListAssetsRequest) (*assetpb.ListAssetsResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAsset, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Asset.ListAssets(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.errors.list_failed", "[ERR-DEFAULT] Failed to list assets")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListAssetsUseCase) validateInput(ctx context.Context, req *assetpb.ListAssetsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "asset.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for listing
func (uc *ListAssetsUseCase) validateBusinessRules(ctx context.Context, req *assetpb.ListAssetsRequest) error {
	// No additional business rules for listing assets
	return nil
}
