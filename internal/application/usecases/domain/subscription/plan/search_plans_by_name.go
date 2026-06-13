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

// SearchPlansByNameRepositories groups all repository dependencies
type SearchPlansByNameRepositories struct {
	Plan planpb.PlanDomainServiceServer
}

// SearchPlansByNameServices groups all business service dependencies
type SearchPlansByNameServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// SearchPlansByNameUseCase handles the business logic for searching plans by name
type SearchPlansByNameUseCase struct {
	repositories SearchPlansByNameRepositories
	services     SearchPlansByNameServices
}

// NewSearchPlansByNameUseCase creates use case with grouped dependencies
func NewSearchPlansByNameUseCase(
	repositories SearchPlansByNameRepositories,
	services SearchPlansByNameServices,
) *SearchPlansByNameUseCase {
	return &SearchPlansByNameUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the search plans by name operation
func (uc *SearchPlansByNameUseCase) Execute(ctx context.Context, req *planpb.SearchPlansByNameRequest) (*planpb.SearchPlansByNameResponse, error) {
	// Authorization check — search is a read/list operation
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Plan,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		req = &planpb.SearchPlansByNameRequest{}
	}

	result, err := uc.repositories.Plan.SearchPlansByName(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan.errors.search_failed", "plan search failed [DEFAULT]"))
	}

	return result, nil
}
