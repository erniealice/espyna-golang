package subscription

import (
	"testing"

	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

func TestEscalationCopyOnceAndExplicitNone(t *testing.T) {
	fixed := priceplanpb.EscalationMode_ESCALATION_MODE_FIXED_PERCENTAGE
	none := priceplanpb.EscalationMode_ESCALATION_MODE_NONE
	within := priceplanpb.EscalationScope_ESCALATION_SCOPE_WITHIN_AGREEMENT
	rate, first, every := int32(500), int32(12), int32(12)
	defaults := &priceplanpb.PricePlan{
		DefaultEscalationMode:             &fixed,
		DefaultEscalationScope:            &within,
		DefaultEscalationRateBps:          &rate,
		DefaultEscalationFirstAfterMonths: &first,
		DefaultEscalationEveryMonths:      &every,
	}

	agreement := &subscriptionpb.Subscription{}
	copyEscalationDefaultsOnce(agreement, defaults)
	if agreement.GetEscalationMode() != fixed || agreement.GetEscalationRateBps() != 500 {
		t.Fatalf("defaults not copied: %+v", agreement)
	}

	changedRate := int32(700)
	defaults.DefaultEscalationRateBps = &changedRate
	copyEscalationDefaultsOnce(agreement, defaults)
	if agreement.GetEscalationRateBps() != 500 {
		t.Fatal("existing agreement was silently recopied")
	}

	explicitNone := &subscriptionpb.Subscription{EscalationMode: &none}
	copyEscalationDefaultsOnce(explicitNone, defaults)
	if explicitNone.GetEscalationMode() != none {
		t.Fatal("explicit NONE did not win")
	}
}
