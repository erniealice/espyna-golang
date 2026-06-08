package product_price_plan

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
)

// ReadProductPricePlanUseCase handles the business logic for reading a product price plan
type ReadProductPricePlanUseCase struct {
	repositories ReadProductPricePlanRepositories
	services     ReadProductPricePlanServices
}

// NewReadProductPricePlanUseCase creates a new ReadProductPricePlanUseCase
func NewReadProductPricePlanUseCase(
	repositories ReadProductPricePlanRepositories,
	services ReadProductPricePlanServices,
) *ReadProductPricePlanUseCase {
	return &ReadProductPricePlanUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read product price plan operation
func (uc *ReadProductPricePlanUseCase) Execute(ctx context.Context, req *productpriceplanpb.ReadProductPricePlanRequest) (*productpriceplanpb.ReadProductPricePlanResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.PricePlan, entityid.ActionRead); err != nil {
		return nil, err
	}

	if err := uc.validateInputWithTranslation(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	resp, err := uc.repositories.ProductPricePlan.ReadProductPricePlan(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.Translator, "product_price_plan.errors.not_found", map[string]interface{}{"productPricePlanId": req.Data.Id}, "Product price plan not found")
			return nil, errors.New(translatedError)
		}
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.read_failed", "Failed to read product price plan")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if resp == nil || resp.Data == nil || len(resp.Data) == 0 {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.not_found", "Product price plan with ID \"{id}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{id}", req.Data.Id)
		return nil, errors.New(translatedError)
	}

	return resp, nil
}

func (uc *ReadProductPricePlanUseCase) validateInputWithTranslation(ctx context.Context, req *productpriceplanpb.ReadProductPricePlanRequest) error {
	if req == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.validation.request_required", "Request is required [DEFAULT]")
		return errors.New(msg)
	}
	if req.Data == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.validation.data_required", "Product price plan data is required [DEFAULT]")
		return errors.New(msg)
	}
	if req.Data.Id == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.validation.id_required", "Product price plan ID is required [DEFAULT]")
		return errors.New(msg)
	}
	return nil
}
