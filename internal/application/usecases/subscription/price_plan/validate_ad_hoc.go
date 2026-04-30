package price_plan

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"

	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
)

// validateAdHoc implements ad-hoc-subscription-billing plan §6 validation rules.
// Each Create/Update PricePlan use case calls this after resolving the parent
// Plan; non-AD_HOC kinds short-circuit so existing scenarios are untouched.
//
// Rules (codex CRIT-4 + MAJ-4 closed):
//
//  1. AD_HOC × basis ∉ {TOTAL_PACKAGE, PER_OCCURRENCE} → ad_hoc_invalid_basis
//  2. AD_HOC × TOTAL_PACKAGE × entitled_occurrences ≤ 0 → entitlement_required
//  3. AD_HOC × PER_OCCURRENCE × entitled_occurrences ≠ NULL → entitlement_invalid_for_per_call
//  4. AD_HOC × billing_cycle_value > 0 → ad_hoc_billing_incompatible_with_cycle_cadence
//  5. AD_HOC × Plan.visits_per_cycle > 1 → ad_hoc_visits_per_cycle_must_be_one
//  6. AD_HOC × {pool, per-call} × Plan.job_template_id IS NULL → pool_no_template / pay_per_call_no_template
func validateAdHoc(
	ctx context.Context,
	translationService ports.TranslationService,
	pricePlan *priceplanpb.PricePlan,
	plan *planpb.Plan,
) error {
	if pricePlan == nil {
		return nil
	}
	if pricePlan.GetBillingKind() != priceplanpb.BillingKind_BILLING_KIND_AD_HOC {
		return nil
	}

	basis := pricePlan.GetAmountBasis()
	switch basis {
	case priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE,
		priceplanpb.AmountBasis_AMOUNT_BASIS_PER_OCCURRENCE:
		// ok
	default:
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, translationService,
			"price_plan.validation.adHocInvalidBasis",
			"AD_HOC plans require amount basis = Total Package (pool) or Per Occurrence (pay-per-call). [DEFAULT]",
		))
	}

	if pricePlan.GetBillingCycleValue() > 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, translationService,
			"price_plan.validation.adHocBillingIncompatibleWithCycleCadence",
			"AD_HOC plans cannot have a billing cycle. [DEFAULT]",
		))
	}

	if plan != nil && plan.GetVisitsPerCycle() > 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, translationService,
			"price_plan.validation.adHocVisitsPerCycleMustBeOne",
			"AD_HOC plans cannot use multi-visit cycles (visits_per_cycle must be 1 or empty). [DEFAULT]",
		))
	}

	// Both AD_HOC variants need a JobTemplate to materialize usage Jobs.
	planTemplateID := ""
	if plan != nil {
		planTemplateID = plan.GetJobTemplateId()
	}
	if planTemplateID == "" {
		key := "price_plan.validation.adHocPoolNoTemplate"
		def := "AD_HOC pool plans require the parent Plan to have a Job Template. [DEFAULT]"
		if basis == priceplanpb.AmountBasis_AMOUNT_BASIS_PER_OCCURRENCE {
			key = "price_plan.validation.adHocPerCallNoTemplate"
			def = "AD_HOC per-occurrence plans require the parent Plan to have a Job Template. [DEFAULT]"
		}
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, translationService, key, def,
		))
	}

	if basis == priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE {
		if pricePlan.GetEntitledOccurrences() <= 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx, translationService,
				"price_plan.validation.adHocEntitlementRequired",
				"AD_HOC pool plans require entitled occurrences > 0. [DEFAULT]",
			))
		}
	} else {
		// PER_OCCURRENCE: entitled_occurrences must be NULL/zero.
		if pricePlan.EntitledOccurrences != nil {
			return errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx, translationService,
				"price_plan.validation.adHocEntitlementInvalidForPerCall",
				"Entitled occurrences are only valid on AD_HOC pool plans, not pay-per-call. [DEFAULT]",
			))
		}
	}

	return nil
}
