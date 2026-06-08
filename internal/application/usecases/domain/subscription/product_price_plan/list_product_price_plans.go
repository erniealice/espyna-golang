package product_price_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
)

// ListProductPricePlansUseCase handles the business logic for listing product price plans
type ListProductPricePlansUseCase struct {
	repositories ListProductPricePlansRepositories
	services     ListProductPricePlansServices
}

// NewListProductPricePlansUseCase creates a new ListProductPricePlansUseCase
func NewListProductPricePlansUseCase(
	repositories ListProductPricePlansRepositories,
	services ListProductPricePlansServices,
) *ListProductPricePlansUseCase {
	return &ListProductPricePlansUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list product price plans operation
func (uc *ListProductPricePlansUseCase) Execute(ctx context.Context, req *productpriceplanpb.ListProductPricePlansRequest) (*productpriceplanpb.ListProductPricePlansResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.PricePlan, entityid.ActionList); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	resp, err := uc.repositories.ProductPricePlan.ListProductPricePlans(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.list_failed", "Failed to retrieve product price plans [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *ListProductPricePlansUseCase) validateInput(ctx context.Context, req *productpriceplanpb.ListProductPricePlansRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.validation.request_required", "Request is required [DEFAULT]"))
	}
	return nil
}
