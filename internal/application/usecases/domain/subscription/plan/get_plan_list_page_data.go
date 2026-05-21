package plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
)

// GetPlanListPageDataRepositories groups all repository dependencies
type GetPlanListPageDataRepositories struct {
	Plan planpb.PlanDomainServiceServer // Primary entity repository
}

// GetPlanListPageDataServices groups all business service dependencies
type GetPlanListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetPlanListPageDataUseCase handles the business logic for getting plan list page data
type GetPlanListPageDataUseCase struct {
	repositories GetPlanListPageDataRepositories
	services     GetPlanListPageDataServices
}

// NewGetPlanListPageDataUseCase creates use case with grouped dependencies
func NewGetPlanListPageDataUseCase(
	repositories GetPlanListPageDataRepositories,
	services GetPlanListPageDataServices,
) *GetPlanListPageDataUseCase {
	return &GetPlanListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get plan list page data operation
func (uc *GetPlanListPageDataUseCase) Execute(ctx context.Context, req *planpb.GetPlanListPageDataRequest) (*planpb.GetPlanListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityPlan, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan.validation.request_required", "Request is required for plan list page data"))
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes plan list page data retrieval within a transaction
func (uc *GetPlanListPageDataUseCase) executeWithTransaction(ctx context.Context, req *planpb.GetPlanListPageDataRequest) (*planpb.GetPlanListPageDataResponse, error) {
	var result *planpb.GetPlanListPageDataResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "plan.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load plan list")
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

// executeCore contains the core business logic for getting plan list page data
func (uc *GetPlanListPageDataUseCase) executeCore(ctx context.Context, req *planpb.GetPlanListPageDataRequest) (*planpb.GetPlanListPageDataResponse, error) {
	// Delegate to repository
	return uc.repositories.Plan.GetPlanListPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetPlanListPageDataUseCase) validateInput(ctx context.Context, req *planpb.GetPlanListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	// Validate pagination parameters
	if req.Pagination != nil {
		if req.Pagination.Limit < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan.validation.invalid_limit", "Pagination limit must be non-negative"))
		}
		if req.Pagination.Limit > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan.validation.limit_too_large", "Pagination limit cannot exceed 1000"))
		}
	}

	// Validate filter parameters for subscription plan data
	if req.Filters != nil {
		for _, filter := range req.Filters.Filters {
			if filter.Field == "active" {
				// Validate active status filter to ensure boolean logic
				if filter.GetBooleanFilter() != nil {
					// Additional validation can be added here if needed
				}
			}
		}
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting plan list page data
func (uc *GetPlanListPageDataUseCase) validateBusinessRules(ctx context.Context, req *planpb.GetPlanListPageDataRequest) error {
	// Subscription security: Ensure proper access control for plan list data
	// Additional authorization checks would be implemented here in a real system
	// For example, certain plans might be restricted based on user roles or subscription tiers

	// Business rule: Apply data filtering based on user permissions
	// This would typically filter results based on user role and permissions
	// For example, only show active plans to non-admin users

	// Business rule: Validate search and filter parameters for security
	if req.Search != nil && req.Search.Query != "" {
		// Prevent SQL injection and other malicious queries
		// In a real system, implement proper query sanitization
	}

	// Business rule: Ensure plan data integrity
	// Validate that only appropriate users can access plan information
	// This is critical for subscription management systems where plan details might be confidential

	return nil
}
