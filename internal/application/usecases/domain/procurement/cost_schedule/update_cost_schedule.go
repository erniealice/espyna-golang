package cost_schedule

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	costschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_schedule"
)

type UpdateCostScheduleRepositories struct {
	CostSchedule costschedulepb.CostScheduleDomainServiceServer
}

type UpdateCostScheduleServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

type UpdateCostScheduleUseCase struct {
	repositories UpdateCostScheduleRepositories
	services     UpdateCostScheduleServices
}

func NewUpdateCostScheduleUseCase(
	repositories UpdateCostScheduleRepositories,
	services UpdateCostScheduleServices,
) *UpdateCostScheduleUseCase {
	return &UpdateCostScheduleUseCase{repositories: repositories, services: services}
}

func (uc *UpdateCostScheduleUseCase) Execute(ctx context.Context, req *costschedulepb.UpdateCostScheduleRequest) (*costschedulepb.UpdateCostScheduleResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.CostSchedule, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "cost_schedule.validation.id_required", "cost schedule ID is required"))
	}
	if req.Data.Name == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "cost_schedule.validation.name_required", "cost schedule name is required"))
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.repositories.CostSchedule.UpdateCostSchedule(ctx, req)
}
