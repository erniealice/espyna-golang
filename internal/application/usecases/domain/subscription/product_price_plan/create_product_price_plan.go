package product_price_plan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
)

// CreateProductPricePlanRepositories groups all repository dependencies
type CreateProductPricePlanRepositories struct {
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer
	PricePlan        priceplanpb.PricePlanDomainServiceServer
	ProductPlan      productplanpb.ProductPlanDomainServiceServer
}

// CreateProductPricePlanServices groups all business service dependencies
type CreateProductPricePlanServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateProductPricePlanUseCase handles the business logic for creating product price plans
type CreateProductPricePlanUseCase struct {
	repositories CreateProductPricePlanRepositories
	services     CreateProductPricePlanServices
}

// NewCreateProductPricePlanUseCase creates a new CreateProductPricePlanUseCase
func NewCreateProductPricePlanUseCase(
	repositories CreateProductPricePlanRepositories,
	services CreateProductPricePlanServices,
) *CreateProductPricePlanUseCase {
	return &CreateProductPricePlanUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create product price plan operation
func (uc *CreateProductPricePlanUseCase) Execute(ctx context.Context, req *productpriceplanpb.CreateProductPricePlanRequest) (*productpriceplanpb.CreateProductPricePlanResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.PricePlan,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.authorization_failed", "Authorization failed for product price plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.PricePlan, entityid.ActionCreate)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.authorization_failed", "Authorization failed for product price plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.authorization_failed", "Authorization failed for product price plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	return uc.executeCore(ctx, req)
}

func (uc *CreateProductPricePlanUseCase) executeWithTransaction(ctx context.Context, req *productpriceplanpb.CreateProductPricePlanRequest) (*productpriceplanpb.CreateProductPricePlanResponse, error) {
	var result *productpriceplanpb.CreateProductPricePlanResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.creation_failed", "Product price plan creation failed [DEFAULT]")
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

func (uc *CreateProductPricePlanUseCase) executeCore(ctx context.Context, req *productpriceplanpb.CreateProductPricePlanRequest) (*productpriceplanpb.CreateProductPricePlanResponse, error) {
	if err := uc.validateInputWithTranslation(ctx, req); err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}

	if err := uc.enrichData(req.Data); err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}

	if err := uc.validateEntityReferencesWithTranslation(ctx, req.Data); err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.entity_reference_validation_failed", "Entity reference validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}

	resp, err := uc.repositories.ProductPricePlan.CreateProductPricePlan(ctx, req)
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.creation_failed", "Product price plan creation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return resp, nil
}

func (uc *CreateProductPricePlanUseCase) validateInputWithTranslation(ctx context.Context, req *productpriceplanpb.CreateProductPricePlanRequest) error {
	if req == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.validation.request_required", "Request is required [DEFAULT]")
		return errors.New(msg)
	}
	if req.Data == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.validation.data_required", "Product price plan data is required [DEFAULT]")
		return errors.New(msg)
	}
	if req.Data.PricePlanId == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.validation.price_plan_id_required", "Price plan ID is required [DEFAULT]")
		return errors.New(msg)
	}
	if req.Data.ProductPlanId == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.validation.product_plan_id_required", "Product plan ID is required [DEFAULT]")
		return errors.New(msg)
	}
	return nil
}

func (uc *CreateProductPricePlanUseCase) enrichData(productPricePlan *productpriceplanpb.ProductPricePlan) error {
	now := time.Now()

	if productPricePlan.Id == "" {
		productPricePlan.Id = uc.services.IDGenerator.GenerateID()
	}

	productPricePlan.DateCreated = &[]int64{now.UnixMilli()}[0]
	productPricePlan.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	productPricePlan.DateModified = &[]int64{now.UnixMilli()}[0]
	productPricePlan.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	productPricePlan.Active = true

	return nil
}

func (uc *CreateProductPricePlanUseCase) validateEntityReferencesWithTranslation(ctx context.Context, productPricePlan *productpriceplanpb.ProductPricePlan) error {
	// Validate referenced PricePlan
	var pricePlanPlanID string
	if productPricePlan.PricePlanId != "" {
		pricePlan, err := uc.repositories.PricePlan.ReadPricePlan(ctx, &priceplanpb.ReadPricePlanRequest{
			Data: &priceplanpb.PricePlan{Id: productPricePlan.PricePlanId},
		})
		if err != nil {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.price_plan_validation_failed", "Failed to validate price plan entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", msg, err)
		}
		if pricePlan == nil || pricePlan.Data == nil || len(pricePlan.Data) == 0 {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.price_plan_not_found", "Referenced price plan does not exist [DEFAULT]")
			return fmt.Errorf("%s with ID '%s'", msg, productPricePlan.PricePlanId)
		}
		pricePlanPlanID = pricePlan.Data[0].GetPlanId()
	}

	// Model D: Validate referenced ProductPlan exists AND belongs to the same Plan
	// as the PricePlan — prevents pricing a product_plan from a different plan.
	if productPricePlan.ProductPlanId != "" && uc.repositories.ProductPlan != nil {
		productPlan, err := uc.repositories.ProductPlan.ReadProductPlan(ctx, &productplanpb.ReadProductPlanRequest{
			Data: &productplanpb.ProductPlan{Id: productPricePlan.ProductPlanId},
		})
		if err != nil {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.product_plan_validation_failed", "Failed to validate product plan entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", msg, err)
		}
		if productPlan == nil || productPlan.Data == nil || len(productPlan.Data) == 0 {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.product_plan_not_found", "Referenced product plan does not exist [DEFAULT]")
			return fmt.Errorf("%s with ID '%s'", msg, productPricePlan.ProductPlanId)
		}
		if pricePlanPlanID != "" && productPlan.Data[0].GetPlanId() != pricePlanPlanID {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.product_plan_plan_mismatch", "Referenced product plan does not belong to the price plan's parent plan [DEFAULT]")
			return fmt.Errorf("%s (product_plan.plan_id='%s', price_plan.plan_id='%s')", msg, productPlan.Data[0].GetPlanId(), pricePlanPlanID)
		}
	}

	return nil
}
