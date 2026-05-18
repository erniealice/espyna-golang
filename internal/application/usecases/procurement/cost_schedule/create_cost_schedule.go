package cost_schedule

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	costschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_schedule"
)

// CreateCostScheduleRepositories groups all repository dependencies
type CreateCostScheduleRepositories struct {
	CostSchedule costschedulepb.CostScheduleDomainServiceServer
}

// CreateCostScheduleServices groups all business service dependencies
type CreateCostScheduleServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateCostScheduleUseCase handles the business logic for creating cost_schedules
type CreateCostScheduleUseCase struct {
	repositories CreateCostScheduleRepositories
	services     CreateCostScheduleServices
}

// NewCreateCostScheduleUseCase creates use case with grouped dependencies
func NewCreateCostScheduleUseCase(
	repositories CreateCostScheduleRepositories,
	services CreateCostScheduleServices,
) *CreateCostScheduleUseCase {
	return &CreateCostScheduleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create cost_schedule operation
func (uc *CreateCostScheduleUseCase) Execute(ctx context.Context, req *costschedulepb.CreateCostScheduleRequest) (*costschedulepb.CreateCostScheduleResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCostSchedule, ports.ActionCreate); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	return uc.executeCore(ctx, req)
}

func (uc *CreateCostScheduleUseCase) validateInput(ctx context.Context, req *costschedulepb.CreateCostScheduleRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_schedule.validation.request_required", "request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_schedule.validation.data_required", "cost schedule data is required"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_schedule.validation.name_required", "cost schedule name is required"))
	}
	if req.Data.GetDateTimeStart() == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_schedule.validation.date_time_start_required", "date time start is required"))
	}
	return nil
}

func (uc *CreateCostScheduleUseCase) enrichData(cs *costschedulepb.CostSchedule) {
	now := time.Now()
	if cs.Id == "" {
		cs.Id = uc.services.IDService.GenerateID()
	}
	cs.Active = true
	cs.DateCreated = &[]int64{now.UnixMilli()}[0]
	cs.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	cs.DateModified = &[]int64{now.UnixMilli()}[0]
	cs.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
}

func (uc *CreateCostScheduleUseCase) executeWithTransaction(ctx context.Context, req *costschedulepb.CreateCostScheduleRequest) (*costschedulepb.CreateCostScheduleResponse, error) {
	var result *costschedulepb.CreateCostScheduleResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_schedule.errors.creation_failed", "cost schedule creation failed")
			return fmt.Errorf("%s: %w", msg, err)
		}
		result = res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (uc *CreateCostScheduleUseCase) executeCore(ctx context.Context, req *costschedulepb.CreateCostScheduleRequest) (*costschedulepb.CreateCostScheduleResponse, error) {
	if len(req.Data.Name) < 3 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_schedule.validation.name_min_length", "cost schedule name must be at least 3 characters long"))
	}
	if len(req.Data.Name) > 100 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "cost_schedule.validation.name_max_length", "cost schedule name cannot exceed 100 characters"))
	}
	uc.enrichData(req.Data)
	return uc.repositories.CostSchedule.CreateCostSchedule(ctx, req)
}
