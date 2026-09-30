package product_price_plan

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	chargepolicypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
	chargepolicyversionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
)

// Charge policy opt-in guard (20260927-usage-and-pass-through-charges, build-spec §3
// "Package-line guard", AC-CP-06 / AC-UC-35 / AC-UC-11).
//
// Refusals are named errors (C2): the message is Translator-translated from
// `product_price_plan.errors.<code>` and ErrorCode() returns the stable <code>, so a presentation
// layer maps the refusal to a localized label without importing this package (it matches the
// anonymous interface `interface{ ErrorCode() string }` via errors.As). There are no exported
// sentinels.

// Stable guard refusal codes.
const (
	guardCodeNotUsageBased    = "charge_policy_not_usage_based"
	guardCodeBillingKind      = "charge_policy_billing_kind"
	guardCodeUnavailable      = "charge_policy_unavailable"
	guardCodeNotFound         = "charge_policy_not_found"
	guardCodeMarkupNotAllowed = "charge_policy_markup_not_allowed"
	guardCodeUnverifiable     = "charge_policy_unverifiable"
)

var guardDefaults = map[string]string{
	guardCodeNotUsageBased:    "A charge policy requires the USAGE_BASED billing treatment",
	guardCodeBillingKind:      "A charge policy requires a RECURRING or CONTRACT price plan",
	guardCodeUnavailable:      "The charge policy is not active or has no approved version",
	guardCodeNotFound:         "The charge policy was not found in this workspace",
	guardCodeMarkupNotAllowed: "Markup must be empty or 0 (at-cost recovery only)",
	guardCodeUnverifiable:     "The charge policy cannot be verified (repositories not wired)",
}

type guardError struct {
	code string
	msg  string
}

func (e *guardError) Error() string     { return e.msg }
func (e *guardError) ErrorCode() string { return e.code }

// GuardRefusal builds the translated named refusal for a guard code (also used by the price_plan
// update guard so both edit surfaces speak one convention).
func GuardRefusal(ctx context.Context, tr ports.Translator, entity, code, detail string) error {
	msg := contextutil.GetTranslatedMessageWithContext(ctx, tr, entity+".errors."+code, guardDefaults[code])
	if detail != "" {
		msg += ": " + detail
	}
	return &guardError{code: code, msg: msg}
}

// GuardCodeBillingKind and GuardCodeUnverifiable are the refusal codes shared with the price_plan
// update guard.
const (
	GuardCodeBillingKind  = guardCodeBillingKind
	GuardCodeUnverifiable = guardCodeUnverifiable
)

// IsGuardCode reports whether err carries the given guard refusal code.
func IsGuardCode(err error, code string) bool {
	var ce interface{ ErrorCode() string }
	return errors.As(err, &ce) && ce.ErrorCode() == code
}

// PricePlanAllowsChargePolicy is the single source of the "billing kind" rule: a charge policy
// may be attached only to lines of a RECURRING or CONTRACT price plan that is not TOTAL_PACKAGE
// (ONE_TIME, MILESTONE, AD_HOC and unspecified kinds are refused).
func PricePlanAllowsChargePolicy(plan *priceplanpb.PricePlan) bool {
	if plan.GetAmountBasis() == priceplanpb.AmountBasis_AMOUNT_BASIS_TOTAL_PACKAGE {
		return false
	}
	switch plan.GetBillingKind() {
	case priceplanpb.BillingKind_BILLING_KIND_RECURRING, priceplanpb.BillingKind_BILLING_KIND_CONTRACT:
		return true
	}
	return false
}

// isRepoNotFound reports the repository "row absent (or foreign workspace)" outcome.
func isRepoNotFound(err error) bool {
	return IsRepoNotFound(err)
}

// IsRepoNotFound classifies a repository not-found outcome ONLY by the locker sentinel or the
// database layer's exact "record not found" message (C9: never Contains("not found"); any other
// message is an infrastructure failure). Shared with the price_plan update guard.
func IsRepoNotFound(err error) bool {
	return err != nil && (errors.Is(err, domainports.ErrLockedRowNotFound) ||
		strings.Contains(strings.ToLower(err.Error()), "record not found"))
}

// guardRepoErr logs a repository failure (operation + ids only) and returns it wrapped: the guard
// fails CLOSED on a read error and never reports it as a named "not found" (C9/C12).
func guardRepoErr(op string, err error, ids ...string) error {
	log.Printf("product_price_plan: %s failed (ids=%v): %v", op, ids, err)
	return fmt.Errorf("product_price_plan: %s: %w", op, err)
}

// chargePolicyDeps are the optional collaborators of the guard. Without them a line that sets
// charge_policy_id is refused (fail closed); lines without a policy are never affected.
type chargePolicyDeps struct {
	ChargePolicy        chargepolicypb.ChargePolicyDomainServiceServer
	ChargePolicyVersion chargepolicyversionpb.ChargePolicyVersionDomainServiceServer
	PricePlan           priceplanpb.PricePlanDomainServiceServer
}

