package price_schedule

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
)

// FindApplicablePriceScheduleRepositories groups all repository dependencies
type FindApplicablePriceScheduleRepositories struct {
	PriceSchedule priceschedulepb.PriceScheduleDomainServiceServer
}

// FindApplicablePriceScheduleServices groups all business service dependencies
type FindApplicablePriceScheduleServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// FindApplicablePriceScheduleUseCase handles the business logic for finding the
// applicable price schedule for a given location and date.
type FindApplicablePriceScheduleUseCase struct {
	repositories FindApplicablePriceScheduleRepositories
	services     FindApplicablePriceScheduleServices
}

// NewFindApplicablePriceScheduleUseCase creates a new FindApplicablePriceScheduleUseCase
func NewFindApplicablePriceScheduleUseCase(
	repositories FindApplicablePriceScheduleRepositories,
	services FindApplicablePriceScheduleServices,
) *FindApplicablePriceScheduleUseCase {
	return &FindApplicablePriceScheduleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute finds the active price schedule applicable to the given location and date.
func (uc *FindApplicablePriceScheduleUseCase) Execute(
	ctx context.Context,
	req *priceschedulepb.FindApplicablePriceScheduleRequest,
) (*priceschedulepb.FindApplicablePriceScheduleResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityPriceSchedule, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	return uc.executeCore(ctx, req)
}

// executeWithTransaction wraps the core logic in a transaction.
func (uc *FindApplicablePriceScheduleUseCase) executeWithTransaction(
	ctx context.Context,
	req *priceschedulepb.FindApplicablePriceScheduleRequest,
) (*priceschedulepb.FindApplicablePriceScheduleResponse, error) {
	var result *priceschedulepb.FindApplicablePriceScheduleResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.Translator,
				"price_schedule.errors.find_applicable_failed",
				"find applicable price schedule failed: %w",
			), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore delegates to the repository.
func (uc *FindApplicablePriceScheduleUseCase) executeCore(
	ctx context.Context,
	req *priceschedulepb.FindApplicablePriceScheduleRequest,
) (*priceschedulepb.FindApplicablePriceScheduleResponse, error) {
	resp, err := uc.repositories.PriceSchedule.FindApplicablePriceSchedule(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"price_schedule.errors.find_applicable_failed",
			"failed to find applicable price schedule: %w",
		), err)
	}
	return resp, nil
}

// validateInput validates the input request.
func (uc *FindApplicablePriceScheduleUseCase) validateInput(
	ctx context.Context,
	req *priceschedulepb.FindApplicablePriceScheduleRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"price_schedule.validation.request_required",
			"request is required",
		))
	}
	if req.LocationId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"price_schedule.validation.location_id_required",
			"location_id is required",
		))
	}
	if req.Date == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"price_schedule.validation.date_required",
			"date is required",
		))
	}
	return nil
}
