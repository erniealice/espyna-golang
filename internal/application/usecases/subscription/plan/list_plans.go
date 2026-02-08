package plan

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
)

// ListPlansRepositories groups all repository dependencies
type ListPlansRepositories struct {
	Plan planpb.PlanDomainServiceServer // Primary entity repository
}

// ListPlansServices groups all business service dependencies
type ListPlansServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ListPlansUseCase handles the business logic for listing plans
type ListPlansUseCase struct {
	repositories ListPlansRepositories
	services     ListPlansServices
}

// NewListPlansUseCase creates a new ListPlansUseCase
func NewListPlansUseCase(
	repositories ListPlansRepositories,
	services ListPlansServices,
) *ListPlansUseCase {
	return &ListPlansUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list plans operation
func (uc *ListPlansUseCase) Execute(ctx context.Context, req *planpb.ListPlansRequest) (*planpb.ListPlansResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	result, err := uc.repositories.Plan.ListPlans(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.errors.list_failed", "plan listing failed [DEFAULT]"))
	}

	return result, nil
}

// validateInput validates the input request
func (uc *ListPlansUseCase) validateInput(ctx context.Context, req *planpb.ListPlansRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan.validation.request_required", "request is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for plan listing
func (uc *ListPlansUseCase) validateBusinessRules(ctx context.Context, req *planpb.ListPlansRequest) error {
	// No specific business rules for listing plans
	return nil
}
