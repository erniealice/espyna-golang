package product

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// CreateProductRepositories groups all repository dependencies
type CreateProductRepositories struct {
	Product productpb.ProductDomainServiceServer // Primary entity repository
}

// CreateProductServices groups all business service dependencies
type CreateProductServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateProductUseCase handles the business logic for creating products
type CreateProductUseCase struct {
	repositories CreateProductRepositories
	services     CreateProductServices
}

// NewCreateProductUseCase creates use case with grouped dependencies
func NewCreateProductUseCase(
	repositories CreateProductRepositories,
	services CreateProductServices,
) *CreateProductUseCase {
	return &CreateProductUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create product operation
func (uc *CreateProductUseCase) Execute(ctx context.Context, req *productpb.CreateProductRequest) (*productpb.CreateProductResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product creation within a transaction
func (uc *CreateProductUseCase) executeWithTransaction(ctx context.Context, req *productpb.CreateProductRequest) (*productpb.CreateProductResponse, error) {
	var result *productpb.CreateProductResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf("product creation failed: %w", err)
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
func (uc *CreateProductUseCase) executeCore(ctx context.Context, req *productpb.CreateProductRequest) (*productpb.CreateProductResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product.errors.authorization_failed", "Authorization failed for products [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProduct, ports.ActionCreate)
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

	// Business logic and enrichment
	if err := uc.enrichProductData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	return uc.repositories.Product.CreateProduct(ctx, req)
}

// validateInput validates the input request
func (uc *CreateProductUseCase) validateInput(ctx context.Context, req *productpb.CreateProductRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product.validation.data_required", "Product data is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product.validation.name_required", "Product name is required [DEFAULT]"))
	}
	return nil
}

// enrichProductData adds generated fields and audit information
func (uc *CreateProductUseCase) enrichProductData(product *productpb.Product) error {
	now := time.Now()

	// Generate Product ID if not provided
	if product.Id == "" {
		product.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	product.DateCreated = &[]int64{now.UnixMilli()}[0]
	product.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	product.DateModified = &[]int64{now.UnixMilli()}[0]
	product.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	product.Active = true

	return nil
}

// validateBusinessRules enforces business constraints for products
func (uc *CreateProductUseCase) validateBusinessRules(ctx context.Context, product *productpb.Product) error {
	// Validate product name length
	name := strings.TrimSpace(product.Name)
	if len(name) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product.validation.name_min_length", "Product name must be at least 2 characters long [DEFAULT]"))
	}

	if len(name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product.validation.name_max_length", "Product name cannot exceed 100 characters [DEFAULT]"))
	}

	// Validate description length if provided
	if product.Description != nil && *product.Description != "" {
		description := strings.TrimSpace(*product.Description)
		if len(description) > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product.validation.description_max_length", "Product description cannot exceed 1000 characters [DEFAULT]"))
		}
	}

	// Normalize name (trim spaces, proper capitalization)
	product.Name = strings.Title(strings.ToLower(name))

	return nil
}
