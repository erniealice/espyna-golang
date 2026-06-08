package price_schedule

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
)

// GetPriceScheduleItemPageDataRepositories groups all repository dependencies
type GetPriceScheduleItemPageDataRepositories struct {
	PriceSchedule priceschedulepb.PriceScheduleDomainServiceServer // Primary entity repository
}

// GetPriceScheduleItemPageDataServices groups all business service dependencies
type GetPriceScheduleItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetPriceScheduleItemPageDataUseCase handles the business logic for getting price schedule item page data
type GetPriceScheduleItemPageDataUseCase struct {
	repositories GetPriceScheduleItemPageDataRepositories
	services     GetPriceScheduleItemPageDataServices
}

// NewGetPriceScheduleItemPageDataUseCase creates use case with grouped dependencies
func NewGetPriceScheduleItemPageDataUseCase(
	repositories GetPriceScheduleItemPageDataRepositories,
	services GetPriceScheduleItemPageDataServices,
) *GetPriceScheduleItemPageDataUseCase {
	return &GetPriceScheduleItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get price schedule item page data operation
func (uc *GetPriceScheduleItemPageDataUseCase) Execute(ctx context.Context, req *priceschedulepb.GetPriceScheduleItemPageDataRequest) (*priceschedulepb.GetPriceScheduleItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.PriceSchedule, entityid.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_schedule.validation.request_required", "Request is required for price schedule item page data"))
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes price schedule item page data retrieval within a transaction
func (uc *GetPriceScheduleItemPageDataUseCase) executeWithTransaction(ctx context.Context, req *priceschedulepb.GetPriceScheduleItemPageDataRequest) (*priceschedulepb.GetPriceScheduleItemPageDataResponse, error) {
	var result *priceschedulepb.GetPriceScheduleItemPageDataResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "price_schedule.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load price schedule details")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting price schedule item page data
func (uc *GetPriceScheduleItemPageDataUseCase) executeCore(ctx context.Context, req *priceschedulepb.GetPriceScheduleItemPageDataRequest) (*priceschedulepb.GetPriceScheduleItemPageDataResponse, error) {
	// Delegate to repository
	return uc.repositories.PriceSchedule.GetPriceScheduleItemPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetPriceScheduleItemPageDataUseCase) validateInput(ctx context.Context, req *priceschedulepb.GetPriceScheduleItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_schedule.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	if req.PriceScheduleId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_schedule.validation.id_required", "Price schedule ID is required"))
	}

	// Validate ID format (basic validation)
	if len(req.PriceScheduleId) > 255 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_schedule.validation.id_too_long", "Price schedule ID cannot exceed 255 characters"))
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting price schedule item page data
func (uc *GetPriceScheduleItemPageDataUseCase) validateBusinessRules(ctx context.Context, req *priceschedulepb.GetPriceScheduleItemPageDataRequest) error {
	// Business rule: Validate price schedule access permissions
	// This would typically check if the current user has permission to view this specific price schedule

	return nil
}
