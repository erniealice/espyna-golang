package price_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
)

// GetPricePlanListPageDataRepositories groups all repository dependencies
type GetPricePlanListPageDataRepositories struct {
	PricePlan priceplanpb.PricePlanDomainServiceServer // Primary entity repository
}

// GetPricePlanListPageDataServices groups all business service dependencies
type GetPricePlanListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetPricePlanListPageDataUseCase handles the business logic for getting price plan list page data
type GetPricePlanListPageDataUseCase struct {
	repositories GetPricePlanListPageDataRepositories
	services     GetPricePlanListPageDataServices
}

// NewGetPricePlanListPageDataUseCase creates use case with grouped dependencies
func NewGetPricePlanListPageDataUseCase(
	repositories GetPricePlanListPageDataRepositories,
	services GetPricePlanListPageDataServices,
) *GetPricePlanListPageDataUseCase {
	return &GetPricePlanListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get price plan list page data operation
func (uc *GetPricePlanListPageDataUseCase) Execute(ctx context.Context, req *priceplanpb.GetPricePlanListPageDataRequest) (*priceplanpb.GetPricePlanListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPricePlan, ports.ActionList); err != nil {
		return nil, err
	}


	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.request_required", "Request is required for price plan list page data"))
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

// executeWithTransaction executes price plan list page data retrieval within a transaction
func (uc *GetPricePlanListPageDataUseCase) executeWithTransaction(ctx context.Context, req *priceplanpb.GetPricePlanListPageDataRequest) (*priceplanpb.GetPricePlanListPageDataResponse, error) {
	var result *priceplanpb.GetPricePlanListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "price_plan.errors.get_list_page_data_failed", "")
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

// executeCore contains the core business logic for getting price plan list page data
func (uc *GetPricePlanListPageDataUseCase) executeCore(ctx context.Context, req *priceplanpb.GetPricePlanListPageDataRequest) (*priceplanpb.GetPricePlanListPageDataResponse, error) {
	// Delegate to repository
	return uc.repositories.PricePlan.GetPricePlanListPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetPricePlanListPageDataUseCase) validateInput(ctx context.Context, req *priceplanpb.GetPricePlanListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.request_required", ""))
	}

	// Validate pagination parameters
	if req.Pagination != nil {
		if req.Pagination.Limit < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.invalid_limit", "Pagination limit must be non-negative"))
		}
		if req.Pagination.Limit > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.limit_too_large", "Pagination limit cannot exceed 1000"))
		}
	}

	// Validate filter parameters for financial data
	if req.Filters != nil {
		for _, filter := range req.Filters.Filters {
			if filter.Field == "amount" {
				// Validate amount filter to prevent invalid monetary operations
				if filter.GetNumberFilter() != nil && filter.GetNumberFilter().Value < 0 {
					return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.invalid_amount_filter", "Amount filter cannot be negative"))
				}
			}
		}
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting price plan list page data
func (uc *GetPricePlanListPageDataUseCase) validateBusinessRules(ctx context.Context, req *priceplanpb.GetPricePlanListPageDataRequest) error {
	// Financial security: Ensure proper access control for price plan list data
	// Additional authorization checks would be implemented here in a real system
	// For example, certain price plans might be restricted based on user roles

	// Business rule: Apply data filtering based on user permissions
	// This would typically filter results based on user role and permissions
	// For example, only show active price plans to non-admin users

	// Business rule: Validate search and filter parameters for security
	if req.Search != nil && req.Search.Query != "" {
		// Prevent SQL injection and other malicious queries
		// In a real system, implement proper query sanitization
	}

	// Business rule: Ensure pricing data integrity
	// Validate that only appropriate users can access pricing information
	// This is critical for subscription billing systems

	return nil
}
