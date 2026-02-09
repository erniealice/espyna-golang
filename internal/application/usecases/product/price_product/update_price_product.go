package price_product

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// UpdatePriceProductRepositories groups all repository dependencies
type UpdatePriceProductRepositories struct {
	PriceProduct priceproductpb.PriceProductDomainServiceServer // Primary entity repository
	Product      productpb.ProductDomainServiceServer           // Entity reference dependency
}

// UpdatePriceProductServices groups all business service dependencies
type UpdatePriceProductServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// UpdatePriceProductUseCase handles the business logic for updating price products
type UpdatePriceProductUseCase struct {
	repositories UpdatePriceProductRepositories
	services     UpdatePriceProductServices
}

// NewUpdatePriceProductUseCase creates a new UpdatePriceProductUseCase
func NewUpdatePriceProductUseCase(
	repositories UpdatePriceProductRepositories,
	services UpdatePriceProductServices,
) *UpdatePriceProductUseCase {
	return &UpdatePriceProductUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update price product operation
func (uc *UpdatePriceProductUseCase) Execute(ctx context.Context, req *priceproductpb.UpdatePriceProductRequest) (*priceproductpb.UpdatePriceProductResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPriceProduct, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// businessType := uc.getBusinessTypeFromContext(ctx)

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes price product update within a transaction
func (uc *UpdatePriceProductUseCase) executeWithTransaction(ctx context.Context, req *priceproductpb.UpdatePriceProductRequest) (*priceproductpb.UpdatePriceProductResponse, error) {
	var result *priceproductpb.UpdatePriceProductResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "price_product.errors.update_failed", "Price Product update failed [DEFAULT]")
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

// executeCore contains the core business logic (moved from original Execute method)
func (uc *UpdatePriceProductUseCase) executeCore(ctx context.Context, req *priceproductpb.UpdatePriceProductRequest) (*priceproductpb.UpdatePriceProductResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.authorization_failed", "Authorization failed for price products [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityPriceProduct, ports.ActionUpdate)
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

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.reference_validation_failed", "Entity reference validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.PriceProduct.UpdatePriceProduct(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.update_failed", "Price Product update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched price plan
func (uc *UpdatePriceProductUseCase) applyBusinessLogic(priceProduct *priceproductpb.PriceProduct) *priceproductpb.PriceProduct {
	now := time.Now()

	// Business logic: Update modification audit fields
	priceProduct.DateModified = &[]int64{now.UnixMilli()}[0]
	priceProduct.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return priceProduct
}

// validateInput validates the input request
func (uc *UpdatePriceProductUseCase) validateInput(ctx context.Context, req *priceproductpb.UpdatePriceProductRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.data_required", "Price Product data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.id_required", "Price Product ID is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.name_required", "Price Product name is required [DEFAULT]"))
	}
	if req.Data.ProductId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.product_id_required", "Product ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for price products
func (uc *UpdatePriceProductUseCase) validateBusinessRules(ctx context.Context, priceProduct *priceproductpb.PriceProduct) error {

	// Validate price product name length
	if len(priceProduct.Name) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.name_min_length", "Price product name must be at least 3 characters long [DEFAULT]"))
	}

	if len(priceProduct.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.name_max_length", "Price product name cannot exceed 100 characters [DEFAULT]"))
	}

	// Validate product ID format
	if len(priceProduct.ProductId) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.product_id_min_length", "Product ID must be at least 5 characters long [DEFAULT]"))
	}

	// Business constraint: Price product must be associated with a valid product
	if priceProduct.ProductId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.product_association_required", "Price product must be associated with a product [DEFAULT]"))
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdatePriceProductUseCase) validateEntityReferences(ctx context.Context, priceProduct *priceproductpb.PriceProduct) error {

	// Validate Product entity reference
	if priceProduct.ProductId != "" {
		result, err := uc.repositories.Product.ReadProduct(ctx, &productpb.ReadProductRequest{
			Data: &productpb.Product{Id: priceProduct.ProductId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.product_reference_validation_failed", "Failed to validate product entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if result == nil || result.Data == nil || len(result.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.product_not_found", "Referenced product with ID '{productId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{productId}", priceProduct.ProductId)
			return errors.New(translatedError)
		}
		if !result.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.product_not_active", "Referenced product with ID '{productId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{productId}", priceProduct.ProductId)
			return errors.New(translatedError)
		}
	}

	return nil
}
