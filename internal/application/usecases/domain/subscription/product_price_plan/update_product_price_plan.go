package product_price_plan

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
)

// UpdateProductPricePlanRepositories groups all repository dependencies
type UpdateProductPricePlanRepositories struct {
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer
	PricePlan        priceplanpb.PricePlanDomainServiceServer
	ProductPlan      productplanpb.ProductPlanDomainServiceServer
}

// UpdateProductPricePlanServices groups all business service dependencies
type UpdateProductPricePlanServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateProductPricePlanUseCase handles the business logic for updating product price plans
type UpdateProductPricePlanUseCase struct {
	repositories UpdateProductPricePlanRepositories
	services     UpdateProductPricePlanServices
}

// NewUpdateProductPricePlanUseCase creates a new UpdateProductPricePlanUseCase
func NewUpdateProductPricePlanUseCase(
	repositories UpdateProductPricePlanRepositories,
	services UpdateProductPricePlanServices,
) *UpdateProductPricePlanUseCase {
	return &UpdateProductPricePlanUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update product price plan operation
func (uc *UpdateProductPricePlanUseCase) Execute(ctx context.Context, req *productpriceplanpb.UpdateProductPricePlanRequest) (*productpriceplanpb.UpdateProductPricePlanResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPricePlan, ports.ActionUpdate); err != nil {
		return nil, err
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_price_plan.errors.authorization_failed", "Authorization failed for product price plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityPricePlan, ports.ActionUpdate)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_price_plan.errors.authorization_failed", "Authorization failed for product price plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_price_plan.errors.authorization_failed", "Authorization failed for product price plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	return uc.executeCore(ctx, req)
}

func (uc *UpdateProductPricePlanUseCase) executeWithTransaction(ctx context.Context, req *productpriceplanpb.UpdateProductPricePlanRequest) (*productpriceplanpb.UpdateProductPricePlanResponse, error) {
	var result *productpriceplanpb.UpdateProductPricePlanResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_price_plan.errors.update_failed", "Product price plan update failed [DEFAULT]")
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

func (uc *UpdateProductPricePlanUseCase) executeCore(ctx context.Context, req *productpriceplanpb.UpdateProductPricePlanRequest) (*productpriceplanpb.UpdateProductPricePlanResponse, error) {
	if err := uc.validateInputWithTranslation(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_price_plan.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if err := uc.enrichData(req.Data); err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_price_plan.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}

	if err := uc.validateEntityReferencesWithTranslation(ctx, req.Data); err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_price_plan.errors.entity_reference_validation_failed", "Entity reference validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}

	resp, err := uc.repositories.ProductPricePlan.UpdateProductPricePlan(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "product_price_plan.errors.not_found", map[string]interface{}{"productPricePlanId": req.Data.Id}, "Product price plan not found")
			return nil, errors.New(translatedError)
		}
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_price_plan.errors.update_failed", "Product price plan update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *UpdateProductPricePlanUseCase) validateInputWithTranslation(ctx context.Context, req *productpriceplanpb.UpdateProductPricePlanRequest) error {
	if req == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_price_plan.validation.request_required", "Request is required [DEFAULT]")
		return errors.New(msg)
	}
	if req.Data == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_price_plan.validation.data_required", "Product price plan data is required [DEFAULT]")
		return errors.New(msg)
	}
	if req.Data.Id == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_price_plan.validation.id_required", "Product price plan ID is required [DEFAULT]")
		return errors.New(msg)
	}
	return nil
}

func (uc *UpdateProductPricePlanUseCase) enrichData(productPricePlan *productpriceplanpb.ProductPricePlan) error {
	now := time.Now()
	productPricePlan.DateModified = &[]int64{now.UnixMilli()}[0]
	productPricePlan.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return nil
}

// validateEntityReferencesWithTranslation validates referenced entities exist.
// Model D: when product_plan_id is provided, confirm it exists and (when the
// price_plan is also identifiable) shares the same plan_id as the parent PricePlan.
func (uc *UpdateProductPricePlanUseCase) validateEntityReferencesWithTranslation(ctx context.Context, productPricePlan *productpriceplanpb.ProductPricePlan) error {
	var pricePlanPlanID string
	if productPricePlan.PricePlanId != "" && uc.repositories.PricePlan != nil {
		pricePlan, err := uc.repositories.PricePlan.ReadPricePlan(ctx, &priceplanpb.ReadPricePlanRequest{
			Data: &priceplanpb.PricePlan{Id: productPricePlan.PricePlanId},
		})
		if err != nil {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_price_plan.errors.price_plan_validation_failed", "Failed to validate price plan entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", msg, err)
		}
		if pricePlan == nil || pricePlan.Data == nil || len(pricePlan.Data) == 0 {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_price_plan.errors.price_plan_not_found", "Referenced price plan does not exist [DEFAULT]")
			return fmt.Errorf("%s with ID '%s'", msg, productPricePlan.PricePlanId)
		}
		pricePlanPlanID = pricePlan.Data[0].GetPlanId()
	}

	if productPricePlan.ProductPlanId != "" && uc.repositories.ProductPlan != nil {
		productPlan, err := uc.repositories.ProductPlan.ReadProductPlan(ctx, &productplanpb.ReadProductPlanRequest{
			Data: &productplanpb.ProductPlan{Id: productPricePlan.ProductPlanId},
		})
		if err != nil {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_price_plan.errors.product_plan_validation_failed", "Failed to validate product plan entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", msg, err)
		}
		if productPlan == nil || productPlan.Data == nil || len(productPlan.Data) == 0 {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_price_plan.errors.product_plan_not_found", "Referenced product plan does not exist [DEFAULT]")
			return fmt.Errorf("%s with ID '%s'", msg, productPricePlan.ProductPlanId)
		}
		if pricePlanPlanID != "" && productPlan.Data[0].GetPlanId() != pricePlanPlanID {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_price_plan.errors.product_plan_plan_mismatch", "Referenced product plan does not belong to the price plan's parent plan [DEFAULT]")
			return fmt.Errorf("%s (product_plan.plan_id='%s', price_plan.plan_id='%s')", msg, productPlan.Data[0].GetPlanId(), pricePlanPlanID)
		}
	}

	return nil
}
