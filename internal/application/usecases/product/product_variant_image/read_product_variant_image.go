package product_variant_image

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	productvariantimagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant_image"
)

// ReadProductVariantImageRepositories groups all repository dependencies
type ReadProductVariantImageRepositories struct {
	ProductVariantImage productvariantimagepb.ProductVariantImageDomainServiceServer // Primary entity repository
}

// ReadProductVariantImageServices groups all business service dependencies
type ReadProductVariantImageServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Transaction management
	TranslationService   ports.TranslationService
}

// ReadProductVariantImageUseCase handles the business logic for reading a product variant image
type ReadProductVariantImageUseCase struct {
	repositories ReadProductVariantImageRepositories
	services     ReadProductVariantImageServices
}

// NewReadProductVariantImageUseCase creates use case with grouped dependencies
func NewReadProductVariantImageUseCase(
	repositories ReadProductVariantImageRepositories,
	services ReadProductVariantImageServices,
) *ReadProductVariantImageUseCase {
	return &ReadProductVariantImageUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read product variant image operation
func (uc *ReadProductVariantImageUseCase) Execute(ctx context.Context, req *productvariantimagepb.ReadProductVariantImageRequest) (*productvariantimagepb.ReadProductVariantImageResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductVariantImage, ports.ActionRead); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.errors.authorization_failed", "Authorization failed for product variant images [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductVariantImage, ports.ActionRead)
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
	resp, err := uc.repositories.ProductVariantImage.ReadProductVariantImage(ctx, req)
	if err != nil {
		// Handle not found errors by checking for specific patterns in error message
		errorMsg := strings.ToLower(err.Error())
		if strings.Contains(errorMsg, "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.errors.not_found", "Product variant image not found [DEFAULT]")
			return nil, errors.New(translatedError)
		}

		// Handle other repository errors
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_image.errors.read_failed", "Failed to retrieve product variant image [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadProductVariantImageUseCase) validateInput(ctx context.Context, req *productvariantimagepb.ReadProductVariantImageRequest) error {
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
