package product_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_attribute"
)

// DeleteProductAttributeUseCase handles the business logic for deleting product attributes
// DeleteProductAttributeRepositories groups all repository dependencies
type DeleteProductAttributeRepositories struct {
	ProductAttribute productattributepb.ProductAttributeDomainServiceServer // Primary entity repository
}

// DeleteProductAttributeServices groups all business service dependencies
type DeleteProductAttributeServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// DeleteProductAttributeUseCase handles the business logic for deleting product attributes
type DeleteProductAttributeUseCase struct {
	repositories DeleteProductAttributeRepositories
	services     DeleteProductAttributeServices
}

// NewDeleteProductAttributeUseCase creates a new DeleteProductAttributeUseCase
func NewDeleteProductAttributeUseCase(
	repositories DeleteProductAttributeRepositories,
	services DeleteProductAttributeServices,
) *DeleteProductAttributeUseCase {
	return &DeleteProductAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete product attribute operation
func (uc *DeleteProductAttributeUseCase) Execute(ctx context.Context, req *productattributepb.DeleteProductAttributeRequest) (*productattributepb.DeleteProductAttributeResponse, error) {
	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product attribute deletion within a transaction
func (uc *DeleteProductAttributeUseCase) executeWithTransaction(ctx context.Context, req *productattributepb.DeleteProductAttributeRequest) (*productattributepb.DeleteProductAttributeResponse, error) {
	var result *productattributepb.DeleteProductAttributeResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return result, nil
}

// executeCore contains the core business logic for deleting a product attribute
func (uc *DeleteProductAttributeUseCase) executeCore(ctx context.Context, req *productattributepb.DeleteProductAttributeRequest) (*productattributepb.DeleteProductAttributeResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductAttribute, ports.ActionDelete)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductAttribute.DeleteProductAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.deletion_failed", "Product attribute deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteProductAttributeUseCase) validateInput(ctx context.Context, req *productattributepb.DeleteProductAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.validation.data_required", "Product attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.validation.id_required", "Product attribute ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for product attribute deletion
func (uc *DeleteProductAttributeUseCase) validateBusinessRules(ctx context.Context, req *productattributepb.DeleteProductAttributeRequest) error {
	// Additional business rule validation can be added here
	// For example: check if product attribute is referenced by other entities
	if uc.isProductAttributeInUse(ctx, req.Data.ProductId, req.Data.AttributeId) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.in_use", "Product attribute is currently in use and cannot be deleted [DEFAULT]"))
	}
	return nil
}

// isProductAttributeInUse checks if the product attribute is referenced by other entities
func (uc *DeleteProductAttributeUseCase) isProductAttributeInUse(ctx context.Context, productID, attributeID string) bool {
	// Placeholder for actual implementation
	// TODO: Implement actual check for product attribute usage
	return false
}
