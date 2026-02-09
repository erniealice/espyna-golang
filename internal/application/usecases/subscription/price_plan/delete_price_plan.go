package price_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
)

// DeletePricePlanRepositories groups all repository dependencies
type DeletePricePlanRepositories struct {
	PricePlan priceplanpb.PricePlanDomainServiceServer // Primary entity repository
}

// DeletePricePlanServices groups all business service dependencies
type DeletePricePlanServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// DeletePricePlanUseCase handles the business logic for deleting price_plans
type DeletePricePlanUseCase struct {
	repositories DeletePricePlanRepositories
	services     DeletePricePlanServices
}

// NewDeletePricePlanUseCase creates use case with grouped dependencies
func NewDeletePricePlanUseCase(
	repositories DeletePricePlanRepositories,
	services DeletePricePlanServices,
) *DeletePricePlanUseCase {
	return &DeletePricePlanUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete price_plan operation
func (uc *DeletePricePlanUseCase) Execute(ctx context.Context, req *priceplanpb.DeletePricePlanRequest) (*priceplanpb.DeletePricePlanResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPricePlan, ports.ActionDelete); err != nil {
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
	result, err := uc.repositories.PricePlan.DeletePricePlan(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.errors.deletion_failed", "price plan deletion failed")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return result, nil
}

// validateInput validates the input request
func (uc *DeletePricePlanUseCase) validateInput(ctx context.Context, req *priceplanpb.DeletePricePlanRequest) error {
	if req == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.request_required", "request is required")
		return errors.New(msg)
	}
	if req.Data == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.data_required", "price plan data is required")
		return errors.New(msg)
	}
	if req.Data.Id == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.id_required", "price plan ID is required")
		return errors.New(msg)
	}
	return nil
}

// validateBusinessRules enforces business constraints for price_plan deletion
func (uc *DeletePricePlanUseCase) validateBusinessRules(ctx context.Context, req *priceplanpb.DeletePricePlanRequest) error {
	// Validate price plan ID format
	if req.Data != nil && len(req.Data.Id) < 3 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.id_min_length", "price plan ID must be at least 3 characters long")
		return errors.New(msg)
	}

	// Additional business rules for deletion can be added here
	// For example, preventing deletion of price plans with active subscriptions

	return nil
}
