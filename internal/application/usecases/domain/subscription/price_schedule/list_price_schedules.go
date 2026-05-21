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

// ListPriceSchedulesRepositories groups all repository dependencies
type ListPriceSchedulesRepositories struct {
	PriceSchedule priceschedulepb.PriceScheduleDomainServiceServer // Primary entity repository
}

// ListPriceSchedulesServices groups all business service dependencies
type ListPriceSchedulesServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
}

// ListPriceSchedulesUseCase handles the business logic for listing price_schedules
type ListPriceSchedulesUseCase struct {
	repositories ListPriceSchedulesRepositories
	services     ListPriceSchedulesServices
}

// NewListPriceSchedulesUseCase creates a new ListPriceSchedulesUseCase
func NewListPriceSchedulesUseCase(
	repositories ListPriceSchedulesRepositories,
	services ListPriceSchedulesServices,
) *ListPriceSchedulesUseCase {
	return &ListPriceSchedulesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list price_schedules operation
func (uc *ListPriceSchedulesUseCase) Execute(ctx context.Context, req *priceschedulepb.ListPriceSchedulesRequest) (*priceschedulepb.ListPriceSchedulesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityPriceSchedule, ports.ActionList); err != nil {
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
	result, err := uc.repositories.PriceSchedule.ListPriceSchedules(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_schedule.errors.list_failed", "price schedule listing failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}

// validateInput validates the input request
func (uc *ListPriceSchedulesUseCase) validateInput(ctx context.Context, req *priceschedulepb.ListPriceSchedulesRequest) error {
	if req == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_schedule.validation.request_required", "request is required")
		return errors.New(msg)
	}
	return nil
}

// validateBusinessRules enforces business constraints for price_schedule listing
func (uc *ListPriceSchedulesUseCase) validateBusinessRules(ctx context.Context, req *priceschedulepb.ListPriceSchedulesRequest) error {
	// No specific business rules for listing price schedules
	return nil
}
