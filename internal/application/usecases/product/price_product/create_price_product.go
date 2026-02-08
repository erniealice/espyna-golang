package price_product

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// CreatePriceProductRepositories groups all repository dependencies
type CreatePriceProductRepositories struct {
	PriceProduct priceproductpb.PriceProductDomainServiceServer // Primary entity repository
	Product      productpb.ProductDomainServiceServer           // Entity reference validation
}

// CreatePriceProductServices groups all business service dependencies
type CreatePriceProductServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Transaction management
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreatePriceProductUseCase handles the business logic for creating price products
type CreatePriceProductUseCase struct {
	repositories CreatePriceProductRepositories
	services     CreatePriceProductServices
}

// NewCreatePriceProductUseCase creates use case with grouped dependencies
func NewCreatePriceProductUseCase(
	repositories CreatePriceProductRepositories,
	services CreatePriceProductServices,
) *CreatePriceProductUseCase {
	return &CreatePriceProductUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create price product operation
func (uc *CreatePriceProductUseCase) Execute(ctx context.Context, req *priceproductpb.CreatePriceProductRequest) (*priceproductpb.CreatePriceProductResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.authorization_failed", "Authorization failed for product pricing [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityPriceProduct, ports.ActionCreate)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.authorization_failed", "Authorization failed for product pricing [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.authorization_failed", "Authorization failed for product pricing [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichPriceProductData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Determine if we should use transactions
	if uc.shouldUseTransaction(ctx) {
		return uc.executeWithTransaction(ctx, req)
	}

	// Execute without transaction (backward compatibility)
	return uc.executeWithoutTransaction(ctx, req)
}

// shouldUseTransaction determines if this operation should use a transaction
func (uc *CreatePriceProductUseCase) shouldUseTransaction(ctx context.Context) bool {
	// Use transaction if:
	// 1. TransactionService is available, AND
	// 2. We're not already in a transaction context
	if uc.services.TransactionService == nil || !uc.services.TransactionService.SupportsTransactions() {
		return false
	}

	// Don't start a nested transaction if we're already in one
	if uc.services.TransactionService.IsTransactionActive(ctx) {
		return false
	}

	return true
}

// executeWithTransaction performs the operation within a transaction
func (uc *CreatePriceProductUseCase) executeWithTransaction(ctx context.Context, req *priceproductpb.CreatePriceProductRequest) (*priceproductpb.CreatePriceProductResponse, error) {
	var response *priceproductpb.CreatePriceProductResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// All validations and operations within transaction

		// Entity reference validation (reads happen in transaction context)
		if err := uc.validateEntityReferences(txCtx, req.Data); err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "price_product.errors.reference_validation_failed", "Entity reference validation failed [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}

		// Business rule validation
		if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "price_product.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}

		// Create PriceProduct (will participate in transaction)
		createResponse, err := uc.repositories.PriceProduct.CreatePriceProduct(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "price_product.errors.creation_failed", "Failed to create price product [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}

		response = createResponse
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("transaction execution failed: %w", err)
	}

	return response, nil
}

// executeWithoutTransaction performs the operation without transaction (backward compatibility)
func (uc *CreatePriceProductUseCase) executeWithoutTransaction(ctx context.Context, req *priceproductpb.CreatePriceProductRequest) (*priceproductpb.CreatePriceProductResponse, error) {
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

	// Call repository (no transaction)
	resp, err := uc.repositories.PriceProduct.CreatePriceProduct(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.creation_failed", "Failed to create price product [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *CreatePriceProductUseCase) validateInput(ctx context.Context, req *priceproductpb.CreatePriceProductRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.request_required", "Request is required for product pricing [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.data_required", "Product pricing data is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.name_required", "Product pricing name is required [DEFAULT]"))
	}
	if req.Data.ProductId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.product_id_required", "Product ID is required for product pricing [DEFAULT]"))
	}
	if req.Data.Amount < 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.amount_invalid", "Product pricing amount must be non-negative [DEFAULT]"))
	}
	if req.Data.Currency == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.validation.currency_required", "Currency is required for product pricing [DEFAULT]"))
	}
	return nil
}

// enrichPriceProductData adds generated fields and audit information
func (uc *CreatePriceProductUseCase) enrichPriceProductData(priceProduct *priceproductpb.PriceProduct) error {
	now := time.Now()

	// Generate PriceProduct ID if not provided
	if priceProduct.Id == "" {
		priceProduct.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	priceProduct.DateCreated = &[]int64{now.UnixMilli()}[0]
	priceProduct.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	priceProduct.DateModified = &[]int64{now.UnixMilli()}[0]
	priceProduct.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	priceProduct.Active = true

	return nil
}

// validateBusinessRules enforces business constraints for price products
func (uc *CreatePriceProductUseCase) validateBusinessRules(ctx context.Context, priceProduct *priceproductpb.PriceProduct) error {
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
func (uc *CreatePriceProductUseCase) validateEntityReferences(ctx context.Context, priceProduct *priceproductpb.PriceProduct) error {
	// Validate Product entity reference
	if priceProduct.ProductId != "" {
		product, err := uc.repositories.Product.ReadProduct(ctx, &productpb.ReadProductRequest{
			Data: &productpb.Product{Id: priceProduct.ProductId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.product_reference_validation_failed", "Failed to validate product entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if product == nil || product.Data == nil || len(product.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.product_not_found", "Referenced product with ID '{productId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{productId}", priceProduct.ProductId)
			return errors.New(translatedError)
		}
		if !product.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_product.errors.product_not_active", "Referenced product with ID '{productId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{productId}", priceProduct.ProductId)
			return errors.New(translatedError)
		}
	}

	return nil
}
