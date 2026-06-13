package plan

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
)

// ListPlansRepositories groups all repository dependencies
type ListPlansRepositories struct {
	Plan planpb.PlanDomainServiceServer // Primary entity repository
}

// ListPlansServices groups all business service dependencies
type ListPlansServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Plan,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

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
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan.errors.list_failed", "plan listing failed [DEFAULT]"))
	}

	return result, nil
}

// validateInput validates the input request
func (uc *ListPlansUseCase) validateInput(ctx context.Context, req *planpb.ListPlansRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan.validation.request_required", "request is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for plan listing
func (uc *ListPlansUseCase) validateBusinessRules(ctx context.Context, req *planpb.ListPlansRequest) error {
	// No specific business rules for listing plans
	return nil
}
