package price_plan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
)

// UpdatePricePlanRepositories groups all repository dependencies.
//
// PriceSchedule + Client refs added 2026-04-28 — same auto-resolve-or-create
// rationale as CreatePricePlanRepositories. UpdatePricePlan reuses the path
// when an operator clears the schedule field on a client-scoped PricePlan.
type UpdatePricePlanRepositories struct {
	PricePlan     priceplanpb.PricePlanDomainServiceServer
	Plan          planpb.PlanDomainServiceServer
	PriceSchedule priceschedulepb.PriceScheduleDomainServiceServer
	Client        clientpb.ClientDomainServiceServer
}

// UpdatePricePlanServices groups all business service dependencies
type UpdatePricePlanServices struct {
	Authorizer       ports.Authorizer // Current: RBAC and permissions
	Transactor       ports.Transactor // Current: Database transactions
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	ReferenceChecker ports.ReferenceChecker // §3.5 — N>1 active subscription confirm gate
	IDGenerator      ports.IDGenerator      // 2026-04-28 — needed for client-scope schedule auto-create
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.PricePlan,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Entity reference validation — also cascades client_id from the parent
	// Plan onto req.Data, mirroring CreatePricePlan (§3.2).
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// §3.5 — N>1 confirm gate for monetary edits on a client-scoped PricePlan.
	if err := uc.checkMultiSubscriptionConfirm(ctx, req.Data); err != nil {
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
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_plan.validation.request_required", "request is required")
		return errors.New(msg)
	}
	if req.Data == nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_plan.validation.data_required", "price plan data is required")
		return errors.New(msg)
	}
	if req.Data.Id == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_plan.validation.id_required", "price plan ID is required")
		return errors.New(msg)
	}
	// Name is optional — when blank, the UI falls back to the parent Plan's name.
	if req.Data.PlanId == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_plan.validation.plan_id_required", "plan ID is required")
		return errors.New(msg)
	}
	if req.Data.BillingCurrency == "" {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_plan.validation.currency_required", "currency is required")
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
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_plan.validation.id_min_length", "price plan ID must be at least 3 characters long")
		return errors.New(msg)
	}

	// Validate price plan name length — only when a name was provided (optional field).
	if pricePlan.GetName() != "" {
		if len(pricePlan.GetName()) < 3 {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_plan.validation.name_min_length", "price plan name must be at least 3 characters long")
			return errors.New(msg)
		}
		if len(pricePlan.GetName()) > 100 {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_plan.validation.name_max_length", "price plan name cannot exceed 100 characters")
			return errors.New(msg)
		}
	}

	// Validate Plan ID format validation
	if len(pricePlan.PlanId) < 3 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_plan.validation.plan_id_min_length", "plan ID must be at least 3 characters long")
		return errors.New(msg)
	}

	// Validate Amount validation
	if pricePlan.BillingAmount <= 0 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_plan.validation.amount_positive", "price plan amount must be greater than 0")
		return errors.New(msg)
	}

	// Validate Currency validation
	if len(pricePlan.BillingCurrency) != 3 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_plan.validation.currency_format", "currency must be a 3-character currency code")
		return errors.New(msg)
	}

	// Validate Description length validation
	if len(pricePlan.GetDescription()) > 500 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_plan.validation.description_max_length", "price plan description cannot exceed 500 characters")
		return errors.New(msg)
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist. Also
// cascades the parent Plan's client_id onto the request body — server-side
// coercion per §3.2 keeps the denormalized invariant
// `price_plan.client_id == plan.client_id` true on every write.
//
// 2026-04-28 addition: same auto-resolve-or-create-client-schedule behaviour
// as CreatePricePlan when the body submits an empty price_schedule_id under
// a client-scoped Plan, and the same scheduleClientMismatch reject path
// when an explicit pick belongs to a different client.
func (uc *UpdatePricePlanUseCase) validateEntityReferences(ctx context.Context, pricePlan *priceplanpb.PricePlan) error {
	if pricePlan.PlanId == "" {
		return nil
	}
	planId := pricePlan.PlanId
	plan, err := uc.repositories.Plan.ReadPlan(ctx, &planpb.ReadPlanRequest{
		Data: &planpb.Plan{Id: &planId},
	})
	if err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_plan.errors.plan_validation_failed", "failed to validate plan entity reference")
		return fmt.Errorf("%s: %w", msg, err)
	}
	if plan == nil || plan.Data == nil || len(plan.Data) == 0 {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_plan.errors.plan_not_found", "referenced plan with ID '%s' does not exist")
		return fmt.Errorf(msg, pricePlan.PlanId)
	}
	if !plan.Data[0].Active {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "price_plan.errors.plan_not_active", "referenced plan with ID '%s' is not active")
		return fmt.Errorf(msg, pricePlan.PlanId)
	}

	// 2026-04-30 cyclic-subscription-jobs plan §6 — reject MILESTONE × cyclic.
	// See create_price_plan.go for the full rationale; this guard is the
	// symmetric Update-side defense. Same lyngua key
	// `price_plan.validation.milestoneCyclicBlock`.
	if err := validateMilestoneCyclicBlock(ctx, uc.services.Translator, pricePlan, plan.Data[0]); err != nil {
		return err
	}
	if err := validateAdHoc(ctx, uc.services.Translator, pricePlan, plan.Data[0]); err != nil {
		return err
	}

	// §3.2 cascade — server-coerce PricePlan.client_id from the parent
	// Plan, overwriting any body-supplied value.
	parentClientID := plan.Data[0].GetClientId()
	pricePlan.ClientId = stringPtrOrNil(parentClientID)

	// Drawer hides the Name field by design — when the operator submits no
	// name, fall back to the parent Plan's name. Operator-supplied names win.
	if pricePlan.GetName() == "" {
		if pn := plan.Data[0].GetName(); pn != "" {
			pricePlan.Name = &pn
		}
	}

	// §3.2 / §4.4 — auto-resolve-or-create matching client PriceSchedule.
	if err := applyClientScopedScheduleRule(
		ctx,
		pricePlan,
		plan.Data[0],
		uc.repositories.PriceSchedule,
		uc.repositories.Client,
		uc.services.IDGenerator,
		uc.services.Translator,
	); err != nil {
		return err
	}

	return nil
}

