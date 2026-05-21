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

// GetPriceScheduleListPageDataRepositories groups all repository dependencies
type GetPriceScheduleListPageDataRepositories struct {
	PriceSchedule priceschedulepb.PriceScheduleDomainServiceServer // Primary entity repository
}

// GetPriceScheduleListPageDataServices groups all business service dependencies
type GetPriceScheduleListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetPriceScheduleListPageDataUseCase handles the business logic for getting price schedule list page data
type GetPriceScheduleListPageDataUseCase struct {
	repositories GetPriceScheduleListPageDataRepositories
	services     GetPriceScheduleListPageDataServices
}

// NewGetPriceScheduleListPageDataUseCase creates use case with grouped dependencies
func NewGetPriceScheduleListPageDataUseCase(
	repositories GetPriceScheduleListPageDataRepositories,
	services GetPriceScheduleListPageDataServices,
) *GetPriceScheduleListPageDataUseCase {
	return &GetPriceScheduleListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get price schedule list page data operation
func (uc *GetPriceScheduleListPageDataUseCase) Execute(ctx context.Context, req *priceschedulepb.GetPriceScheduleListPageDataRequest) (*priceschedulepb.GetPriceScheduleListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPriceSchedule, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_schedule.validation.request_required", "Request is required for price schedule list page data"))
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes price schedule list page data retrieval within a transaction
func (uc *GetPriceScheduleListPageDataUseCase) executeWithTransaction(ctx context.Context, req *priceschedulepb.GetPriceScheduleListPageDataRequest) (*priceschedulepb.GetPriceScheduleListPageDataResponse, error) {
	var result *priceschedulepb.GetPriceScheduleListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "price_schedule.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load price schedule list")
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

// executeCore contains the core business logic for getting price schedule list page data
func (uc *GetPriceScheduleListPageDataUseCase) executeCore(ctx context.Context, req *priceschedulepb.GetPriceScheduleListPageDataRequest) (*priceschedulepb.GetPriceScheduleListPageDataResponse, error) {
	// Delegate to repository
	return uc.repositories.PriceSchedule.GetPriceScheduleListPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetPriceScheduleListPageDataUseCase) validateInput(ctx context.Context, req *priceschedulepb.GetPriceScheduleListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_schedule.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	// Validate pagination parameters
	if req.Pagination != nil {
		if req.Pagination.Limit < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_schedule.validation.invalid_limit", "Pagination limit must be non-negative"))
		}
		if req.Pagination.Limit > 1000 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_schedule.validation.limit_too_large", "Pagination limit cannot exceed 1000"))
		}
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting price schedule list page data
func (uc *GetPriceScheduleListPageDataUseCase) validateBusinessRules(ctx context.Context, req *priceschedulepb.GetPriceScheduleListPageDataRequest) error {
	// Business rule: Validate search and filter parameters for security
	if req.Search != nil && req.Search.Query != "" {
		// Prevent SQL injection and other malicious queries
		// In a real system, implement proper query sanitization
	}

	return nil
}
