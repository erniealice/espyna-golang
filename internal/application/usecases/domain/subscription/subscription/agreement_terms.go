package subscription

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	chargepolicypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
	agreementlinetermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/agreement_line_term"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// Agreement line terms on subscription create (20260927-usage-and-pass-through-charges, build-spec
// §6.1, AC-UC-02 / AC-UC-35): one COPIED term per package line that opted in to a charge policy,
// pinned to the policy's current APPROVED version (resolved by the charge_policy ResolveChargePolicy
// use case, never re-implemented here), effective from the subscription start. Everything runs INSIDE
// the create transaction (executeCoreWithTerms): a resolver refusal rolls the subscription back.
// Only RECURRING / CONTRACT price plans that are not TOTAL_PACKAGE can carry opted-in lines (the
// product_price_plan guard refuses them elsewhere), so every other create performs no extra read.

// ChargePolicyResolver is the charge_policy resolution rule WITHOUT its permission gate
// (ledger/charge_policy.ResolveChargePolicyUseCase.ResolveUngated satisfies it). Subscription
// create composes another domain's rule under its own subscription:create gate and must not
// inherit charge_policy:read (R4 m9). Injected by the aggregate (usecases.go); nil = a line that
// opted in cannot be verified and the create is refused.
type ChargePolicyResolver interface {
	ResolveUngated(ctx context.Context, req *chargepolicypb.ResolveChargePolicyRequest) (*chargepolicypb.ResolveChargePolicyResponse, error)
}

// Named refusals of the term step (C2): message translated from subscription.errors.<code>,
// ErrorCode() = <code>. Refusals of the resolver keep the resolver's own charge_policy.errors.<code>.
const (
	codeChargePolicyUnverifiable = "charge_policy_unverifiable"
	codeTransactionRequired      = "transaction_required"
)

var agreementTermFallbacks = map[string]string{
	codeChargePolicyUnverifiable: "The charge policies of this package cannot be verified.",
	codeTransactionRequired:      "A transaction is required to create the charge terms of this subscription.",
}

func (uc *CreateSubscriptionUseCase) refuseTerm(ctx context.Context, code string) error {
	return usecaseerr.New("", code, contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription.errors."+code, agreementTermFallbacks[code]))
}

// agreementTermSpec is a resolved, not yet persisted term.
type agreementTermSpec struct {
	productPricePlanID string
	versionID          string
	markupBps          *int32
}

// canCarryPolicyLines mirrors the product_price_plan guard: only RECURRING / CONTRACT price plans
// that are not TOTAL_PACKAGE can hold a line with a charge policy.
func canCarryPolicyLines(pp *priceplanpb.PricePlan) bool {
	if pp == nil || pp.GetAmountBasis() == priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE {
		return false
	}
	switch pp.GetBillingKind() {
	case priceplanpb.BillingKind_BILLING_KIND_RECURRING, priceplanpb.BillingKind_BILLING_KIND_CONTRACT:
		return true
	}
	return false
}

// resolveAgreementTerms resolves the opted-in lines of the subscription's price plan through the
// charge_policy resolver. It performs only reads (inside the create transaction) and returns
// (nil, nil) when the S1 collaborators are not wired, the plan cannot carry policy lines, or no
// line opted in. A failing line read or resolver refusal refuses the create (fail closed).
func (uc *CreateSubscriptionUseCase) resolveAgreementTerms(ctx context.Context, sub *subscriptionpb.Subscription, pricePlan *priceplanpb.PricePlan) ([]agreementTermSpec, error) {
	if uc.repositories.ProductPricePlan == nil || uc.repositories.AgreementLineTerm == nil || sub == nil || sub.GetPricePlanId() == "" || !canCarryPolicyLines(pricePlan) {
		return nil, nil
	}
	lines, err := uc.repositories.ProductPricePlan.ListProductPricePlans(ctx, &productpriceplanpb.ListProductPricePlansRequest{
		Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
			Field: "price_plan_id",
			FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
				Value: sub.GetPricePlanId(), Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true}},
		}}},
	})
	if err != nil {
		log.Printf("subscription: list package lines price_plan=%s: %v", sub.GetPricePlanId(), err)
		return nil, fmt.Errorf("subscription: list package lines: %w", err)
	}
	var specs []agreementTermSpec
	for _, l := range lines.GetData() {
		if l.GetPricePlanId() != sub.GetPricePlanId() || !l.GetActive() || l.GetChargePolicyId() == "" {
			continue
		}
		if uc.services.ChargePolicyResolver == nil {
			return nil, uc.refuseTerm(ctx, codeChargePolicyUnverifiable)
		}
		policyID := l.GetChargePolicyId()
		res, err := uc.services.ChargePolicyResolver.ResolveUngated(ctx, &chargepolicypb.ResolveChargePolicyRequest{PackageLineChargePolicyId: &policyID})
		if err != nil {
			return nil, err // a named charge_policy refusal (retired, no_approved_version, ...) or infrastructure error
		}
		if res == nil || res.GetVersion() == nil {
			return nil, uc.refuseTerm(ctx, codeChargePolicyUnverifiable)
		}
		spec := agreementTermSpec{productPricePlanID: l.GetId(), versionID: res.GetVersion().GetId()}
		if l.MarkupBps != nil {
			m := l.GetMarkupBps()
			spec.markupBps = &m
		}
		specs = append(specs, spec)
	}
	return specs, nil
}

// createAgreementTerms persists the resolved terms (call inside the create transaction).
func (uc *CreateSubscriptionUseCase) createAgreementTerms(ctx context.Context, sub *subscriptionpb.Subscription, specs []agreementTermSpec) error {
	if len(specs) == 0 {
		return nil
	}
	from := time.Now().UTC().Format("2006-01-02")
	if ts := sub.GetDateTimeStart(); ts != nil {
		from = ts.AsTime().UTC().Format("2006-01-02")
	}
	now := time.Now()
	ms, ts := now.UnixMilli(), now.Format(time.RFC3339)
	actor := contextutil.ExtractUserIDFromContext(ctx)
	for _, sp := range specs {
		term := &agreementlinetermpb.AgreementLineTerm{
			Id:                    uc.services.IDGenerator.GenerateID(),
			SubscriptionId:        sub.GetId(),
			ProductPricePlanId:    sp.productPricePlanID,
			ClientId:              sub.GetClientId(),
			ChargePolicyVersionId: sp.versionID,
			MarkupBps:             sp.markupBps,
			EffectiveFrom:         from,
			Origin:                agreementlinetermpb.AgreementLineTermOrigin_AGREEMENT_LINE_TERM_ORIGIN_COPIED,
			AcceptedAt:            &ms,
			Active:                true,
			DateCreated:           &ms,
			DateCreatedString:     &ts,
			DateModified:          &ms,
			DateModifiedString:    &ts,
		}
		if actor != "" {
			term.AcceptedBy = &actor
		}
		if _, err := uc.repositories.AgreementLineTerm.CreateAgreementLineTerm(ctx, &agreementlinetermpb.CreateAgreementLineTermRequest{Data: term}); err != nil {
			return fmt.Errorf("subscription: create agreement term: %w", err)
		}
	}
	return nil
}
