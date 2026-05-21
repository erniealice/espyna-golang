package cost_schedule

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	costschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_schedule"
)

type ListCostSchedulesRepositories struct {
	CostSchedule costschedulepb.CostScheduleDomainServiceServer
}

type ListCostSchedulesServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

type ListCostSchedulesUseCase struct {
	repositories ListCostSchedulesRepositories
	services     ListCostSchedulesServices
}

func NewListCostSchedulesUseCase(
	repositories ListCostSchedulesRepositories,
	services ListCostSchedulesServices,
) *ListCostSchedulesUseCase {
	return &ListCostSchedulesUseCase{repositories: repositories, services: services}
}

func (uc *ListCostSchedulesUseCase) Execute(ctx context.Context, req *costschedulepb.ListCostSchedulesRequest) (*costschedulepb.ListCostSchedulesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityCostSchedule, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "cost_schedule.validation.request_required", "request is required"))
	}
	result, err := uc.repositories.CostSchedule.ListCostSchedules(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "cost_schedule.errors.list_failed", "cost schedule listing failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}
