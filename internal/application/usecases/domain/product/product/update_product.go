package product

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// UpdateProductRepositories groups all repository dependencies
type UpdateProductRepositories struct {
	Product productpb.ProductDomainServiceServer // Primary entity repository
}

// UpdateProductServices groups all business service dependencies
type UpdateProductServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateProductUseCase handles the business logic for updating products
type UpdateProductUseCase struct {
	repositories UpdateProductRepositories
	services     UpdateProductServices
}

// NewUpdateProductUseCase creates use case with grouped dependencies
func NewUpdateProductUseCase(
	repositories UpdateProductRepositories,
	services UpdateProductServices,
) *UpdateProductUseCase {
	return &UpdateProductUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update product operation
func (uc *UpdateProductUseCase) Execute(ctx context.Context, req *productpb.UpdateProductRequest) (*productpb.UpdateProductResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Product,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product update within a transaction
func (uc *UpdateProductUseCase) executeWithTransaction(ctx context.Context, req *productpb.UpdateProductRequest) (*productpb.UpdateProductResponse, error) {
	var result *productpb.UpdateProductResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "product.errors.update_failed", "Product update failed [DEFAULT]")
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
func (uc *UpdateProductUseCase) executeCore(ctx context.Context, req *productpb.UpdateProductRequest) (*productpb.UpdateProductResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.errors.authorization_failed", "Authorization failed for products [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.Product, entityid.ActionUpdate)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.errors.authorization_failed", "Authorization failed for products [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.errors.authorization_failed", "Authorization failed for products [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichProductData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Get the existing product to preserve fields not included in the update
	existingProductResp, err := uc.repositories.Product.ReadProduct(ctx, &productpb.ReadProductRequest{Data: &productpb.Product{Id: req.Data.Id}})
	if err != nil || existingProductResp == nil || len(existingProductResp.Data) == 0 {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.errors.not_found", "Product not found for update [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	existingProduct := existingProductResp.Data[0]

	// Preserve the active status if not provided in the request
	if req.Data.Active == false { // Protobuf bool defaults to false
		req.Data.Active = existingProduct.Active
	}

	// Call repository
	resp, err := uc.repositories.Product.UpdateProduct(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.errors.update_failed", "Product update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateProductUseCase) validateInput(ctx context.Context, req *productpb.UpdateProductRequest) error {

	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.validation.data_required", "Product data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.validation.id_required", "Product ID is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.validation.name_required", "Product name is required [DEFAULT]"))
	}
	return nil
}

// enrichProductData adds generated fields and audit information
func (uc *UpdateProductUseCase) enrichProductData(product *productpb.Product) error {
	now := time.Now()

	// Update audit fields
	product.DateModified = &[]int64{now.UnixMilli()}[0]
	product.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints for products
func (uc *UpdateProductUseCase) validateBusinessRules(ctx context.Context, product *productpb.Product) error {

	// Validate product name length
	name := strings.TrimSpace(product.Name)
	if len(name) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.validation.name_min_length", "Product name must be at least 2 characters long [DEFAULT]"))
	}

	if len(name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.validation.name_max_length", "Product name cannot exceed 100 characters [DEFAULT]"))
	}

	// Validate description length if provided
	if product.Description != nil && *product.Description != "" {
		description := strings.TrimSpace(*product.Description)
		if len(description) > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.validation.description_max_length", "Product description cannot exceed 1000 characters [DEFAULT]"))
		}
	}

	// Model D price rule: simple + active => price required; configurable =>
	// price optional (per-variant pricing); draft => skip.
	if err := uc.validateVariantModePriceRule(ctx, product); err != nil {
		return err
	}

	// Normalize name (trim spaces, proper capitalization)
	product.Name = strings.TrimSpace(name)

	return nil
}

// validateVariantModePriceRule mirrors the create-side rule for updates.
func (uc *UpdateProductUseCase) validateVariantModePriceRule(ctx context.Context, product *productpb.Product) error {
	if !product.Active {
		return nil
	}
	if product.VariantMode == "configurable" {
		return nil
	}
	if product.Price == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product.validation.price_required_for_simple", "Price is required for simple products [DEFAULT]")
		return errors.New(msg)
	}
	return nil
}
