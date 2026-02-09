package price_plan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
)

// UpdatePricePlanRepositories groups all repository dependencies
type UpdatePricePlanRepositories struct {
	PricePlan priceplanpb.PricePlanDomainServiceServer // Primary entity repository
	Plan      planpb.PlanDomainServiceServer           // Entity reference dependency
}

// UpdatePricePlanServices groups all business service dependencies
type UpdatePricePlanServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// UpdatePricePlanUseCase handles the business logic for updating price_plans
type UpdatePricePlanUseCase struct {
	repositories UpdatePricePlanRepositories
	services     UpdatePricePlanServices
}

// NewUpdatePricePlanUseCase creates use case with grouped dependencies
func NewUpdatePricePlanUseCase(
	repositories UpdatePricePlanRepositories,
	services UpdatePricePlanServices,
) *UpdatePricePlanUseCase {
	return &UpdatePricePlanUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update price_plan operation
func (uc *UpdatePricePlanUseCase) Execute(ctx context.Context, req *priceplanpb.UpdatePricePlanRequest) (*priceplanpb.UpdatePricePlanResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPricePlan, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichPricePlanData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	result, err := uc.repositories.PricePlan.UpdatePricePlan(ctx, req)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// validateInput validates the input request
func (uc *UpdatePricePlanUseCase) validateInput(ctx context.Context, req *priceplanpb.UpdatePricePlanRequest) error {
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
	if req.Data.Name == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.name_required", "price plan name is required")
		return errors.New(msg)
	}
	if req.Data.PlanId == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.plan_id_required", "plan ID is required")
		return errors.New(msg)
	}
	if req.Data.Currency == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.currency_required", "currency is required")
		return errors.New(msg)
	}
	return nil
}

// enrichPricePlanData adds generated fields and audit information
func (uc *UpdatePricePlanUseCase) enrichPricePlanData(pricePlan *priceplanpb.PricePlan) error {
	now := time.Now()

	// Update audit fields
	pricePlan.DateModified = &[]int64{now.UnixMilli()}[0]
	pricePlan.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints for price plans
func (uc *UpdatePricePlanUseCase) validateBusinessRules(ctx context.Context, pricePlan *priceplanpb.PricePlan) error {
	// Validate price plan ID length
	if len(pricePlan.Id) < 3 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.id_min_length", "price plan ID must be at least 3 characters long")
		return errors.New(msg)
	}

	// Validate price plan name length
	if len(pricePlan.Name) < 3 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.name_min_length", "price plan name must be at least 3 characters long")
		return errors.New(msg)
	}

	if len(pricePlan.Name) > 100 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.name_max_length", "price plan name cannot exceed 100 characters")
		return errors.New(msg)
	}

	// Validate Plan ID format validation
	if len(pricePlan.PlanId) < 3 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.plan_id_min_length", "plan ID must be at least 3 characters long")
		return errors.New(msg)
	}

	// Validate Amount validation
	if pricePlan.Amount <= 0 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.amount_positive", "price plan amount must be greater than 0")
		return errors.New(msg)
	}

	// Validate Currency validation
	if len(pricePlan.Currency) != 3 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.currency_format", "currency must be a 3-character currency code")
		return errors.New(msg)
	}

	// Validate Description length validation
	if len(pricePlan.Description) > 500 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.description_max_length", "price plan description cannot exceed 500 characters")
		return errors.New(msg)
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdatePricePlanUseCase) validateEntityReferences(ctx context.Context, pricePlan *priceplanpb.PricePlan) error {
	// Validate Plan entity reference
	if pricePlan.PlanId != "" {
		planId := pricePlan.PlanId
		plan, err := uc.repositories.Plan.ReadPlan(ctx, &planpb.ReadPlanRequest{
			Data: &planpb.Plan{Id: &planId},
		})
		if err != nil {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.errors.plan_validation_failed", "failed to validate plan entity reference")
			return fmt.Errorf("%s: %w", msg, err)
		}
		if plan == nil || plan.Data == nil || len(plan.Data) == 0 {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.errors.plan_not_found", "referenced plan with ID '%s' does not exist")
			return fmt.Errorf(msg, pricePlan.PlanId)
		}
		if !plan.Data[0].Active {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.errors.plan_not_active", "referenced plan with ID '%s' is not active")
			return fmt.Errorf(msg, pricePlan.PlanId)
		}
	}

	return nil
}
