package plan_attribute

import (
	"context"
	"errors"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	planattributepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan_attribute"
)

// ListPlanAttributesRepositories groups all repository dependencies
type ListPlanAttributesRepositories struct {
	PlanAttribute planattributepb.PlanAttributeDomainServiceServer // Primary entity repository
}

// ListPlanAttributesServices groups all business service dependencies
type ListPlanAttributesServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// ListPlanAttributesUseCase handles the business logic for listing plan attributes
type ListPlanAttributesUseCase struct {
	repositories ListPlanAttributesRepositories
	services     ListPlanAttributesServices
}

// NewListPlanAttributesUseCase creates a new ListPlanAttributesUseCase
func NewListPlanAttributesUseCase(
	repositories ListPlanAttributesRepositories,
	services ListPlanAttributesServices,
) *ListPlanAttributesUseCase {
	return &ListPlanAttributesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list plan attributes operation
func (uc *ListPlanAttributesUseCase) Execute(ctx context.Context, req *planattributepb.ListPlanAttributesRequest) (*planattributepb.ListPlanAttributesResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.PlanAttribute.ListPlanAttributes(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListPlanAttributesUseCase) validateInput(ctx context.Context, req *planattributepb.ListPlanAttributesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
