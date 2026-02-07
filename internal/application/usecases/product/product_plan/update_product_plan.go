package product_plan

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	productpb "leapfor.xyz/esqyma/golang/v1/domain/product/product"
	productplanpb "leapfor.xyz/esqyma/golang/v1/domain/product/product_plan"
)

// UpdateProductPlanRepositories groups all repository dependencies
type UpdateProductPlanRepositories struct {
	ProductPlan productplanpb.ProductPlanDomainServiceServer // Primary entity repository
	Product     productpb.ProductDomainServiceServer         // Entity reference dependency
}

// UpdateProductPlanServices groups all business service dependencies
type UpdateProductPlanServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// UpdateProductPlanUseCase handles the business logic for updating product plans
type UpdateProductPlanUseCase struct {
	repositories UpdateProductPlanRepositories
	services     UpdateProductPlanServices
}

// NewUpdateProductPlanUseCase creates a new UpdateProductPlanUseCase
func NewUpdateProductPlanUseCase(
	repositories UpdateProductPlanRepositories,
	services UpdateProductPlanServices,
) *UpdateProductPlanUseCase {
	return &UpdateProductPlanUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update product plan operation
func (uc *UpdateProductPlanUseCase) Execute(ctx context.Context, req *productplanpb.UpdateProductPlanRequest) (*productplanpb.UpdateProductPlanResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.authorization_failed", "Authorization failed for product plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductPlan, ports.ActionUpdate)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.authorization_failed", "Authorization failed for product plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.authorization_failed", "Authorization failed for product plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product plan update within a transaction
func (uc *UpdateProductPlanUseCase) executeWithTransaction(ctx context.Context, req *productplanpb.UpdateProductPlanRequest) (*productplanpb.UpdateProductPlanResponse, error) {
	var result *productplanpb.UpdateProductPlanResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.update_failed", "Product plan update failed [DEFAULT]")
			return fmt.Errorf("%s: %w", msg, err)
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
func (uc *UpdateProductPlanUseCase) executeCore(ctx context.Context, req *productplanpb.UpdateProductPlanRequest) (*productplanpb.UpdateProductPlanResponse, error) {
	// Input validation with translation support
	if err := uc.validateInputWithTranslation(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichProductPlanData(req.Data); err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}

	// Entity reference validation with translation support
	if err := uc.validateEntityReferencesWithTranslation(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.entity_reference_validation_failed", "Entity reference validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation with translation support
	if err := uc.validateBusinessRulesWithTranslation(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductPlan.UpdateProductPlan(ctx, req)
	if err != nil {
		// Check if it's a not found error and convert to translated message
		if strings.Contains(err.Error(), "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "product_plan.errors.not_found", map[string]interface{}{"productPlanId": req.Data.Id}, "Course plan not found")
			return nil, errors.New(translatedError)
		}
		// Other repository errors
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.update_failed", "Product update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateProductPlanUseCase) validateInput(req *productplanpb.UpdateProductPlanRequest) error {
	// This function is deprecated and should not be used directly. Use validateInputWithTranslation instead.
	return errors.New("deprecated: use validateInputWithTranslation")
}

// validateInputWithTranslation validates the input request with translated messages
func (uc *UpdateProductPlanUseCase) validateInputWithTranslation(ctx context.Context, req *productplanpb.UpdateProductPlanRequest) error {
	if req == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.validation.request_required", "Request is required [DEFAULT]")
		return errors.New(msg)
	}
	if req.Data == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.validation.data_required", "Product plan data is required [DEFAULT]")
		return errors.New(msg)
	}
	if req.Data.Id == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.validation.id_required", "Product plan ID is required [DEFAULT]")
		return errors.New(msg)
	}
	if req.Data.Name == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.validation.name_required", "Product plan name is required [DEFAULT]")
		return errors.New(msg)
	}
	if req.Data.ProductId == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.validation.product_id_required", "Product ID is required [DEFAULT]")
		return errors.New(msg)
	}
	return nil
}

// enrichProductPlanData adds generated fields and audit information
func (uc *UpdateProductPlanUseCase) enrichProductPlanData(productPlan *productplanpb.ProductPlan) error {
	now := time.Now()

	// Update audit fields
	productPlan.DateModified = &[]int64{now.UnixMilli()}[0]
	productPlan.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints for product plans
func (uc *UpdateProductPlanUseCase) validateBusinessRules(productPlan *productplanpb.ProductPlan) error {
	// This function is deprecated and should not be used directly. Use validateBusinessRulesWithTranslation instead.
	return errors.New("deprecated: use validateBusinessRulesWithTranslation")
}

// validateBusinessRulesWithTranslation enforces business constraints with translated messages
func (uc *UpdateProductPlanUseCase) validateBusinessRulesWithTranslation(ctx context.Context, productPlan *productplanpb.ProductPlan) error {
	// Validate product plan name length
	if len(productPlan.Name) < 3 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.validation.name_min_length", "Product plan name must be at least 3 characters long [DEFAULT]")
		return errors.New(msg)
	}

	if len(productPlan.Name) > 100 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.validation.name_max_length", "Product plan name cannot exceed 100 characters [DEFAULT]")
		return errors.New(msg)
	}

	// Validate product ID format
	if len(productPlan.ProductId) < 5 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.validation.product_id_min_length", "Product ID must be at least 5 characters long [DEFAULT]")
		return errors.New(msg)
	}

	// Business constraint: Product plan must be associated with valid product and plan
	if productPlan.ProductId == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.validation.product_association_required", "Product plan must be associated with a product [DEFAULT]")
		return errors.New(msg)
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateProductPlanUseCase) validateEntityReferences(ctx context.Context, productPlan *productplanpb.ProductPlan) error {
	// This function is deprecated and should not be used directly. Use validateEntityReferencesWithTranslation instead.
	return errors.New("deprecated: use validateEntityReferencesWithTranslation")
}

// validateEntityReferencesWithTranslation validates entity references with translated messages
func (uc *UpdateProductPlanUseCase) validateEntityReferencesWithTranslation(ctx context.Context, productPlan *productplanpb.ProductPlan) error {
	// Validate ProductId entity reference
	if productPlan.ProductId != "" {
		product, err := uc.repositories.Product.ReadProduct(ctx, &productpb.ReadProductRequest{
			Data: &productpb.Product{Id: productPlan.ProductId},
		})
		if err != nil {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.product_validation_failed", "Referenced entity validation failed [DEFAULT]")
			return fmt.Errorf("%s: %w", msg, err)
		}
		if product == nil || product.Data == nil || len(product.Data) == 0 {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.product_not_available", "Referenced product is not available [DEFAULT]")
			return fmt.Errorf("%s with ID '%s'", msg, productPlan.ProductId)
		}
		if !product.Data[0].Active {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.product_not_active", "Referenced product is not active [DEFAULT]")
			return fmt.Errorf("%s with ID '%s'", msg, productPlan.ProductId)
		}
	}

	return nil
}
