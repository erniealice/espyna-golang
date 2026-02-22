package product_variant

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	productvariantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_variant"
)

// ReadProductVariantRepositories groups all repository dependencies
type ReadProductVariantRepositories struct {
	ProductVariant productvariantpb.ProductVariantDomainServiceServer // Primary entity repository
}

// ReadProductVariantServices groups all business service dependencies
type ReadProductVariantServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Transaction management
	TranslationService   ports.TranslationService
}

// ReadProductVariantUseCase handles the business logic for reading a product variant
type ReadProductVariantUseCase struct {
	repositories ReadProductVariantRepositories
	services     ReadProductVariantServices
}

// NewReadProductVariantUseCase creates use case with grouped dependencies
func NewReadProductVariantUseCase(
	repositories ReadProductVariantRepositories,
	services ReadProductVariantServices,
) *ReadProductVariantUseCase {
	return &ReadProductVariantUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read product variant operation
func (uc *ReadProductVariantUseCase) Execute(ctx context.Context, req *productvariantpb.ReadProductVariantRequest) (*productvariantpb.ReadProductVariantResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductVariant, ports.ActionRead); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant.errors.authorization_failed", "Authorization failed for product variants [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductVariant, ports.ActionRead)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant.errors.authorization_failed", "Authorization failed for product variants [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant.errors.authorization_failed", "Authorization failed for product variants [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductVariant.ReadProductVariant(ctx, req)
	if err != nil {
		// Handle not found errors by checking for specific patterns in error message
		errorMsg := strings.ToLower(err.Error())
		if strings.Contains(errorMsg, "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant.errors.not_found", "Product variant not found [DEFAULT]")
			return nil, errors.New(translatedError)
		}

		// Handle other repository errors
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant.errors.read_failed", "Failed to retrieve product variant [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadProductVariantUseCase) validateInput(ctx context.Context, req *productvariantpb.ReadProductVariantRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant.validation.data_required", "Product variant data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_variant.validation.id_required", "Product variant ID is required [DEFAULT]"))
	}
	return nil
}
