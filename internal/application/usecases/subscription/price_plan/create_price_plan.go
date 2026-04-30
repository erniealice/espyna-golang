package price_plan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
)

// CreatePricePlanRepositories groups all repository dependencies.
//
// The PriceSchedule + Client refs were added 2026-04-28 to support the
// resolve-or-create-client-schedule path on client-scoped parent Plans (see
// plan §3.2 / §4.4 of 20260427-plan-client-scope). When the operator submits
// an empty price_schedule_id under a Plan whose client_id is set, the use
// case looks up or creates a matching client-scoped PriceSchedule and stamps
// its ID before delegating to the repository.
type CreatePricePlanRepositories struct {
	PricePlan     priceplanpb.PricePlanDomainServiceServer
	Plan          planpb.PlanDomainServiceServer
	PriceSchedule priceschedulepb.PriceScheduleDomainServiceServer
	Client        clientpb.ClientDomainServiceServer
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
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPricePlan, ports.ActionCreate); err != nil {
		return nil, err
	}

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
	// Name is optional — when blank, the UI falls back to the parent Plan's name.
	if req.Data.PlanId == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.plan_id_required", "plan ID is required")
		return errors.New(msg)
	}
	if req.Data.BillingCurrency == "" {
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
	// Validate price plan name length — only when a name was provided (optional field).
	if pricePlan.GetName() != "" {
		if len(pricePlan.GetName()) < 3 {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.name_min_length", "price plan name must be at least 3 characters long")
			return errors.New(msg)
		}
		if len(pricePlan.GetName()) > 100 {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.name_max_length", "price plan name cannot exceed 100 characters")
			return errors.New(msg)
		}
	}

	// Validate Plan ID format validation
	if len(pricePlan.PlanId) < 3 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.plan_id_min_length", "plan ID must be at least 3 characters long")
		return errors.New(msg)
	}

	// Validate Amount validation
	if pricePlan.BillingAmount <= 0 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.amount_positive", "price plan amount must be greater than 0")
		return errors.New(msg)
	}

	// Validate Currency validation
	if len(pricePlan.BillingCurrency) != 3 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.currency_format", "currency must be a 3-character currency code")
		return errors.New(msg)
	}

	// Validate Description length validation
	if len(pricePlan.GetDescription()) > 500 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "price_plan.validation.description_max_length", "price plan description cannot exceed 500 characters")
		return errors.New(msg)
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist.
// As a side effect (per plan §3.2 / 20260427-plan-client-scope), it cascades
// the parent Plan's client_id onto the supplied PricePlan, overwriting any
// caller-supplied value. Server-side coercion ensures pricePlan.client_id ==
// plan.client_id always.
//
// 2026-04-28 addition: when the parent Plan is client-scoped and the body
// either submits no price_schedule_id OR submits one belonging to a
// different client, applyClientScopedScheduleRule resolves-or-creates the
// matching client schedule and stamps its ID on the request. Mismatches
// are surfaced as `price_plan.errors.scheduleClientMismatch`. Master parents
// retain prior behaviour.
func (uc *CreatePricePlanUseCase) validateEntityReferences(ctx context.Context, pricePlan *priceplanpb.PricePlan) error {
	// Validate Plan entity reference
	if pricePlan.PlanId == "" {
		return nil
	}
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

	// 2026-04-30 cyclic-subscription-jobs plan §6 — reject MILESTONE × cyclic.
	//
	// Milestone billing implies a one-time fixed schedule (e.g., 4 milestone
	// payments over a project), which doesn't compose with recurring per-visit
	// cycles. The cyclic indicator on the Plan side is visits_per_cycle > 1
	// (the plan defaults to 1 for non-cyclic Plans). On the PricePlan side, a
	// non-zero billing_cycle_value also implies a recurring cadence.
	//
	// Both branches surface the same lyngua key
	// `price_plan.validation.milestoneCyclicBlock` so the drawer renders one
	// consistent banner. The PricePlan-edit drawer disables the MILESTONE
	// option client-side; this is the server-side defense.
	if err := validateMilestoneCyclicBlock(ctx, uc.services.TranslationService, pricePlan, plan.Data[0]); err != nil {
		return err
	}

	// §3.2 cascade — server-coerce PricePlan.client_id from the parent Plan.
	// Any body-supplied value is ignored to keep the denormalized invariant
	// `price_plan.client_id == plan.client_id` true by construction.
	parentClientID := plan.Data[0].GetClientId()
	pricePlan.ClientId = stringPtrOrNil(parentClientID)

	// Drawer hides the Name field by design — when the operator submits no
	// name, fall back to the parent Plan's name so the inserted row has a
	// stable display value. Operator-supplied names always win.
	if pricePlan.GetName() == "" {
		if pn := plan.Data[0].GetName(); pn != "" {
			pricePlan.Name = &pn
		}
	}

	// §3.2 / §4.4 — auto-resolve-or-create matching client PriceSchedule
	// when the parent Plan is client-scoped.
	if err := applyClientScopedScheduleRule(
		ctx,
		pricePlan,
		plan.Data[0],
		uc.repositories.PriceSchedule,
		uc.repositories.Client,
		uc.services.IDService,
		uc.services.TranslationService,
	); err != nil {
		return err
	}

	return nil
}

// stringPtrOrNil maps "" → nil (representing NULL on the optional client_id
// proto field), any non-empty value → its pointer.
func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	v := s
	return &v
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
