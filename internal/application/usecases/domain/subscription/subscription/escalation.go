package subscription

import (
	"context"

	priceplanuc "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/price_plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

func agreementEscalationWasSupplied(s *subscriptionpb.Subscription) bool {
	return s != nil && (s.EscalationMode != nil || s.EscalationScope != nil ||
		s.EscalationRateBps != nil || s.EscalationFirstAfterMonths != nil || s.EscalationEveryMonths != nil)
}

func copyEscalationDefaultsOnce(s *subscriptionpb.Subscription, pp *priceplanpb.PricePlan) {
	if s == nil || pp == nil || agreementEscalationWasSupplied(s) || pp.DefaultEscalationMode == nil {
		return
	}
	mode := pp.GetDefaultEscalationMode()
	s.EscalationMode = &mode
	if pp.DefaultEscalationScope != nil {
		scope := pp.GetDefaultEscalationScope()
		s.EscalationScope = &scope
	}
	if pp.DefaultEscalationRateBps != nil {
		v := pp.GetDefaultEscalationRateBps()
		s.EscalationRateBps = &v
	}
	if pp.DefaultEscalationFirstAfterMonths != nil {
		v := pp.GetDefaultEscalationFirstAfterMonths()
		s.EscalationFirstAfterMonths = &v
	}
	if pp.DefaultEscalationEveryMonths != nil {
		v := pp.GetDefaultEscalationEveryMonths()
		s.EscalationEveryMonths = &v
	}
}

func normalizeAndValidateAgreementEscalation(s *subscriptionpb.Subscription, billingKind priceplanpb.BillingKind, amountBasis priceplanpb.AmountBasis) error {
	if s == nil {
		return priceplanuc.ErrInvalidEscalation
	}
	value := priceplanuc.EscalationValue{
		Mode:             s.EscalationMode,
		Scope:            s.EscalationScope,
		RateBPS:          s.EscalationRateBps,
		FirstAfterMonths: s.EscalationFirstAfterMonths,
		EveryMonths:      s.EscalationEveryMonths,
	}
	if err := priceplanuc.NormalizeAndValidateEscalation(&value, billingKind, amountBasis); err != nil {
		return err
	}
	s.EscalationScope = value.Scope
	s.EscalationRateBps = value.RateBPS
	s.EscalationFirstAfterMonths = value.FirstAfterMonths
	s.EscalationEveryMonths = value.EveryMonths
	return nil
}

func (uc *UpdateSubscriptionUseCase) effectiveEscalationPricing(ctx context.Context, s *subscriptionpb.Subscription) (priceplanpb.BillingKind, priceplanpb.AmountBasis) {
	if s == nil {
		return priceplanpb.BillingKind_BILLING_KIND_UNSPECIFIED, priceplanpb.AmountBasis_AMOUNT_BASIS_UNSPECIFIED
	}
	pricePlanID := s.GetPricePlanId()
	if pricePlanID == "" && s.GetId() != "" {
		if existing, err := uc.repositories.Subscription.ReadSubscription(ctx, &subscriptionpb.ReadSubscriptionRequest{Data: &subscriptionpb.Subscription{Id: s.GetId()}}); err == nil && len(existing.GetData()) > 0 {
			pricePlanID = existing.GetData()[0].GetPricePlanId()
		}
	}
	if pricePlanID == "" {
		return priceplanpb.BillingKind_BILLING_KIND_UNSPECIFIED, priceplanpb.AmountBasis_AMOUNT_BASIS_UNSPECIFIED
	}
	resp, err := uc.repositories.PricePlan.ReadPricePlan(ctx, &priceplanpb.ReadPricePlanRequest{Data: &priceplanpb.PricePlan{Id: pricePlanID}})
	if err != nil || len(resp.GetData()) == 0 {
		return priceplanpb.BillingKind_BILLING_KIND_UNSPECIFIED, priceplanpb.AmountBasis_AMOUNT_BASIS_UNSPECIFIED
	}
	return resp.GetData()[0].GetBillingKind(), resp.GetData()[0].GetAmountBasis()
}
