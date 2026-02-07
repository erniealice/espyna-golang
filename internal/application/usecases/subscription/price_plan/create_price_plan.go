package price_plan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	planpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan"
	priceplanpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/price_plan"
)

// CreatePricePlanRepositories groups all repository dependencies
type CreatePricePlanRepositories struct {
	PricePlan priceplanpb.PricePlanDomainServiceServer // Primary entity repository
	Plan      planpb.PlanDomainServiceServer           // Entity reference dependency
}

// CreatePricePlanServices groups all business service dependencies
type CreatePricePlanServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreatePricePlanUseCase handles the business logic for creating price_plans
type CreatePricePlanUseCase struct {
	repositories CreatePricePlanRepositories
	services     CreatePricePlanServices
}

// NewCreatePricePlanUseCase creates use case with grouped dependencies
func NewCreatePricePlanUseCase(
	repositories CreatePricePlanRepositories,
	services CreatePricePlanServices,
) *CreatePricePlanUseCase {
	return &CreatePricePlanUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create price_plan operation
func (uc *CreatePricePlanUseCase) Execute(ctx context.Context, req *priceplanpb.CreatePricePlanRequest) (*priceplanpb.CreatePricePlanResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// validateInput validates the input request
func (uc *CreatePricePlanUseCase) validateInput(ctx context.Context, req *priceplanpb.CreatePricePlanRequest) error {
	if req == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.request_required", "request is required")
		return errors.New(msg)
	}
	if req.Data == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.data_required", "price plan data is required")
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
func (uc *CreatePricePlanUseCase) enrichPricePlanData(pricePlan *priceplanpb.PricePlan) error {
	now := time.Now()

	// Generate PricePlan ID if not provided
	if pricePlan.Id == "" {
		pricePlan.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	pricePlan.DateCreated = &[]int64{now.UnixMilli()}[0]
	pricePlan.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	pricePlan.DateModified = &[]int64{now.UnixMilli()}[0]
	pricePlan.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	pricePlan.Active = true

	return nil
}

// validateBusinessRules enforces business constraints for price plans
func (uc *CreatePricePlanUseCase) validateBusinessRules(ctx context.Context, pricePlan *priceplanpb.PricePlan) error {
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
func (uc *CreatePricePlanUseCase) validateEntityReferences(ctx context.Context, pricePlan *priceplanpb.PricePlan) error {
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

// executeWithTransaction executes price plan creation within a transaction
func (uc *CreatePricePlanUseCase) executeWithTransaction(ctx context.Context, req *priceplanpb.CreatePricePlanRequest) (*priceplanpb.CreatePricePlanResponse, error) {
	var result *priceplanpb.CreatePricePlanResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.errors.creation_failed", "price plan creation failed")
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

// executeCore contains the core business logic (moved from original Execute method)
func (uc *CreatePricePlanUseCase) executeCore(ctx context.Context, req *priceplanpb.CreatePricePlanRequest) (*priceplanpb.CreatePricePlanResponse, error) {
	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	if err := uc.enrichPricePlanData(req.Data); err != nil {
		return nil, err
	}

	// Delegate to repository
	return uc.repositories.PricePlan.CreatePricePlan(ctx, &priceplanpb.CreatePricePlanRequest{
		Data: req.Data,
	})
}
