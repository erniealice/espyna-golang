package price_schedule

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
)

// CreatePriceScheduleRepositories groups all repository dependencies
type CreatePriceScheduleRepositories struct {
	PriceSchedule priceschedulepb.PriceScheduleDomainServiceServer // Primary entity repository
}

// CreatePriceScheduleServices groups all business service dependencies
type CreatePriceScheduleServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreatePriceScheduleUseCase handles the business logic for creating price_schedules
type CreatePriceScheduleUseCase struct {
	repositories CreatePriceScheduleRepositories
	services     CreatePriceScheduleServices
}

// NewCreatePriceScheduleUseCase creates use case with grouped dependencies
func NewCreatePriceScheduleUseCase(
	repositories CreatePriceScheduleRepositories,
	services CreatePriceScheduleServices,
) *CreatePriceScheduleUseCase {
	return &CreatePriceScheduleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create price_schedule operation
func (uc *CreatePriceScheduleUseCase) Execute(ctx context.Context, req *priceschedulepb.CreatePriceScheduleRequest) (*priceschedulepb.CreatePriceScheduleResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPriceSchedule, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// validateInput validates the input request
func (uc *CreatePriceScheduleUseCase) validateInput(ctx context.Context, req *priceschedulepb.CreatePriceScheduleRequest) error {
	if req == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_schedule.validation.request_required", "request is required")
		return errors.New(msg)
	}
	if req.Data == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_schedule.validation.data_required", "price schedule data is required")
		return errors.New(msg)
	}
	if req.Data.Name == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_schedule.validation.name_required", "price schedule name is required")
		return errors.New(msg)
	}
	if req.Data.GetDateTimeStart() == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_schedule.validation.date_time_start_required", "date time start is required")
		return errors.New(msg)
	}
	return nil
}

// enrichPriceScheduleData adds generated fields and audit information
func (uc *CreatePriceScheduleUseCase) enrichPriceScheduleData(priceSchedule *priceschedulepb.PriceSchedule) error {
	now := time.Now()

	// Generate PriceSchedule ID if not provided
	if priceSchedule.Id == "" {
		priceSchedule.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	priceSchedule.DateCreated = &[]int64{now.UnixMilli()}[0]
	priceSchedule.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	priceSchedule.DateModified = &[]int64{now.UnixMilli()}[0]
	priceSchedule.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	priceSchedule.Active = true

	return nil
}

// validateBusinessRules enforces business constraints for price schedules
func (uc *CreatePriceScheduleUseCase) validateBusinessRules(ctx context.Context, priceSchedule *priceschedulepb.PriceSchedule) error {
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

// executeWithTransaction executes price schedule creation within a transaction
func (uc *CreatePriceScheduleUseCase) executeWithTransaction(ctx context.Context, req *priceschedulepb.CreatePriceScheduleRequest) (*priceschedulepb.CreatePriceScheduleResponse, error) {
	var result *priceschedulepb.CreatePriceScheduleResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_schedule.errors.creation_failed", "price schedule creation failed")
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

// executeCore contains the core business logic (moved from original Execute method)
func (uc *CreatePriceScheduleUseCase) executeCore(ctx context.Context, req *priceschedulepb.CreatePriceScheduleRequest) (*priceschedulepb.CreatePriceScheduleResponse, error) {
	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	if err := uc.enrichPriceScheduleData(req.Data); err != nil {
		return nil, err
	}

	// Delegate to repository
	return uc.repositories.PriceSchedule.CreatePriceSchedule(ctx, &priceschedulepb.CreatePriceScheduleRequest{
		Data: req.Data,
	})
}
