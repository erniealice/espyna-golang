package price_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
)

// ListPricePlansRepositories groups all repository dependencies
type ListPricePlansRepositories struct {
	PricePlan priceplanpb.PricePlanDomainServiceServer // Primary entity repository
}

// ListPricePlansServices groups all business service dependencies
type ListPricePlansServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ListPricePlansUseCase handles the business logic for listing price_plans
type ListPricePlansUseCase struct {
	repositories ListPricePlansRepositories
	services     ListPricePlansServices
}

// NewListPricePlansUseCase creates a new ListPricePlansUseCase
func NewListPricePlansUseCase(
	repositories ListPricePlansRepositories,
	services ListPricePlansServices,
) *ListPricePlansUseCase {
	return &ListPricePlansUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list price_plans operation
func (uc *ListPricePlansUseCase) Execute(ctx context.Context, req *priceplanpb.ListPricePlansRequest) (*priceplanpb.ListPricePlansResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Call repository with error wrapping
	result, err := uc.repositories.PricePlan.ListPricePlans(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.errors.list_failed", "price plan listing failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}

// validateInput validates the input request
func (uc *ListPricePlansUseCase) validateInput(ctx context.Context, req *priceplanpb.ListPricePlansRequest) error {
	if req == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.request_required", "request is required")
		return errors.New(msg)
	}
	return nil
}

// validateBusinessRules enforces business constraints for price_plan listing
func (uc *ListPricePlansUseCase) validateBusinessRules(ctx context.Context, req *priceplanpb.ListPricePlansRequest) error {
	// No specific business rules for listing price plans
	return nil
}
