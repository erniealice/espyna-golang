package price_product

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
)

// ListPriceProductsRepositories groups all repository dependencies
type ListPriceProductsRepositories struct {
	PriceProduct priceproductpb.PriceProductDomainServiceServer // Primary entity repository
}

// ListPriceProductsServices groups all business service dependencies
type ListPriceProductsServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ListPriceProductsUseCase handles the business logic for listing price products
type ListPriceProductsUseCase struct {
	repositories ListPriceProductsRepositories
	services     ListPriceProductsServices
}

// NewListPriceProductsUseCase creates a new ListPriceProductsUseCase
func NewListPriceProductsUseCase(
	repositories ListPriceProductsRepositories,
	services ListPriceProductsServices,
) *ListPriceProductsUseCase {
	return &ListPriceProductsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list price products operation
func (uc *ListPriceProductsUseCase) Execute(ctx context.Context, req *priceproductpb.ListPriceProductsRequest) (*priceproductpb.ListPriceProductsResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.authorization_failed", "Authorization failed for price products [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityPriceProduct, ports.ActionList)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.authorization_failed", "Authorization failed for price products [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.authorization_failed", "Authorization failed for price products [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.PriceProduct.ListPriceProducts(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.list_failed", "Failed to retrieve price products [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListPriceProductsUseCase) validateInput(ctx context.Context, req *priceproductpb.ListPriceProductsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.request_required", "Request is required [DEFAULT]"))
	}
	// Additional validation can be added here if needed
	return nil
}
