package product

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// DeleteProductRepositories groups all repository dependencies
type DeleteProductRepositories struct {
	Product productpb.ProductDomainServiceServer // Primary entity repository
}

// DeleteProductServices groups all business service dependencies
type DeleteProductServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// DeleteProductUseCase handles the business logic for deleting products
type DeleteProductUseCase struct {
	repositories DeleteProductRepositories
	services     DeleteProductServices
}

// NewDeleteProductUseCase creates a new DeleteProductUseCase
func NewDeleteProductUseCase(
	repositories DeleteProductRepositories,
	services DeleteProductServices,
) *DeleteProductUseCase {
	return &DeleteProductUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete product operation
func (uc *DeleteProductUseCase) Execute(ctx context.Context, req *productpb.DeleteProductRequest) (*productpb.DeleteProductResponse, error) {
	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product deletion within a transaction
func (uc *DeleteProductUseCase) executeWithTransaction(ctx context.Context, req *productpb.DeleteProductRequest) (*productpb.DeleteProductResponse, error) {
	var result *productpb.DeleteProductResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return result, nil
}

// executeCore contains the core business logic for deleting a product
func (uc *DeleteProductUseCase) executeCore(ctx context.Context, req *productpb.DeleteProductRequest) (*productpb.DeleteProductResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product.errors.authorization_failed", "Authorization failed for products [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProduct, ports.ActionDelete)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product.errors.authorization_failed", "Authorization failed for products [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product.errors.authorization_failed", "Authorization failed for products [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Product.DeleteProduct(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product.errors.deletion_failed", "Product deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteProductUseCase) validateInput(ctx context.Context, req *productpb.DeleteProductRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product.validation.data_required", "Product data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product.validation.id_required", "Product ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for product deletion
func (uc *DeleteProductUseCase) validateBusinessRules(ctx context.Context, req *productpb.DeleteProductRequest) error {
	// Additional business rule validation can be added here
	// For example: check if product is in use by active collections or subscriptions
	if uc.isProductInUse(ctx, req.Data.Id) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product.errors.in_use", "Product is currently in use and cannot be deleted [DEFAULT]"))
	}
	return nil
}

// isProductInUse checks if the product is referenced by other entities (e.g., collections, subscriptions)
func (uc *DeleteProductUseCase) isProductInUse(ctx context.Context, productID string) bool {
	// Placeholder for actual implementation
	// TODO: Implement actual check for product usage
	return false
}
