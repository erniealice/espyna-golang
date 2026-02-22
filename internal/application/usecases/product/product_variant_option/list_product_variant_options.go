package product_variant_option

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	productvariantoptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant_option"
)

// ListProductVariantOptionsRepositories groups all repository dependencies
type ListProductVariantOptionsRepositories struct {
	ProductVariantOption productvariantoptionpb.ProductVariantOptionDomainServiceServer // Primary entity repository
}

// ListProductVariantOptionsServices groups all business service dependencies
type ListProductVariantOptionsServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListProductVariantOptionsUseCase handles the business logic for listing product variant options
type ListProductVariantOptionsUseCase struct {
	repositories ListProductVariantOptionsRepositories
	services     ListProductVariantOptionsServices
}

// NewListProductVariantOptionsUseCase creates a new ListProductVariantOptionsUseCase
func NewListProductVariantOptionsUseCase(
	repositories ListProductVariantOptionsRepositories,
	services ListProductVariantOptionsServices,
) *ListProductVariantOptionsUseCase {
	return &ListProductVariantOptionsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list product variant options operation
func (uc *ListProductVariantOptionsUseCase) Execute(ctx context.Context, req *productvariantoptionpb.ListProductVariantOptionsRequest) (*productvariantoptionpb.ListProductVariantOptionsResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductVariantOption, ports.ActionList); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_option.errors.authorization_failed", "Authorization failed for product variant options [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductVariantOption, ports.ActionList)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_option.errors.authorization_failed", "Authorization failed for product variant options [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_option.errors.authorization_failed", "Authorization failed for product variant options [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_option.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductVariantOption.ListProductVariantOptions(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_option.errors.list_failed", "Failed to retrieve product variant options [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListProductVariantOptionsUseCase) validateInput(ctx context.Context, req *productvariantoptionpb.ListProductVariantOptionsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_option.validation.request_required", "Request is required [DEFAULT]"))
	}
	// Additional validation can be added here if needed
	return nil
}
