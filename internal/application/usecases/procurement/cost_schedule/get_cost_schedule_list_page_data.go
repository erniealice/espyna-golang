package cost_schedule

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	costschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_schedule"
)

type GetCostScheduleListPageDataRepositories struct {
	CostSchedule costschedulepb.CostScheduleDomainServiceServer
}

type GetCostScheduleListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type GetCostScheduleListPageDataUseCase struct {
	repositories GetCostScheduleListPageDataRepositories
	services     GetCostScheduleListPageDataServices
}

func NewGetCostScheduleListPageDataUseCase(
	repositories GetCostScheduleListPageDataRepositories,
	services GetCostScheduleListPageDataServices,
) *GetCostScheduleListPageDataUseCase {
	return &GetCostScheduleListPageDataUseCase{repositories: repositories, services: services}
}

func (uc *GetCostScheduleListPageDataUseCase) Execute(ctx context.Context, req *costschedulepb.GetCostScheduleListPageDataRequest) (*costschedulepb.GetCostScheduleListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCostSchedule, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_schedule.validation.request_required", "request is required"))
	}
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *costschedulepb.GetCostScheduleListPageDataResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.CostSchedule.GetCostScheduleListPageData(txCtx, req)
			if err != nil {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "cost_schedule.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load cost schedule list: %w"), err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.repositories.CostSchedule.GetCostScheduleListPageData(ctx, req)
}
