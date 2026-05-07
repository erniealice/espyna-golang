package cost_schedule

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	costschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_schedule"
)

type UpdateCostScheduleRepositories struct {
	CostSchedule costschedulepb.CostScheduleDomainServiceServer
}

type UpdateCostScheduleServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCostSchedule, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_schedule.validation.id_required", "cost schedule ID is required"))
	}
	if req.Data.Name == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_schedule.validation.name_required", "cost schedule name is required"))
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.repositories.CostSchedule.UpdateCostSchedule(ctx, req)
}
