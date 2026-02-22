package product_variant_option

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	productvariantoptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant_option"
)

// ReadProductVariantOptionRepositories groups all repository dependencies
type ReadProductVariantOptionRepositories struct {
	ProductVariantOption productvariantoptionpb.ProductVariantOptionDomainServiceServer // Primary entity repository
}

// ReadProductVariantOptionServices groups all business service dependencies
type ReadProductVariantOptionServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Transaction management
	TranslationService   ports.TranslationService
}

// ReadProductVariantOptionUseCase handles the business logic for reading a product variant option
type ReadProductVariantOptionUseCase struct {
	repositories ReadProductVariantOptionRepositories
	services     ReadProductVariantOptionServices
}

// NewReadProductVariantOptionUseCase creates use case with grouped dependencies
func NewReadProductVariantOptionUseCase(
	repositories ReadProductVariantOptionRepositories,
	services ReadProductVariantOptionServices,
) *ReadProductVariantOptionUseCase {
	return &ReadProductVariantOptionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read product variant option operation
func (uc *ReadProductVariantOptionUseCase) Execute(ctx context.Context, req *productvariantoptionpb.ReadProductVariantOptionRequest) (*productvariantoptionpb.ReadProductVariantOptionResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductVariantOption, ports.ActionRead); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_option.errors.authorization_failed", "Authorization failed for product variant options [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductVariantOption, ports.ActionRead)
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
	resp, err := uc.repositories.ProductVariantOption.ReadProductVariantOption(ctx, req)
	if err != nil {
		// Handle not found errors by checking for specific patterns in error message
		errorMsg := strings.ToLower(err.Error())
		if strings.Contains(errorMsg, "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_option.errors.not_found", "Product variant option not found [DEFAULT]")
			return nil, errors.New(translatedError)
		}

		// Handle other repository errors
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_option.errors.read_failed", "Failed to retrieve product variant option [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadProductVariantOptionUseCase) validateInput(ctx context.Context, req *productvariantoptionpb.ReadProductVariantOptionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_option.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_option.validation.data_required", "Product variant option data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant_option.validation.id_required", "Product variant option ID is required [DEFAULT]"))
	}
	return nil
}
