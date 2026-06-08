package plan_attribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	planattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_attribute"
)

// ListPlanAttributesRepositories groups all repository dependencies
type ListPlanAttributesRepositories struct {
	PlanAttribute planattributepb.PlanAttributeDomainServiceServer // Primary entity repository
}

// ListPlanAttributesServices groups all business service dependencies
type ListPlanAttributesServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
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
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.PlanAttribute, entityid.ActionList); err != nil {
		return nil, err
	}

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
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
