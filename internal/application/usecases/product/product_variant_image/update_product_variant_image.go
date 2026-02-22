package product_variant_image

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	productvariantimagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant_image"
)

// UpdateProductVariantImageRepositories groups all repository dependencies
type UpdateProductVariantImageRepositories struct {
	ProductVariantImage productvariantimagepb.ProductVariantImageDomainServiceServer // Primary entity repository
}

// UpdateProductVariantImageServices groups all business service dependencies
type UpdateProductVariantImageServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateProductVariantImageUseCase handles the business logic for updating product variant images
type UpdateProductVariantImageUseCase struct {
	repositories UpdateProductVariantImageRepositories
	services     UpdateProductVariantImageServices
}

// NewUpdateProductVariantImageUseCase creates use case with grouped dependencies
func NewUpdateProductVariantImageUseCase(
	repositories UpdateProductVariantImageRepositories,
	services UpdateProductVariantImageServices,
) *UpdateProductVariantImageUseCase {
	return &UpdateProductVariantImageUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update product variant image operation
func (uc *UpdateProductVariantImageUseCase) Execute(ctx context.Context, req *productvariantimagepb.UpdateProductVariantImageRequest) (*productvariantimagepb.UpdateProductVariantImageResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductVariantImage, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product variant image update within a transaction
func (uc *UpdateProductVariantImageUseCase) executeWithTransaction(ctx context.Context, req *productvariantimagepb.UpdateProductVariantImageRequest) (*productvariantimagepb.UpdateProductVariantImageResponse, error) {
	var result *productvariantimagepb.UpdateProductVariantImageResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "product_variant_image.errors.update_failed", "Product variant image update failed [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic
func (uc *UpdateProductVariantImageUseCase) executeCore(ctx context.Context, req *productvariantimagepb.UpdateProductVariantImageRequest) (*productvariantimagepb.UpdateProductVariantImageResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.errors.authorization_failed", "Authorization failed for product variant images [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductVariantImage, ports.ActionUpdate)
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

	// Business logic and enrichment
	if err := uc.enrichData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductVariantImage.UpdateProductVariantImage(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.errors.update_failed", "Product variant image update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateProductVariantImageUseCase) validateInput(ctx context.Context, req *productvariantimagepb.UpdateProductVariantImageRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.validation.data_required", "Product variant image data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.validation.id_required", "Product variant image ID is required [DEFAULT]"))
	}
	if req.Data.ImageUrl == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.validation.image_url_required", "Image URL is required [DEFAULT]"))
	}
	return nil
}

// enrichData adds generated fields and audit information
func (uc *UpdateProductVariantImageUseCase) enrichData(data *productvariantimagepb.ProductVariantImage) error {
	now := time.Now()

	// Update audit fields
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}
