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

type GetCostScheduleItemPageDataRepositories struct {
	CostSchedule costschedulepb.CostScheduleDomainServiceServer
}

type GetCostScheduleItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type GetCostScheduleItemPageDataUseCase struct {
	repositories GetCostScheduleItemPageDataRepositories
	services     GetCostScheduleItemPageDataServices
}

func NewGetCostScheduleItemPageDataUseCase(
	repositories GetCostScheduleItemPageDataRepositories,
	services GetCostScheduleItemPageDataServices,
) *GetCostScheduleItemPageDataUseCase {
	return &GetCostScheduleItemPageDataUseCase{repositories: repositories, services: services}
}

func (uc *GetCostScheduleItemPageDataUseCase) Execute(ctx context.Context, req *costschedulepb.GetCostScheduleItemPageDataRequest) (*costschedulepb.GetCostScheduleItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCostSchedule, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil || req.CostScheduleId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_schedule.validation.id_required", "cost schedule ID is required"))
	}
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *costschedulepb.GetCostScheduleItemPageDataResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.CostSchedule.GetCostScheduleItemPageData(txCtx, req)
			if err != nil {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "cost_schedule.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load cost schedule details: %w"), err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.repositories.CostSchedule.GetCostScheduleItemPageData(ctx, req)
}
