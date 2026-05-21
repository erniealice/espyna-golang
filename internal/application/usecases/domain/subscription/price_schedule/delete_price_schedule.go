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

// DeletePriceScheduleRepositories groups all repository dependencies
type DeletePriceScheduleRepositories struct {
	PriceSchedule priceschedulepb.PriceScheduleDomainServiceServer // Primary entity repository
}

// DeletePriceScheduleServices groups all business service dependencies
type DeletePriceScheduleServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
}

// DeletePriceScheduleUseCase handles the business logic for deleting price_schedules
type DeletePriceScheduleUseCase struct {
	repositories DeletePriceScheduleRepositories
	services     DeletePriceScheduleServices
}

// NewDeletePriceScheduleUseCase creates use case with grouped dependencies
func NewDeletePriceScheduleUseCase(
	repositories DeletePriceScheduleRepositories,
	services DeletePriceScheduleServices,
) *DeletePriceScheduleUseCase {
	return &DeletePriceScheduleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete price_schedule operation
func (uc *DeletePriceScheduleUseCase) Execute(ctx context.Context, req *priceschedulepb.DeletePriceScheduleRequest) (*priceschedulepb.DeletePriceScheduleResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityPriceSchedule, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Call repository with error wrapping
	result, err := uc.repositories.PriceSchedule.DeletePriceSchedule(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_schedule.errors.deletion_failed", "price schedule deletion failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}

// validateInput validates the input request
func (uc *DeletePriceScheduleUseCase) validateInput(ctx context.Context, req *priceschedulepb.DeletePriceScheduleRequest) error {
	if req == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_schedule.validation.request_required", "request is required")
		return errors.New(msg)
	}
	if req.Data == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_schedule.validation.data_required", "price schedule data is required")
		return errors.New(msg)
	}
	if req.Data.Id == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_schedule.validation.id_required", "price schedule ID is required")
		return errors.New(msg)
	}
	return nil
}

// validateBusinessRules enforces business constraints for price_schedule deletion
func (uc *DeletePriceScheduleUseCase) validateBusinessRules(ctx context.Context, req *priceschedulepb.DeletePriceScheduleRequest) error {
	// Validate price schedule ID format
	if req.Data != nil && len(req.Data.Id) < 3 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_schedule.validation.id_min_length", "price schedule ID must be at least 3 characters long")
		return errors.New(msg)
	}

	// Additional business rules for deletion can be added here
	// For example, preventing deletion of price schedules with active items

	return nil
}
