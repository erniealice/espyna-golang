package product_plan

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	productplanpb "leapfor.xyz/esqyma/golang/v1/domain/product/product_plan"
)

type GetProductPlanItemPageDataRepositories struct {
	ProductPlan productplanpb.ProductPlanDomainServiceServer
}

type GetProductPlanItemPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetProductPlanItemPageDataUseCase handles the business logic for getting product plan item page data
type GetProductPlanItemPageDataUseCase struct {
	repositories GetProductPlanItemPageDataRepositories
	services     GetProductPlanItemPageDataServices
}

// NewGetProductPlanItemPageDataUseCase creates a new GetProductPlanItemPageDataUseCase
func NewGetProductPlanItemPageDataUseCase(
	repositories GetProductPlanItemPageDataRepositories,
	services GetProductPlanItemPageDataServices,
) *GetProductPlanItemPageDataUseCase {
	return &GetProductPlanItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get product plan item page data operation
func (uc *GetProductPlanItemPageDataUseCase) Execute(
	ctx context.Context,
	req *productplanpb.GetProductPlanItemPageDataRequest,
) (*productplanpb.GetProductPlanItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.ProductPlanId); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product plan item page data retrieval within a transaction
func (uc *GetProductPlanItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *productplanpb.GetProductPlanItemPageDataRequest,
) (*productplanpb.GetProductPlanItemPageDataResponse, error) {
	var result *productplanpb.GetProductPlanItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"product_plan.errors.item_page_data_failed",
				"product plan item page data retrieval failed: %w",
			), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting product plan item page data
func (uc *GetProductPlanItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *productplanpb.GetProductPlanItemPageDataRequest,
) (*productplanpb.GetProductPlanItemPageDataResponse, error) {
	// Create read request for the product plan
	readReq := &productplanpb.ReadProductPlanRequest{
		Data: &productplanpb.ProductPlan{
			Id: req.ProductPlanId,
		},
	}

	// Retrieve the product plan
	readResp, err := uc.repositories.ProductPlan.ReadProductPlan(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"product_plan.errors.read_failed",
			"failed to retrieve product plan: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"product_plan.errors.not_found",
			"product plan not found",
		))
	}

	// Get the product plan (should be only one)
	productPlan := readResp.Data[0]

	// Validate that we got the expected product plan
	if productPlan.Id != req.ProductPlanId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"product_plan.errors.id_mismatch",
			"retrieved product plan ID does not match requested ID",
		))
	}

	// TODO: In a real implementation, you might want to:
	// 1. Load related data (product details, pricing tiers, features) if not already populated
	// 2. Apply business rules for data visibility/access control
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging for item access
	// 5. Calculate derived fields like effective pricing, trial information

	// For now, return the product plan as-is
	return &productplanpb.GetProductPlanItemPageDataResponse{
		ProductPlan: productPlan,
		Success:     true,
	}, nil
}

// validateInput validates the input request
func (uc *GetProductPlanItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *productplanpb.GetProductPlanItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"product_plan.validation.request_required",
			"request is required",
		))
	}

	if req.ProductPlanId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"product_plan.validation.id_required",
			"product plan ID is required",
		))
	}

	return nil
}

// validateBusinessRules enforces business constraints for reading product plan item page data
func (uc *GetProductPlanItemPageDataUseCase) validateBusinessRules(
	ctx context.Context,
	productPlanId string,
) error {
	// Validate product plan ID format
	if len(productPlanId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"product_plan.validation.id_too_short",
			"product plan ID is too short",
		))
	}

	// Additional business rules could be added here:
	// - Check user permissions to access this product plan
	// - Validate product plan belongs to the current user's organization
	// - Check if product plan is in a state that allows viewing
	// - Rate limiting for product plan access
	// - Audit logging requirements
	// - Validate pricing model constraints
	// - Check feature access permissions

	return nil
}

// Optional: Helper methods for future enhancements

// loadRelatedData loads related entities like product and pricing details
// This would be called from executeCore if needed
func (uc *GetProductPlanItemPageDataUseCase) loadRelatedData(
	ctx context.Context,
	productPlan *productplanpb.ProductPlan,
) error {
	// TODO: Implement loading of related data
	// This could involve calls to product and pricing repositories
	// to populate the nested product object and pricing details if they're not already loaded

	// Example implementation would be:
	// if productPlan.Product == nil && productPlan.ProductId != "" {
	//     // Load product data
	// }
	// if productPlan.PricingTiers == nil {
	//     // Load pricing tier data
	// }

	return nil
}

// applyDataTransformation applies any necessary data transformations for the frontend
func (uc *GetProductPlanItemPageDataUseCase) applyDataTransformation(
	ctx context.Context,
	productPlan *productplanpb.ProductPlan,
) *productplanpb.ProductPlan {
	// TODO: Apply any transformations needed for optimal frontend consumption
	// This could include:
	// - Formatting pricing information
	// - Computing derived fields like effective monthly cost
	// - Converting billing cycles to user-friendly display formats
	// - Applying localization for currency and dates
	// - Sanitizing sensitive pricing data based on user permissions
	// - Adding calculated fields for trial information

	return productPlan
}

// checkAccessPermissions validates user has permission to access this product plan
func (uc *GetProductPlanItemPageDataUseCase) checkAccessPermissions(
	ctx context.Context,
	productPlanId string,
) error {
	// TODO: Implement proper access control
	// This could involve:
	// - Checking user role/permissions
	// - Validating product plan belongs to user's organization
	// - Applying multi-tenant access controls
	// - Checking product plan visibility settings
	// - Validating feature access permissions

	return nil
}

// calculatePricingMetrics computes derived pricing information
func (uc *GetProductPlanItemPageDataUseCase) calculatePricingMetrics(
	ctx context.Context,
	productPlan *productplanpb.ProductPlan,
) error {
	// TODO: Calculate derived pricing metrics
	// This could include:
	// - Effective monthly/yearly costs
	// - Discount calculations
	// - Tax implications
	// - Currency conversions
	// - Proration calculations

	return nil
}
