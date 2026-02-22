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

// DeleteProductVariantImageRepositories groups all repository dependencies
type DeleteProductVariantImageRepositories struct {
	ProductVariantImage productvariantimagepb.ProductVariantImageDomainServiceServer // Primary entity repository
}

// DeleteProductVariantImageServices groups all business service dependencies
type DeleteProductVariantImageServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// DeleteProductVariantImageUseCase handles the business logic for deleting product variant images
type DeleteProductVariantImageUseCase struct {
	repositories DeleteProductVariantImageRepositories
	services     DeleteProductVariantImageServices
}

// NewDeleteProductVariantImageUseCase creates a new DeleteProductVariantImageUseCase
func NewDeleteProductVariantImageUseCase(
	repositories DeleteProductVariantImageRepositories,
	services DeleteProductVariantImageServices,
) *DeleteProductVariantImageUseCase {
	return &DeleteProductVariantImageUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete product variant image operation
func (uc *DeleteProductVariantImageUseCase) Execute(ctx context.Context, req *productvariantimagepb.DeleteProductVariantImageRequest) (*productvariantimagepb.DeleteProductVariantImageResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductVariantImage, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product variant image deletion within a transaction
func (uc *DeleteProductVariantImageUseCase) executeWithTransaction(ctx context.Context, req *productvariantimagepb.DeleteProductVariantImageRequest) (*productvariantimagepb.DeleteProductVariantImageResponse, error) {
	var result *productvariantimagepb.DeleteProductVariantImageResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return result, nil
}

// executeCore contains the core business logic for deleting a product variant image
func (uc *DeleteProductVariantImageUseCase) executeCore(ctx context.Context, req *productvariantimagepb.DeleteProductVariantImageRequest) (*productvariantimagepb.DeleteProductVariantImageResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.errors.authorization_failed", "Authorization failed for product variant images [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductVariantImage, ports.ActionDelete)
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
	resp, err := uc.repositories.ProductVariantImage.DeleteProductVariantImage(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.errors.deletion_failed", "Product variant image deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteProductVariantImageUseCase) validateInput(ctx context.Context, req *productvariantimagepb.DeleteProductVariantImageRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.validation.data_required", "Product variant image data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.validation.id_required", "Product variant image ID is required [DEFAULT]"))
	}
	return nil
}
