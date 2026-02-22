package product_option_value

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	productoptionvaluepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_option_value"
)

// ListProductOptionValuesRepositories groups all repository dependencies
type ListProductOptionValuesRepositories struct {
	ProductOptionValue productoptionvaluepb.ProductOptionValueDomainServiceServer // Primary entity repository
}

// ListProductOptionValuesServices groups all business service dependencies
type ListProductOptionValuesServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListProductOptionValuesUseCase handles the business logic for listing product option values
type ListProductOptionValuesUseCase struct {
	repositories ListProductOptionValuesRepositories
	services     ListProductOptionValuesServices
}

// NewListProductOptionValuesUseCase creates a new ListProductOptionValuesUseCase
func NewListProductOptionValuesUseCase(
	repositories ListProductOptionValuesRepositories,
	services ListProductOptionValuesServices,
) *ListProductOptionValuesUseCase {
	return &ListProductOptionValuesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list product option values operation
func (uc *ListProductOptionValuesUseCase) Execute(ctx context.Context, req *productoptionvaluepb.ListProductOptionValuesRequest) (*productoptionvaluepb.ListProductOptionValuesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductOptionValue, ports.ActionList); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.errors.authorization_failed", "Authorization failed for product option values [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductOptionValue, ports.ActionList)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.errors.authorization_failed", "Authorization failed for product option values [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.errors.authorization_failed", "Authorization failed for product option values [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductOptionValue.ListProductOptionValues(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.errors.list_failed", "Failed to retrieve product option values [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListProductOptionValuesUseCase) validateInput(ctx context.Context, req *productoptionvaluepb.ListProductOptionValuesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_option_value.validation.request_required", "Request is required [DEFAULT]"))
	}
	// Additional validation can be added here if needed
	return nil
}
