package product_plan

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	productplanpb "leapfor.xyz/esqyma/golang/v1/domain/product/product_plan"
)

// ListProductPlansUseCase handles the business logic for listing product plans
type ListProductPlansUseCase struct {
	repositories ListProductPlansRepositories
	services     ListProductPlansServices
}

// NewListProductPlansUseCase creates a new ListProductPlansUseCase
func NewListProductPlansUseCase(
	repositories ListProductPlansRepositories,
	services ListProductPlansServices,
) *ListProductPlansUseCase {
	return &ListProductPlansUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list product plans operation
func (uc *ListProductPlansUseCase) Execute(ctx context.Context, req *productplanpb.ListProductPlansRequest) (*productplanpb.ListProductPlansResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.authorization_failed", "Authorization failed for product plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductPlan, ports.ActionList)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.authorization_failed", "Authorization failed for product plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.authorization_failed", "Authorization failed for product plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductPlan.ListProductPlans(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.errors.list_failed", "Failed to retrieve product plans [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListProductPlansUseCase) validateInput(ctx context.Context, req *productplanpb.ListProductPlansRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_plan.validation.request_required", "Request is required [DEFAULT]"))
	}
	// Additional validation can be added here if needed
	return nil
}
