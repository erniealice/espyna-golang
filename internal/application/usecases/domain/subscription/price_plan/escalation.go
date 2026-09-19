package price_plan

import (
	"errors"

	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
)

// EscalationValue is the shared, persistence-neutral representation used by
// price-plan defaults and agreement snapshots.
type EscalationValue struct {
	Mode             *priceplanpb.EscalationMode
	Scope            *priceplanpb.EscalationScope
	RateBPS          *int32
	FirstAfterMonths *int32
	EveryMonths      *int32
}

var ErrInvalidEscalation = errors.New("invalid escalation clause")

// NormalizeAndValidateEscalation enforces the tuple stored by the database.
// A present UNSPECIFIED mode means an explicit request to clear the clause;
// NONE is recorded but clears every dependent value.
func NormalizeAndValidateEscalation(value *EscalationValue, billingKind priceplanpb.BillingKind, amountBasis priceplanpb.AmountBasis) error {
	if value == nil {
		return nil
	}
	if value.Mode == nil {
		if value.Scope != nil || value.RateBPS != nil || value.FirstAfterMonths != nil || value.EveryMonths != nil {
			return ErrInvalidEscalation
		}
		return nil
	}

	switch *value.Mode {
	case priceplanpb.EscalationMode_ESCALATION_MODE_UNSPECIFIED:
		value.Scope = nil
		value.RateBPS = nil
		value.FirstAfterMonths = nil
		value.EveryMonths = nil
		return nil
	case priceplanpb.EscalationMode_ESCALATION_MODE_NONE:
		value.Scope = nil
		value.RateBPS = nil
		value.FirstAfterMonths = nil
		value.EveryMonths = nil
		return nil
	case priceplanpb.EscalationMode_ESCALATION_MODE_FIXED_PERCENTAGE:
		if billingKind != priceplanpb.BillingKind_BILLING_KIND_RECURRING &&
			billingKind != priceplanpb.BillingKind_BILLING_KIND_CONTRACT {
			return ErrInvalidEscalation
		}
		if amountBasis != priceplanpb.AmountBasis_AMOUNT_BASIS_PER_CYCLE {
			return ErrInvalidEscalation
		}
		if value.Scope == nil || value.RateBPS == nil || *value.RateBPS < 1 || *value.RateBPS > 10000 {
			return ErrInvalidEscalation
		}
		switch *value.Scope {
		case priceplanpb.EscalationScope_ESCALATION_SCOPE_WITHIN_AGREEMENT:
			if value.FirstAfterMonths == nil || value.EveryMonths == nil ||
				*value.FirstAfterMonths < 1 || *value.FirstAfterMonths > 1200 ||
				*value.EveryMonths < 1 || *value.EveryMonths > 1200 {
				return ErrInvalidEscalation
			}
			return nil
		case priceplanpb.EscalationScope_ESCALATION_SCOPE_ON_RENEWAL:
			value.FirstAfterMonths = nil
			value.EveryMonths = nil
			return nil
		default:
			return ErrInvalidEscalation
		}
	default:
		return ErrInvalidEscalation
	}
}

func NormalizeAndValidatePricePlanEscalation(pricePlan *priceplanpb.PricePlan) error {
	if pricePlan == nil {
		return ErrInvalidEscalation
	}
	value := EscalationValue{
		Mode:             pricePlan.DefaultEscalationMode,
		Scope:            pricePlan.DefaultEscalationScope,
		RateBPS:          pricePlan.DefaultEscalationRateBps,
		FirstAfterMonths: pricePlan.DefaultEscalationFirstAfterMonths,
		EveryMonths:      pricePlan.DefaultEscalationEveryMonths,
	}
	if err := NormalizeAndValidateEscalation(&value, pricePlan.GetBillingKind(), pricePlan.GetAmountBasis()); err != nil {
		return err
	}
	pricePlan.DefaultEscalationScope = value.Scope
	pricePlan.DefaultEscalationRateBps = value.RateBPS
	pricePlan.DefaultEscalationFirstAfterMonths = value.FirstAfterMonths
	pricePlan.DefaultEscalationEveryMonths = value.EveryMonths
	return nil
}
