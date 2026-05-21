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

type DeleteCostScheduleRepositories struct {
	CostSchedule costschedulepb.CostScheduleDomainServiceServer
}

type DeleteCostScheduleServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

type DeleteCostScheduleUseCase struct {
	repositories DeleteCostScheduleRepositories
	services     DeleteCostScheduleServices
}

func NewDeleteCostScheduleUseCase(
	repositories DeleteCostScheduleRepositories,
	services DeleteCostScheduleServices,
) *DeleteCostScheduleUseCase {
	return &DeleteCostScheduleUseCase{repositories: repositories, services: services}
}

func (uc *DeleteCostScheduleUseCase) Execute(ctx context.Context, req *costschedulepb.DeleteCostScheduleRequest) (*costschedulepb.DeleteCostScheduleResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityCostSchedule, ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "cost_schedule.validation.id_required", "cost schedule ID is required"))
	}
	result, err := uc.repositories.CostSchedule.DeleteCostSchedule(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "cost_schedule.errors.deletion_failed", "cost schedule deletion failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}
