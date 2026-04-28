package revenue

import (
	"testing"

	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
)

// helper: int32-pointer literal
func i32p(v int32) *int32 { return &v }

// 1. ONE_TIME × PER_CYCLE → coerced to TOTAL_PACKAGE.
func TestNormalize_OneTimePerCycle_Coerced(t *testing.T) {
	pp := &priceplanpb.PricePlan{
		BillingKind: priceplanpb.BillingKind_BILLING_KIND_ONE_TIME,
		AmountBasis: priceplanpb.AmountBasis_AMOUNT_BASIS_PER_CYCLE,
	}
	normalizePricePlan(pp)
	if pp.GetAmountBasis() != priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE {
		t.Errorf("expected coercion to TOTAL_PACKAGE, got %v", pp.GetAmountBasis())
	}
}

// 2. RECURRING × TOTAL_PACKAGE → coerced to PER_CYCLE.
func TestNormalize_RecurringTotalPackage_Coerced(t *testing.T) {
	pp := &priceplanpb.PricePlan{
		BillingKind: priceplanpb.BillingKind_BILLING_KIND_RECURRING,
		AmountBasis: priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE,
	}
	normalizePricePlan(pp)
	if pp.GetAmountBasis() != priceplanpb.AmountBasis_AMOUNT_BASIS_PER_CYCLE {
		t.Errorf("expected coercion to PER_CYCLE, got %v", pp.GetAmountBasis())
	}
}

// 3. ONE_TIME × TOTAL_PACKAGE with stale billing_cycle → cycle cleared.
func TestNormalize_OneTimeTotalPackage_ClearsCycle(t *testing.T) {
	pp := &priceplanpb.PricePlan{
		BillingKind:       priceplanpb.BillingKind_BILLING_KIND_ONE_TIME,
		AmountBasis:       priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE,
		BillingCycleValue: i32p(1),
		BillingCycleUnit:  stringPtrTest("month"),
	}
	normalizePricePlan(pp)
	if pp.BillingCycleValue != nil {
		t.Errorf("expected billing_cycle_value cleared, got %v", *pp.BillingCycleValue)
	}
	if pp.BillingCycleUnit != nil {
		t.Errorf("expected billing_cycle_unit cleared, got %q", *pp.BillingCycleUnit)
	}
}

// 4. RECURRING × PER_CYCLE (open-ended) with stale default_term → term cleared.
func TestNormalize_RecurringPerCycle_ClearsTerm(t *testing.T) {
	pp := &priceplanpb.PricePlan{
		BillingKind:      priceplanpb.BillingKind_BILLING_KIND_RECURRING,
		AmountBasis:      priceplanpb.AmountBasis_AMOUNT_BASIS_PER_CYCLE,
		DefaultTermValue: i32p(12),
		DefaultTermUnit:  stringPtrTest("month"),
	}
	normalizePricePlan(pp)
	if pp.DefaultTermValue != nil {
		t.Errorf("expected default_term_value cleared, got %v", *pp.DefaultTermValue)
	}
	if pp.DefaultTermUnit != nil {
		t.Errorf("expected default_term_unit cleared, got %q", *pp.DefaultTermUnit)
	}
}

// 5. Coherent CONTRACT × PER_CYCLE preserves cycle + term.
func TestNormalize_ContractPerCycle_Preserves(t *testing.T) {
	pp := &priceplanpb.PricePlan{
		BillingKind:       priceplanpb.BillingKind_BILLING_KIND_CONTRACT,
		AmountBasis:       priceplanpb.AmountBasis_AMOUNT_BASIS_PER_CYCLE,
		BillingCycleValue: i32p(1),
		BillingCycleUnit:  stringPtrTest("month"),
		DefaultTermValue:  i32p(60),
		DefaultTermUnit:   stringPtrTest("month"),
	}
	normalizePricePlan(pp)
	if pp.GetBillingCycleValue() != 1 || pp.GetBillingCycleUnit() != "month" {
		t.Error("expected cycle preserved on CONTRACT × PER_CYCLE")
	}
	if pp.GetDefaultTermValue() != 60 || pp.GetDefaultTermUnit() != "month" {
		t.Error("expected term preserved on CONTRACT × PER_CYCLE")
	}
}

// 6. nil PricePlan does not panic.
func TestNormalize_Nil_Safe(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("expected no panic on nil, got %v", r)
		}
	}()
	normalizePricePlan(nil)
}
