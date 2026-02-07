package plan

import (
	"context"
	"errors"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	planpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan"
)

// DeletePlanRepositories groups all repository dependencies
type DeletePlanRepositories struct {
	Plan planpb.PlanDomainServiceServer // Primary entity repository
}

// DeletePlanServices groups all business service dependencies
type DeletePlanServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// DeletePlanUseCase handles the business logic for deleting plans
type DeletePlanUseCase struct {
	repositories DeletePlanRepositories
	services     DeletePlanServices
}

// NewDeletePlanUseCase creates a new DeletePlanUseCase
func NewDeletePlanUseCase(
	repositories DeletePlanRepositories,
	services DeletePlanServices,
) *DeletePlanUseCase {
	return &DeletePlanUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete plan operation
func (uc *DeletePlanUseCase) Execute(ctx context.Context, req *planpb.DeletePlanRequest) (*planpb.DeletePlanResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	result, err := uc.repositories.Plan.DeletePlan(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.errors.deletion_failed", "plan deletion failed [DEFAULT]"))
	}

	return result, nil
}

// validateInput validates the input request
func (uc *DeletePlanUseCase) validateInput(ctx context.Context, req *planpb.DeletePlanRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.request_required", "request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.data_required", "plan data is required"))
	}
	if req.Data.Id == nil || *req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.id_required", "plan ID is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for plan deletion
func (uc *DeletePlanUseCase) validateBusinessRules(ctx context.Context, req *planpb.DeletePlanRequest) error {
	// Validate plan ID format
	if req.Data.Id == nil || len(*req.Data.Id) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.id_too_short", "plan ID must be at least 3 characters long"))
	}

	// Additional business rules for deletion can be added here
	// For example, preventing deletion of plans with active subscriptions

	return nil
}
