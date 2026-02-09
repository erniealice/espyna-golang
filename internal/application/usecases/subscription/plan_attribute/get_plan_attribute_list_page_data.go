package plan_attribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	planattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_attribute"
)

// GetPlanAttributeListPageDataRepositories groups all repository dependencies
type GetPlanAttributeListPageDataRepositories struct {
	PlanAttribute planattributepb.PlanAttributeDomainServiceServer // Primary entity repository
}

// GetPlanAttributeListPageDataServices groups all business service dependencies
type GetPlanAttributeListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPlanAttribute, ports.ActionList); err != nil {
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
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
