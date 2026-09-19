package price_plan

import (
	"testing"

	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
)

func enumPtr[T ~int32](v T) *T { return &v }
func int32Ptr(v int32) *int32  { return &v }

func TestEscalationValueMatrix(t *testing.T) {
	fixed := priceplanpb.EscalationMode_ESCALATION_MODE_FIXED_PERCENTAGE
	within := priceplanpb.EscalationScope_ESCALATION_SCOPE_WITHIN_AGREEMENT
	renewal := priceplanpb.EscalationScope_ESCALATION_SCOPE_ON_RENEWAL
	none := priceplanpb.EscalationMode_ESCALATION_MODE_NONE

	tests := []struct {
		name    string
		value   EscalationValue
		kind    priceplanpb.BillingKind
		basis   priceplanpb.AmountBasis
		wantErr bool
	}{
		{name: "legacy absent", kind: priceplanpb.BillingKind_BILLING_KIND_ONE_TIME},
		{name: "none clears dependents", value: EscalationValue{Mode: &none, Scope: &within, RateBPS: int32Ptr(500)}, kind: priceplanpb.BillingKind_BILLING_KIND_ONE_TIME},
		{name: "within agreement", value: EscalationValue{Mode: &fixed, Scope: &within, RateBPS: int32Ptr(500), FirstAfterMonths: int32Ptr(12), EveryMonths: int32Ptr(12)}, kind: priceplanpb.BillingKind_BILLING_KIND_CONTRACT, basis: priceplanpb.AmountBasis_AMOUNT_BASIS_PER_CYCLE},
		{name: "renewal", value: EscalationValue{Mode: &fixed, Scope: &renewal, RateBPS: int32Ptr(500)}, kind: priceplanpb.BillingKind_BILLING_KIND_RECURRING, basis: priceplanpb.AmountBasis_AMOUNT_BASIS_PER_CYCLE},
		{name: "partial fixed", value: EscalationValue{Mode: &fixed, Scope: &within, RateBPS: int32Ptr(500)}, kind: priceplanpb.BillingKind_BILLING_KIND_CONTRACT, basis: priceplanpb.AmountBasis_AMOUNT_BASIS_PER_CYCLE, wantErr: true},
		{name: "out of range", value: EscalationValue{Mode: &fixed, Scope: &renewal, RateBPS: int32Ptr(10001)}, kind: priceplanpb.BillingKind_BILLING_KIND_CONTRACT, basis: priceplanpb.AmountBasis_AMOUNT_BASIS_PER_CYCLE, wantErr: true},
		{name: "unsupported billing kind", value: EscalationValue{Mode: &fixed, Scope: &renewal, RateBPS: int32Ptr(500)}, kind: priceplanpb.BillingKind_BILLING_KIND_ONE_TIME, basis: priceplanpb.AmountBasis_AMOUNT_BASIS_PER_CYCLE, wantErr: true},
		{name: "unsupported amount basis", value: EscalationValue{Mode: &fixed, Scope: &renewal, RateBPS: int32Ptr(500)}, kind: priceplanpb.BillingKind_BILLING_KIND_CONTRACT, basis: priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := NormalizeAndValidateEscalation(&tc.value, tc.kind, tc.basis)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.name == "none clears dependents" && (tc.value.Scope != nil || tc.value.RateBPS != nil) {
				t.Fatal("NONE did not clear dependent values")
			}
		})
	}
}
