package cost_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
)

type ListCostPlansRepositories struct {
	CostPlan costplanpb.CostPlanDomainServiceServer
}

type ListCostPlansServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

type ListCostPlansUseCase struct {
	repositories ListCostPlansRepositories
	services     ListCostPlansServices
}

func NewListCostPlansUseCase(
	repositories ListCostPlansRepositories,
	services ListCostPlansServices,
) *ListCostPlansUseCase {
	return &ListCostPlansUseCase{repositories: repositories, services: services}
}

func (uc *ListCostPlansUseCase) Execute(ctx context.Context, req *costplanpb.ListCostPlansRequest) (*costplanpb.ListCostPlansResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityCostPlan, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "cost_plan.validation.request_required", "request is required"))
	}
	result, err := uc.repositories.CostPlan.ListCostPlans(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "cost_plan.errors.list_failed", "cost plan listing failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}
