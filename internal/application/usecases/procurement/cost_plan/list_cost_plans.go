package cost_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
)

type ListCostPlansRepositories struct {
	CostPlan costplanpb.CostPlanDomainServiceServer
}

type ListCostPlansServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCostPlan, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_plan.validation.request_required", "request is required"))
	}
	result, err := uc.repositories.CostPlan.ListCostPlans(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_plan.errors.list_failed", "cost plan listing failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}
