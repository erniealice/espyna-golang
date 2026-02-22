package product_variant_image

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	productvariantimagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant_image"
)

// ListProductVariantImagesRepositories groups all repository dependencies
type ListProductVariantImagesRepositories struct {
	ProductVariantImage productvariantimagepb.ProductVariantImageDomainServiceServer // Primary entity repository
}

// ListProductVariantImagesServices groups all business service dependencies
type ListProductVariantImagesServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListProductVariantImagesUseCase handles the business logic for listing product variant images
type ListProductVariantImagesUseCase struct {
	repositories ListProductVariantImagesRepositories
	services     ListProductVariantImagesServices
}

// NewListProductVariantImagesUseCase creates a new ListProductVariantImagesUseCase
func NewListProductVariantImagesUseCase(
	repositories ListProductVariantImagesRepositories,
	services ListProductVariantImagesServices,
) *ListProductVariantImagesUseCase {
	return &ListProductVariantImagesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list product variant images operation
func (uc *ListProductVariantImagesUseCase) Execute(ctx context.Context, req *productvariantimagepb.ListProductVariantImagesRequest) (*productvariantimagepb.ListProductVariantImagesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductVariantImage, ports.ActionList); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.errors.authorization_failed", "Authorization failed for product variant images [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductVariantImage, ports.ActionList)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.errors.authorization_failed", "Authorization failed for product variant images [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.errors.authorization_failed", "Authorization failed for product variant images [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductVariantImage.ListProductVariantImages(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.errors.list_failed", "Failed to retrieve product variant images [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListProductVariantImagesUseCase) validateInput(ctx context.Context, req *productvariantimagepb.ListProductVariantImagesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.validation.request_required", "Request is required [DEFAULT]"))
	}
	// Additional validation can be added here if needed
	return nil
}