// checkMultiSubscriptionConfirm implements plan §3.5 — when a client-scoped
// PricePlan has its monetary fields changed and N > 1 active subscriptions
// reference it, require an explicit confirmation flag (carried via context).
//
// The check is deliberately a no-op when:
//   - the PricePlan is master (client_id == "")
//   - no monetary fields are changing
//   - the reference checker is unwired (provider doesn't support it)
//   - the caller has already confirmed (contextutil.IsConfirmed(ctx) == true)
//
// The error message uses the lyngua key
// `price_plan.errors.multiSubscriptionConfirmRequired`. The handler intercepts
// it and renders the user-facing
// `price_plan.confirms.editAmountMultipleEngagements` dialog.
func (uc *UpdatePricePlanUseCase) checkMultiSubscriptionConfirm(ctx context.Context, pricePlan *priceplanpb.PricePlan) error {
	if pricePlan == nil || pricePlan.GetClientId() == "" {
		return nil
	}
	if uc.services.ReferenceChecker == nil {
		return nil
	}
	if contextutil.IsConfirmed(ctx) {
		return nil
	}

	// Read the existing row so we can detect whether monetary fields changed.
	existingResp, err := uc.repositories.PricePlan.ReadPricePlan(ctx, &priceplanpb.ReadPricePlanRequest{
		Data: &priceplanpb.PricePlan{Id: pricePlan.GetId()},
	})
	if err != nil || existingResp == nil || len(existingResp.GetData()) == 0 {
		// Defer to other validators if the row is missing — they'll surface
		// the not-found error.
		return nil
	}
	existing := existingResp.GetData()[0]

	monetaryChanged := existing.GetBillingAmount() != pricePlan.GetBillingAmount() ||
		existing.GetBillingCurrency() != pricePlan.GetBillingCurrency() ||
		existing.GetBillingCycleValue() != pricePlan.GetBillingCycleValue() ||
		existing.GetBillingCycleUnit() != pricePlan.GetBillingCycleUnit() ||
		existing.GetDefaultTermValue() != pricePlan.GetDefaultTermValue() ||
		existing.GetDefaultTermUnit() != pricePlan.GetDefaultTermUnit()
	if !monetaryChanged {
		return nil
	}

	count, err := uc.services.ReferenceChecker.GetActiveSubscriptionCountForPricePlan(ctx, pricePlan.GetId())
	if err != nil {
		return err
	}
	if count <= 1 {
		return nil
	}

	msg := contextutil.GetTranslatedMessageWithContext(
		ctx, uc.services.Translator,
		"price_plan.errors.multiSubscriptionConfirmRequired",
		"Confirmation required — N > 1 attached subscriptions and monetary fields changing. [DEFAULT]",
	)
	return errors.New(msg)
}
