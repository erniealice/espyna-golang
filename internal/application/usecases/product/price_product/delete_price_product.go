package price_product

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
)

// DeletePriceProductRepositories groups all repository dependencies
type DeletePriceProductRepositories struct {
	PriceProduct priceproductpb.PriceProductDomainServiceServer // Primary entity repository
}

// DeletePriceProductServices groups all business service dependencies
type DeletePriceProductServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// DeletePriceProductUseCase handles the business logic for deleting price products
type DeletePriceProductUseCase struct {
	repositories DeletePriceProductRepositories
	services     DeletePriceProductServices
}

// NewDeletePriceProductUseCase creates a new DeletePriceProductUseCase
func NewDeletePriceProductUseCase(
	repositories DeletePriceProductRepositories,
	services DeletePriceProductServices,
) *DeletePriceProductUseCase {
	return &DeletePriceProductUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete price product operation
func (uc *DeletePriceProductUseCase) Execute(ctx context.Context, req *priceproductpb.DeletePriceProductRequest) (*priceproductpb.DeletePriceProductResponse, error) {
	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes price product deletion within a transaction
func (uc *DeletePriceProductUseCase) executeWithTransaction(ctx context.Context, req *priceproductpb.DeletePriceProductRequest) (*priceproductpb.DeletePriceProductResponse, error) {
	var result *priceproductpb.DeletePriceProductResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return result, nil
}

// executeCore contains the core business logic for deleting a price product
func (uc *DeletePriceProductUseCase) executeCore(ctx context.Context, req *priceproductpb.DeletePriceProductRequest) (*priceproductpb.DeletePriceProductResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.authorization_failed", "Authorization failed for price products [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityPriceProduct, ports.ActionDelete)
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

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.PriceProduct.DeletePriceProduct(ctx, req)
	if err != nil {
		// Handle not found error specifically - repository should return proper not found error
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.not_found", "Product pricing with ID \"{id}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{id}", req.Data.Id)
		return nil, errors.New(translatedError)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeletePriceProductUseCase) validateInput(ctx context.Context, req *priceproductpb.DeletePriceProductRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.data_required", "Price Product data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.id_required", "Price Product ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for price product deletion
func (uc *DeletePriceProductUseCase) validateBusinessRules(ctx context.Context, req *priceproductpb.DeletePriceProductRequest) error {
	// Additional business rule validation can be added here
	// For example: check if price product is in use by active subscriptions
	if uc.hasActiveSubscriptions(ctx, req.Data.Id) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.in_use", "Price product is currently in use and cannot be deleted [DEFAULT]"))
	}
	return nil
}

// hasActiveSubscriptions checks if there are active subscriptions using this price product
func (uc *DeletePriceProductUseCase) hasActiveSubscriptions(ctx context.Context, priceProductID string) bool {
	// This would typically query the subscription repository
	// For now, we'll return false as a placeholder
	// TODO: Implement actual check for active subscriptions
	return false
}
