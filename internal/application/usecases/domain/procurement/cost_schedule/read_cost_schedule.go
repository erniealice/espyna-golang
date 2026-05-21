package cost_schedule

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	costschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_schedule"
)

type ReadCostScheduleRepositories struct {
	CostSchedule costschedulepb.CostScheduleDomainServiceServer
}

type ReadCostScheduleServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

type ReadCostScheduleUseCase struct {
	repositories ReadCostScheduleRepositories
	services     ReadCostScheduleServices
}

func NewReadCostScheduleUseCase(
	repositories ReadCostScheduleRepositories,
	services ReadCostScheduleServices,
) *ReadCostScheduleUseCase {
	return &ReadCostScheduleUseCase{repositories: repositories, services: services}
}

func (uc *ReadCostScheduleUseCase) Execute(ctx context.Context, req *costschedulepb.ReadCostScheduleRequest) (*costschedulepb.ReadCostScheduleResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityCostSchedule, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "cost_schedule.validation.id_required", "cost schedule ID is required"))
	}
	return uc.repositories.CostSchedule.ReadCostSchedule(ctx, req)
}
