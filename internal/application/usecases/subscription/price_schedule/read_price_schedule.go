package price_schedule

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
)

// ReadPriceScheduleRepositories groups all repository dependencies
type ReadPriceScheduleRepositories struct {
	PriceSchedule priceschedulepb.PriceScheduleDomainServiceServer // Primary entity repository
}

// ReadPriceScheduleServices groups all business service dependencies
type ReadPriceScheduleServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ReadPriceScheduleUseCase handles the business logic for reading price_schedules
type ReadPriceScheduleUseCase struct {
	repositories ReadPriceScheduleRepositories
	services     ReadPriceScheduleServices
}

// NewReadPriceScheduleUseCase creates use case with grouped dependencies
func NewReadPriceScheduleUseCase(
	repositories ReadPriceScheduleRepositories,
	services ReadPriceScheduleServices,
) *ReadPriceScheduleUseCase {
	return &ReadPriceScheduleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read price_schedule operation
func (uc *ReadPriceScheduleUseCase) Execute(ctx context.Context, req *priceschedulepb.ReadPriceScheduleRequest) (*priceschedulepb.ReadPriceScheduleResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPriceSchedule, ports.ActionRead); err != nil {
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

	// Call repository
	result, err := uc.repositories.PriceSchedule.ReadPriceSchedule(ctx, req)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// validateInput validates the input request
func (uc *ReadPriceScheduleUseCase) validateInput(ctx context.Context, req *priceschedulepb.ReadPriceScheduleRequest) error {
	if req == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_schedule.validation.request_required", "request is required")
		return errors.New(msg)
	}
	if req.Data == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_schedule.validation.data_required", "price schedule data is required")
		return errors.New(msg)
	}
	if req.Data.Id == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_schedule.validation.id_required", "price schedule ID is required")
		return errors.New(msg)
	}
	return nil
}

// validateBusinessRules enforces business constraints for price_schedule reading
func (uc *ReadPriceScheduleUseCase) validateBusinessRules(ctx context.Context, req *priceschedulepb.ReadPriceScheduleRequest) error {
	// Validate price schedule ID format
	if req.Data != nil && len(req.Data.Id) < 3 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_schedule.validation.id_min_length", "price schedule ID must be at least 3 characters long")
		return errors.New(msg)
	}

	return nil
}
