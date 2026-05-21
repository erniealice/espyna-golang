package price_schedule

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
)

// UpdatePriceScheduleRepositories groups all repository dependencies
type UpdatePriceScheduleRepositories struct {
	PriceSchedule priceschedulepb.PriceScheduleDomainServiceServer // Primary entity repository
}

// UpdatePriceScheduleServices groups all business service dependencies
type UpdatePriceScheduleServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// UpdatePriceScheduleUseCase handles the business logic for updating price_schedules
type UpdatePriceScheduleUseCase struct {
	repositories UpdatePriceScheduleRepositories
	services     UpdatePriceScheduleServices
}

// NewUpdatePriceScheduleUseCase creates use case with grouped dependencies
func NewUpdatePriceScheduleUseCase(
	repositories UpdatePriceScheduleRepositories,
	services UpdatePriceScheduleServices,
) *UpdatePriceScheduleUseCase {
	return &UpdatePriceScheduleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update price_schedule operation
func (uc *UpdatePriceScheduleUseCase) Execute(ctx context.Context, req *priceschedulepb.UpdatePriceScheduleRequest) (*priceschedulepb.UpdatePriceScheduleResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPriceSchedule, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichPriceScheduleData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	result, err := uc.repositories.PriceSchedule.UpdatePriceSchedule(ctx, req)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// validateInput validates the input request
func (uc *UpdatePriceScheduleUseCase) validateInput(ctx context.Context, req *priceschedulepb.UpdatePriceScheduleRequest) error {
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
	if req.Data.Name == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_schedule.validation.name_required", "price schedule name is required")
		return errors.New(msg)
	}
	return nil
}

// enrichPriceScheduleData adds generated fields and audit information
func (uc *UpdatePriceScheduleUseCase) enrichPriceScheduleData(priceSchedule *priceschedulepb.PriceSchedule) error {
	now := time.Now()

	// Update audit fields
	priceSchedule.DateModified = &[]int64{now.UnixMilli()}[0]
	priceSchedule.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints for price schedules
func (uc *UpdatePriceScheduleUseCase) validateBusinessRules(ctx context.Context, priceSchedule *priceschedulepb.PriceSchedule) error {
	// Validate price schedule ID length
	if len(priceSchedule.Id) < 3 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_schedule.validation.id_min_length", "price schedule ID must be at least 3 characters long")
		return errors.New(msg)
	}

	// Validate price schedule name length
	if len(priceSchedule.Name) < 3 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_schedule.validation.name_min_length", "price schedule name must be at least 3 characters long")
		return errors.New(msg)
	}

	if len(priceSchedule.Name) > 100 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_schedule.validation.name_max_length", "price schedule name cannot exceed 100 characters")
		return errors.New(msg)
	}

	// Validate Description length validation
	if priceSchedule.Description != nil && len(*priceSchedule.Description) > 500 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_schedule.validation.description_max_length", "price schedule description cannot exceed 500 characters")
		return errors.New(msg)
	}

	return nil
}
