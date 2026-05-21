package price_plan

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"

	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
)

// validateMilestoneCyclicBlock implements the cyclic-subscription-jobs plan §6
// rule: MILESTONE billing is incompatible with cyclic engagement modelling.
//
// Two cyclic indicators trigger the block:
//
//  1. Plan.visits_per_cycle > 1 — the parent Plan declares a multi-visit
//     cadence (Lawn Care Weekly = 4, Pro Cleaning Biweekly = 2). visits=1 is
//     the default for both cyclic single-visit AND non-cyclic Plans, so it
//     does NOT alone trip the block; the PricePlan-side cycle field handles
//     that case.
//  2. PricePlan.billing_cycle_value > 0 — the PricePlan declares a recurring
//     cadence (RECURRING/CONTRACT × PER_CYCLE). A non-zero cycle_value with
//     billing_kind = MILESTONE is the canonical rejection case.
//
// Reasoning (plan §6): milestones are phase-completion-driven, single-engagement
// events; cycles are time-driven, recurring events. Trying to combine them
// produces ambiguous semantics — does milestone M1 fire on cycle 1's design
// phase or cycle 2's design phase?
//
// Both Create and Update use cases call this helper after they've resolved
// the parent Plan record, so the pricePlan + plan args are non-nil.
//
// Returns nil when the combination is allowed. Returns a translation-keyed
// error (key `price_plan.validation.milestoneCyclicBlock`) when blocked.
func validateMilestoneCyclicBlock(
	ctx context.Context,
	translationService ports.Translator,
	pricePlan *priceplanpb.PricePlan,
	plan *planpb.Plan,
) error {
	if pricePlan == nil {
		return nil
	}
	if pricePlan.GetBillingKind() != priceplanpb.BillingKind_BILLING_KIND_MILESTONE {
		return nil
	}

	planVisits := int32(0)
	if plan != nil {
		planVisits = plan.GetVisitsPerCycle()
	}
	cycleValue := pricePlan.GetBillingCycleValue()

	// MILESTONE × multi-visit cadence on Plan: blocked.
	// MILESTONE × non-zero billing_cycle_value on PricePlan: blocked.
	if planVisits > 1 || cycleValue > 0 {
		msg := contextutil.GetTranslatedMessageWithContext(
			ctx, translationService,
			"price_plan.validation.milestoneCyclicBlock",
			"Milestone billing is not supported on cyclic plans (RECURRING / CONTRACT × PER_CYCLE / multi-visit). [DEFAULT]",
		)
		return errors.New(msg)
	}
	return nil
}
