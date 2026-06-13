package plan_attribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	planattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_attribute"
)

// GetPlanAttributeListPageDataRepositories groups all repository dependencies
type GetPlanAttributeListPageDataRepositories struct {
	PlanAttribute planattributepb.PlanAttributeDomainServiceServer // Primary entity repository
}

// GetPlanAttributeListPageDataServices groups all business service dependencies
type GetPlanAttributeListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetPlanAttributeListPageDataUseCase handles the business logic for getting plan attribute list page data
type GetPlanAttributeListPageDataUseCase struct {
	repositories GetPlanAttributeListPageDataRepositories
	services     GetPlanAttributeListPageDataServices
}

// NewGetPlanAttributeListPageDataUseCase creates a new GetPlanAttributeListPageDataUseCase
func NewGetPlanAttributeListPageDataUseCase(
	repositories GetPlanAttributeListPageDataRepositories,
	services GetPlanAttributeListPageDataServices,
) *GetPlanAttributeListPageDataUseCase {
	return &GetPlanAttributeListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get plan attribute list page data operation
func (uc *GetPlanAttributeListPageDataUseCase) Execute(ctx context.Context, req *planattributepb.GetPlanAttributeListPageDataRequest) (*planattributepb.GetPlanAttributeListPageDataResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.PlanAttribute,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.PlanAttribute.GetPlanAttributeListPageData(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetPlanAttributeListPageDataUseCase) validateInput(ctx context.Context, req *planattributepb.GetPlanAttributeListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
