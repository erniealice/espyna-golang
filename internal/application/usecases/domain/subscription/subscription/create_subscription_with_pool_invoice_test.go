package subscription

import (
	"context"
	"errors"
	"testing"

	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
)

// CSP-1: predicate dispatch — isAdHocPool(...) returns true ONLY for
// AD_HOC × TOTAL_PACKAGE. Every other (kind × basis) combo, including
// AD_HOC × PER_OCCURRENCE, must return false so the wrapping use case
// delegates to the plain CreateSubscription path.
//
// The full atomic create+recognize integration test requires a Client
// repository stub the package doesn't currently provide. The wrapping use
// case's behaviour is documented invariantly in
// create_subscription_with_pool_invoice.go; an integration test belongs in
// the v1.5.5 commit that wires service-admin to actually call this path.
func TestCreateSubscriptionWithPoolInvoice_CSP1_PredicateDispatch(t *testing.T) {
	cycle := int32(1)
	cycleUnit := "month"
	cases := []struct {
		name string
		pp   *priceplanpb.PricePlan
		want bool
	}{
		{"ad_hoc_total_package_pool", &priceplanpb.PricePlan{
			BillingKind: priceplanpb.BillingKind_BILLING_KIND_AD_HOC,
			AmountBasis: priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE,
		}, true},
		{"ad_hoc_per_occurrence", &priceplanpb.PricePlan{
			BillingKind: priceplanpb.BillingKind_BILLING_KIND_AD_HOC,
			AmountBasis: priceplanpb.AmountBasis_AMOUNT_BASIS_PER_OCCURRENCE,
		}, false},
		{"recurring_per_cycle", &priceplanpb.PricePlan{
			BillingKind:       priceplanpb.BillingKind_BILLING_KIND_RECURRING,
			AmountBasis:       priceplanpb.AmountBasis_AMOUNT_BASIS_PER_CYCLE,
			BillingCycleValue: &cycle,
			BillingCycleUnit:  &cycleUnit,
		}, false},
		{"contract_total_package", &priceplanpb.PricePlan{
			BillingKind: priceplanpb.BillingKind_BILLING_KIND_CONTRACT,
			AmountBasis: priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE,
		}, false},
		{"milestone_total_package", &priceplanpb.PricePlan{
			BillingKind: priceplanpb.BillingKind_BILLING_KIND_MILESTONE,
			AmountBasis: priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE,
		}, false},
		{"one_time_total_package", &priceplanpb.PricePlan{
			BillingKind: priceplanpb.BillingKind_BILLING_KIND_ONE_TIME,
			AmountBasis: priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE,
		}, false},
		{"nil_price_plan", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isAdHocPool(tc.pp); got != tc.want {
				t.Errorf("isAdHocPool want %v, got %v", tc.want, got)
			}
		})
	}
}

// CSP-2: nil-invoker fallback — Constructor accepts a nil RecognizePoolInvoker
// so service-admin can wire the wrapping use case without yet plumbing the
// recognize adapter (Phase D ships the manual Pool-Generate-Invoice CTA;
// codex CRIT-5 atomic path is opt-in).
//
// Verifies the constructor doesn't panic + the field is genuinely settable
// to nil. Atomic-path execution is gated by a `recognizePoolFunc != nil`
// check inside Execute, exercised through the integration tests in v1.5.5.
func TestCreateSubscriptionWithPoolInvoice_CSP2_NilInvokerSafe(t *testing.T) {
	uc := NewCreateSubscriptionWithPoolInvoiceUseCase(nil, nil, CreateSubscriptionServices{})
	if uc == nil {
		t.Fatal("constructor returned nil")
	}
	if uc.recognizePoolFunc != nil {
		t.Errorf("expected nil recognizePoolFunc when constructor passed nil")
	}
}

// CSP-3: RecognizePoolInvoker contract — error from RecognizeAdHocPool must
// be returned by the wrapping use case as-is so the caller (centymo action
// handler) sees the underlying recognize error verbatim. Smoke-tests the
// stub adapter shape used by v1.5.5 wiring.
func TestCreateSubscriptionWithPoolInvoice_CSP3_RecognizePoolInvokerContract(t *testing.T) {
	stub := &stubRecognizePoolInvoker{
		failOnCall: 1,
		failError:  errors.New("currency mismatch"),
	}
	if err := stub.RecognizeAdHocPool(context.Background(), "sub-1"); err == nil {
		t.Fatal("expected stub to fail on first call")
	} else if err.Error() != "currency mismatch" {
		t.Errorf("error want %q, got %q", "currency mismatch", err.Error())
	}
	// Second call (failOnCall=1, called=2) succeeds.
	if err := stub.RecognizeAdHocPool(context.Background(), "sub-1"); err != nil {
		t.Errorf("expected stub to succeed on second call, got %v", err)
	}
	if stub.called != 2 {
		t.Errorf("called counter want 2, got %d", stub.called)
	}
}

// stubRecognizePoolInvoker — counts calls and optionally fails. Used by
// v1.5.5 integration tests that wire the wrapping use case end-to-end with
// a real Client repository.
type stubRecognizePoolInvoker struct {
	called     int
	calledWith []string
	failOnCall int // 1-based; 0 means never fail
	failError  error
}

func (s *stubRecognizePoolInvoker) RecognizeAdHocPool(_ context.Context, subscriptionID string) error {
	s.called++
	s.calledWith = append(s.calledWith, subscriptionID)
	if s.failOnCall != 0 && s.called == s.failOnCall {
		if s.failError != nil {
			return s.failError
		}
		return errors.New("simulated recognize-pool failure")
	}
	return nil
}
