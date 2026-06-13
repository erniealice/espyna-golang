package cost_schedule

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	costschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_schedule"
)

type GetCostScheduleListPageDataRepositories struct {
	CostSchedule costschedulepb.CostScheduleDomainServiceServer
}

type GetCostScheduleListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.CostSchedule,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "cost_schedule.validation.request_required", "request is required"))
	}
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *costschedulepb.GetCostScheduleListPageDataResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.CostSchedule.GetCostScheduleListPageData(txCtx, req)
			if err != nil {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "cost_schedule.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load cost schedule list: %w"), err)
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
