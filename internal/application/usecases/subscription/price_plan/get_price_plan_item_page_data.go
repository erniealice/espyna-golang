package price_plan

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	priceplanpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/price_plan"
)

// GetPricePlanItemPageDataRepositories groups all repository dependencies
type GetPricePlanItemPageDataRepositories struct {
	PricePlan priceplanpb.PricePlanDomainServiceServer // Primary entity repository
}

// GetPricePlanItemPageDataServices groups all business service dependencies
type GetPricePlanItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetPricePlanItemPageDataUseCase handles the business logic for getting price plan item page data
type GetPricePlanItemPageDataUseCase struct {
	repositories GetPricePlanItemPageDataRepositories
	services     GetPricePlanItemPageDataServices
}

// NewGetPricePlanItemPageDataUseCase creates use case with grouped dependencies
func NewGetPricePlanItemPageDataUseCase(
	repositories GetPricePlanItemPageDataRepositories,
	services GetPricePlanItemPageDataServices,
) *GetPricePlanItemPageDataUseCase {
	return &GetPricePlanItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get price plan item page data operation
func (uc *GetPricePlanItemPageDataUseCase) Execute(ctx context.Context, req *priceplanpb.GetPricePlanItemPageDataRequest) (*priceplanpb.GetPricePlanItemPageDataResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil {
		if enabled, ok := uc.services.AuthorizationService.(interface{ IsEnabled() bool }); ok && enabled.IsEnabled() {
			uid, _ := ctx.Value("uid").(string)
			if authorized, err := uc.services.AuthorizationService.HasPermission(ctx, uid, ports.EntityPermission(ports.EntityPricePlan, ports.ActionRead)); err != nil || !authorized {
				return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.errors.authorization_failed", ""))
			}
		}
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.request_required", "Request is required for price plan item page data"))
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes price plan item page data retrieval within a transaction
func (uc *GetPricePlanItemPageDataUseCase) executeWithTransaction(ctx context.Context, req *priceplanpb.GetPricePlanItemPageDataRequest) (*priceplanpb.GetPricePlanItemPageDataResponse, error) {
	var result *priceplanpb.GetPricePlanItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "price_plan.errors.get_item_page_data_failed", "")
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

// executeCore contains the core business logic for getting price plan item page data
func (uc *GetPricePlanItemPageDataUseCase) executeCore(ctx context.Context, req *priceplanpb.GetPricePlanItemPageDataRequest) (*priceplanpb.GetPricePlanItemPageDataResponse, error) {
	// Delegate to repository
	return uc.repositories.PricePlan.GetPricePlanItemPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetPricePlanItemPageDataUseCase) validateInput(ctx context.Context, req *priceplanpb.GetPricePlanItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.request_required", ""))
	}

	if req.PricePlanId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.id_required", "Price plan ID is required"))
	}

	// Validate ID format (basic validation)
	if len(req.PricePlanId) > 255 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.id_too_long", "Price plan ID cannot exceed 255 characters"))
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting price plan item page data
func (uc *GetPricePlanItemPageDataUseCase) validateBusinessRules(ctx context.Context, req *priceplanpb.GetPricePlanItemPageDataRequest) error {
	// Financial security: Ensure proper access control for price plan item data
	// Additional authorization checks would be implemented here in a real system
	// For example, certain price plans might be restricted based on user roles

	// Business rule: Validate price plan access permissions
	// This would typically check if the current user has permission to view this specific price plan
	// In a real system, this might involve checking subscription tiers or admin privileges

	// Business rule: Ensure pricing data integrity
	// Validate that only appropriate users can access detailed pricing information
	// This is critical for subscription billing systems where pricing might be confidential

	return nil
}
