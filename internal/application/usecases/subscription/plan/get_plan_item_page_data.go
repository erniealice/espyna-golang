package plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
)

// GetPlanItemPageDataRepositories groups all repository dependencies
type GetPlanItemPageDataRepositories struct {
	Plan planpb.PlanDomainServiceServer // Primary entity repository
}

// GetPlanItemPageDataServices groups all business service dependencies
type GetPlanItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// GetPlanItemPageDataUseCase handles the business logic for getting plan item page data
type GetPlanItemPageDataUseCase struct {
	repositories GetPlanItemPageDataRepositories
	services     GetPlanItemPageDataServices
}

// NewGetPlanItemPageDataUseCase creates use case with grouped dependencies
func NewGetPlanItemPageDataUseCase(
	repositories GetPlanItemPageDataRepositories,
	services GetPlanItemPageDataServices,
) *GetPlanItemPageDataUseCase {
	return &GetPlanItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get plan item page data operation
func (uc *GetPlanItemPageDataUseCase) Execute(ctx context.Context, req *planpb.GetPlanItemPageDataRequest) (*planpb.GetPlanItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPlan, ports.ActionList); err != nil {
		return nil, err
	}


	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.request_required", "Request is required for plan item page data"))
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

// executeWithTransaction executes plan item page data retrieval within a transaction
func (uc *GetPlanItemPageDataUseCase) executeWithTransaction(ctx context.Context, req *planpb.GetPlanItemPageDataRequest) (*planpb.GetPlanItemPageDataResponse, error) {
	var result *planpb.GetPlanItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "plan.errors.get_item_page_data_failed", "")
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

// executeCore contains the core business logic for getting plan item page data
func (uc *GetPlanItemPageDataUseCase) executeCore(ctx context.Context, req *planpb.GetPlanItemPageDataRequest) (*planpb.GetPlanItemPageDataResponse, error) {
	// Delegate to repository
	return uc.repositories.Plan.GetPlanItemPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetPlanItemPageDataUseCase) validateInput(ctx context.Context, req *planpb.GetPlanItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.request_required", ""))
	}

	if req.PlanId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.id_required", "Plan ID is required"))
	}

	// Validate ID format (basic validation)
	if len(req.PlanId) > 255 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.id_too_long", "Plan ID cannot exceed 255 characters"))
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting plan item page data
func (uc *GetPlanItemPageDataUseCase) validateBusinessRules(ctx context.Context, req *planpb.GetPlanItemPageDataRequest) error {
	// Subscription security: Ensure proper access control for plan item data
	// Additional authorization checks would be implemented here in a real system
	// For example, certain plans might be restricted based on user roles or subscription tiers

	// Business rule: Validate plan access permissions
	// This would typically check if the current user has permission to view this specific plan
	// In a real system, this might involve checking subscription tiers or admin privileges

	// Business rule: Ensure plan data integrity
	// Validate that only appropriate users can access detailed plan information
	// This is critical for subscription management systems where plan details might be confidential

	return nil
}