// validateChargePolicyOptIn checks the EFFECTIVE (merged) line. It is a no-op when the line has
// no charge_policy_id and markup_bps is null/0, so the legacy path is byte-identical.
func validateChargePolicyOptIn(ctx context.Context, d chargePolicyDeps, line *productpriceplanpb.ProductPricePlan, tr ports.Translator) error {
	refuse := func(code string) error { return GuardRefusal(ctx, tr, "product_price_plan", code, "") }
	if line.MarkupBps != nil && line.GetMarkupBps() != 0 {
		// While the S1 matrix only allows at-cost recovery, any non-zero markup is refused
		// (negative values are additionally blocked by the DB CHECK).
		return refuse(guardCodeMarkupNotAllowed)
	}
	policyID := line.GetChargePolicyId()
	if policyID == "" {
		return nil
	}
	if d.ChargePolicy == nil || d.ChargePolicyVersion == nil || d.PricePlan == nil {
		return refuse(guardCodeUnverifiable)
	}
	if line.GetBillingTreatment() != productpriceplanpb.BillingTreatment_BILLING_TREATMENT_USAGE_BASED {
		return refuse(guardCodeNotUsageBased)
	}
	// Serialize with a concurrent parent price_plan billing_kind / amount_basis write: both sides
	// take the price plan's lock, so the check below cannot race an opposite change (C12). A repo
	// that cannot lock, or a call outside a transaction, refuses (fail closed, C4).
	locker, canLock := d.PricePlan.(domainports.PricePlanLocker)
	if !canLock {
		return refuse(guardCodeUnverifiable)
	}
	if err := locker.LockPricePlanForUpdate(ctx, line.GetPricePlanId()); err != nil {
		if !isRepoNotFound(err) {
			return guardRepoErr("lock price_plan", err, line.GetPricePlanId())
		}
		return GuardRefusal(ctx, tr, "product_price_plan", guardCodeBillingKind, "parent price plan "+line.GetPricePlanId())
	}
	pp, err := d.PricePlan.ReadPricePlan(ctx, &priceplanpb.ReadPricePlanRequest{Data: &priceplanpb.PricePlan{Id: line.GetPricePlanId()}})
	if err != nil {
		if !isRepoNotFound(err) {
			return guardRepoErr("read price_plan", err, line.GetPricePlanId())
		}
		return GuardRefusal(ctx, tr, "product_price_plan", guardCodeBillingKind, "parent price plan "+line.GetPricePlanId())
	}
	if pp == nil || len(pp.Data) == 0 {
		return GuardRefusal(ctx, tr, "product_price_plan", guardCodeBillingKind, "parent price plan "+line.GetPricePlanId())
	}
	if !PricePlanAllowsChargePolicy(pp.Data[0]) {
		return refuse(guardCodeBillingKind)
	}
	// Workspace-scoped read: a foreign or missing policy is reported as not found.
	pol, err := d.ChargePolicy.ReadChargePolicy(ctx, &chargepolicypb.ReadChargePolicyRequest{Data: &chargepolicypb.ChargePolicy{Id: policyID}})
	if err != nil {
		if !isRepoNotFound(err) {
			return guardRepoErr("read charge_policy", err, policyID)
		}
		return refuse(guardCodeNotFound)
	}
	if pol == nil || len(pol.Data) == 0 {
		return refuse(guardCodeNotFound)
	}
	if pol.Data[0].GetStatus() != enumspb.ChargePolicyStatus_CHARGE_POLICY_STATUS_ACTIVE {
		return refuse(guardCodeUnavailable)
	}
	// Scope to the policy AND its APPROVED status so a single row answers the question (no page cap).
	approved := enumspb.ChargePolicyVersionStatus_CHARGE_POLICY_VERSION_STATUS_APPROVED.String()
	vs, err := d.ChargePolicyVersion.ListChargePolicyVersions(ctx, &chargepolicyversionpb.ListChargePolicyVersionsRequest{
		ChargePolicyId: &policyID,
		Filters: &commonpb.FilterRequest{Logic: commonpb.FilterLogic_AND, Filters: []*commonpb.TypedFilter{
			eqTyped("charge_policy_id", policyID),
			eqTyped("status", approved),
		}},
		Pagination: &commonpb.PaginationRequest{Limit: 1},
	})
	if err != nil {
		return guardRepoErr("list charge_policy_version", err, policyID)
	}
	for _, v := range vs.GetData() {
		if v.GetStatus() == enumspb.ChargePolicyVersionStatus_CHARGE_POLICY_VERSION_STATUS_APPROVED {
			return nil
		}
	}
	return refuse(guardCodeUnavailable)
}

func eqTyped(field, value string) *commonpb.TypedFilter {
	return &commonpb.TypedFilter{
		Field: field,
		FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
			Value: value, Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true,
		}},
	}
}
