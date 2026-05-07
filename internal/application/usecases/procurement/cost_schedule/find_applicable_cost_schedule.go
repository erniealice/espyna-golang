package cost_schedule

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	costschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_schedule"
)

type FindApplicableCostScheduleRepositories struct {
	CostSchedule costschedulepb.CostScheduleDomainServiceServer
}

type FindApplicableCostScheduleServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type FindApplicableCostScheduleUseCase struct {
	repositories FindApplicableCostScheduleRepositories
	services     FindApplicableCostScheduleServices
}

func NewFindApplicableCostScheduleUseCase(
	repositories FindApplicableCostScheduleRepositories,
	services FindApplicableCostScheduleServices,
) *FindApplicableCostScheduleUseCase {
	return &FindApplicableCostScheduleUseCase{repositories: repositories, services: services}
}

func (uc *FindApplicableCostScheduleUseCase) Execute(ctx context.Context, req *costschedulepb.FindApplicableCostScheduleRequest) (*costschedulepb.FindApplicableCostScheduleResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCostSchedule, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_schedule.validation.request_required", "request is required"))
	}
	if req.LocationId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_schedule.validation.location_id_required", "location_id is required"))
	}
	if req.Date == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_schedule.validation.date_required", "date is required"))
	}
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *costschedulepb.FindApplicableCostScheduleResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.CostSchedule.FindApplicableCostSchedule(txCtx, req)
			if err != nil {
				return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "cost_schedule.errors.find_applicable_failed", "find applicable cost schedule failed: %w"), err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	resp, err := uc.repositories.CostSchedule.FindApplicableCostSchedule(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_schedule.errors.find_applicable_failed", "failed to find applicable cost schedule: %w"), err)
	}
	return resp, nil
}
