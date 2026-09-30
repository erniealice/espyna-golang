package product_price_plan

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	chargepolicypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
	chargepolicyversionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
	"google.golang.org/protobuf/proto"
)

// UpdateProductPricePlanRepositories groups all repository dependencies
type UpdateProductPricePlanRepositories struct {
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer
	PricePlan        priceplanpb.PricePlanDomainServiceServer
	ProductPlan      productplanpb.ProductPlanDomainServiceServer
	// Optional charge policy opt-in guard collaborators (see create).
	ChargePolicy        chargepolicypb.ChargePolicyDomainServiceServer
	ChargePolicyVersion chargepolicyversionpb.ChargePolicyVersionDomainServiceServer
}

// UpdateProductPricePlanServices groups all business service dependencies
type UpdateProductPricePlanServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.PricePlan,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.authorization_failed", "Authorization failed for product price plans [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.PricePlan, entityid.ActionUpdate)
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

func (uc *UpdateProductPricePlanUseCase) executeWithTransaction(ctx context.Context, req *productpriceplanpb.UpdateProductPricePlanRequest) (*productpriceplanpb.UpdateProductPricePlanResponse, error) {
	var result *productpriceplanpb.UpdateProductPricePlanResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.update_failed", "Product price plan update failed [DEFAULT]")
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
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if err := uc.enrichData(req.Data); err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}

	if err := uc.validateEntityReferencesWithTranslation(ctx, req.Data); err != nil {
		msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.entity_reference_validation_failed", "Entity reference validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", msg, err)
	}

	// Charge policy opt-in guard on the EFFECTIVE line (stored row overlaid with the request):
	// a partial update that changes billing_treatment or markup_bps of a policy-bound line is
	// re-validated, and a request that sets charge_policy_id is validated as a whole.
	if err := uc.guardChargePolicy(ctx, req.Data); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.ProductPricePlan.UpdateProductPricePlan(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.Translator, "product_price_plan.errors.not_found", map[string]interface{}{"productPricePlanId": req.Data.Id}, "Product price plan not found")
			return nil, errors.New(translatedError)
		}
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.update_failed", "Product price plan update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *UpdateProductPricePlanUseCase) validateInputWithTranslation(ctx context.Context, req *productpriceplanpb.UpdateProductPricePlanRequest) error {
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
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.price_plan_validation_failed", "Failed to validate price plan entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", msg, err)
		}
		if pricePlan == nil || pricePlan.Data == nil || len(pricePlan.Data) == 0 {
			msg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_price_plan.errors.price_plan_not_found", "Referenced price plan does not exist [DEFAULT]")
			return fmt.Errorf("%s with ID '%s'", msg, productPricePlan.PricePlanId)
		}
		pricePlanPlanID = pricePlan.Data[0].GetPlanId()
	}

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

// guardChargePolicy merges the stored line with the request and runs validateChargePolicyOptIn.
// When the request touches none of the guarded fields and the stored line has no policy, the
// legacy path is untouched (no extra reads beyond the single stored-row read).
func (uc *UpdateProductPricePlanUseCase) guardChargePolicy(ctx context.Context, patch *productpriceplanpb.ProductPricePlan) error {
	if uc.repositories.ProductPricePlan == nil {
		return nil
	}
	touches := patch.ChargePolicyId != nil || patch.MarkupBps != nil || patch.BillingTreatment != productpriceplanpb.BillingTreatment_BILLING_TREATMENT_UNSPECIFIED || patch.PricePlanId != ""
	if !touches {
		return nil
	}
	cur, err := uc.repositories.ProductPricePlan.ReadProductPricePlan(ctx, &productpriceplanpb.ReadProductPricePlanRequest{Data: &productpriceplanpb.ProductPricePlan{Id: patch.Id}})
	if err != nil && !isRepoNotFound(err) {
		// Fail closed (C12): a stored-row read failure must never let an unguarded write through.
		return guardRepoErr("read product_price_plan", err, patch.Id)
	}
	if err != nil || cur == nil || len(cur.Data) == 0 {
		if isClearChargePolicy(patch) {
			// Fail closed on the explicit clear: a foreign-workspace or unknown row is not found
			// (the workspace-scoped read hides it); never fall through to a write.
			msg := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.Translator, "product_price_plan.errors.not_found", map[string]interface{}{"productPricePlanId": patch.Id}, "Product price plan not found")
			return errors.New(msg)
		}
		// Unknown row: the update itself will report not-found; the guard has nothing to merge.
		return nil
	}
	// Derived write (C12, moved here from the view): switching a policy-bound line off
	// USAGE_BASED clears the policy in the same write, whichever edit surface sent the request
	// (an explicit charge_policy_id in the request still wins and is refused by the guard below).
	if patch.ChargePolicyId == nil && cur.Data[0].GetChargePolicyId() != "" &&
		patch.BillingTreatment != productpriceplanpb.BillingTreatment_BILLING_TREATMENT_UNSPECIFIED &&
		patch.BillingTreatment != productpriceplanpb.BillingTreatment_BILLING_TREATMENT_USAGE_BASED {
		cleared := ""
		patch.ChargePolicyId = &cleared
	}
	eff := proto.Clone(cur.Data[0]).(*productpriceplanpb.ProductPricePlan)
	if isClearChargePolicy(patch) {
		// Explicit clear: the effective line has no policy and no markup (always allowed).
		eff.ChargePolicyId = nil
		eff.MarkupBps = nil
	} else {
		if patch.ChargePolicyId != nil {
			eff.ChargePolicyId = patch.ChargePolicyId
		}
		if patch.MarkupBps != nil {
			eff.MarkupBps = patch.MarkupBps
		}
	}
	if patch.BillingTreatment != productpriceplanpb.BillingTreatment_BILLING_TREATMENT_UNSPECIFIED {
		eff.BillingTreatment = patch.BillingTreatment
	}
	if patch.PricePlanId != "" {
		eff.PricePlanId = patch.PricePlanId
	}
	return validateChargePolicyOptIn(ctx, chargePolicyDeps{
		ChargePolicy:        uc.repositories.ChargePolicy,
		ChargePolicyVersion: uc.repositories.ChargePolicyVersion,
		PricePlan:           uc.repositories.PricePlan,
	}, eff, uc.services.Translator)
}

// isClearChargePolicy reports the typed "clear charge policy" intent: charge_policy_id present
// (optional field set) and empty. The postgres adapter maps it to charge_policy_id = NULL and
// markup_bps = NULL; an empty string is never written to the FK column.
func isClearChargePolicy(patch *productpriceplanpb.ProductPricePlan) bool {
	return patch.ChargePolicyId != nil && patch.GetChargePolicyId() == ""
}
