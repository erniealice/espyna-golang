package agreement_line_term

import (
	"context"
	"fmt"
	"sort"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	agreementlinetermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/agreement_line_term"
)

// Agreement-term rules shared by subscription create and allocation publish
// (20260927-usage-and-pass-through-charges, build-spec §6.1/§6.3). Intervals are half-open
// [effective_from, effective_to) ISO dates; an empty effective_to is open-ended.

// The refusals are usecaseerr.Error values (§7c C29) of a pure rule function: it carries ErrorCode() (C2) and an
// English message. The pure functions have no request context or Translator, so the calling use
// case (allocation publish, subscription create) re-issues the code as its own translated named
// refusal (<entity>.errors.<code>); the codes are agreement_term_missing, term_boundary_crossed and
// overlap. Nothing here is an exported sentinel; consumers read the code via
// interface{ ErrorCode() string }.

var (
	// errTermBoundaryCrossed: terms exist for the agreement but none covers the whole interval.
	errTermBoundaryCrossed = usecaseerr.New("", "term_boundary_crossed", "agreement_line_term: service interval is not covered by one term")
	// errTermMissing: the agreement has no charge term for this client at all.
	errTermMissing = usecaseerr.New("", "agreement_term_missing", "agreement_line_term: no charge term on this agreement")
)

// Covers reports whether the term's [from,to) contains the whole [from,to) interval.
func Covers(t *agreementlinetermpb.AgreementLineTerm, from, to string) bool {
	if t == nil || from == "" || to == "" || from >= to {
		return false
	}
	if t.GetEffectiveFrom() > from {
		return false
	}
	return t.GetEffectiveTo() == "" || to <= t.GetEffectiveTo()
}

// Overlaps reports whether two half-open term intervals intersect.
func Overlaps(a, b *agreementlinetermpb.AgreementLineTerm) bool {
	aEndsBeforeB := a.GetEffectiveTo() != "" && a.GetEffectiveTo() <= b.GetEffectiveFrom()
	bEndsBeforeA := b.GetEffectiveTo() != "" && b.GetEffectiveTo() <= a.GetEffectiveFrom()
	return !aEndsBeforeB && !bEndsBeforeA
}

// FindCoveringTerm picks, among the terms of `clientID`, the earliest (effective_from, id)
// term that covers [from,to). No term for the client -> ErrTermMissing; terms exist but none
// covers the whole interval (it crosses a term boundary) -> ErrTermBoundaryCrossed.
func FindCoveringTerm(terms []*agreementlinetermpb.AgreementLineTerm, clientID, from, to string) (*agreementlinetermpb.AgreementLineTerm, error) {
	var mine []*agreementlinetermpb.AgreementLineTerm
	for _, t := range terms {
		if t != nil && t.GetActive() && t.GetClientId() == clientID {
			mine = append(mine, t)
		}
	}
	if len(mine) == 0 {
		return nil, errTermMissing
	}
	sort.SliceStable(mine, func(i, j int) bool {
		if mine[i].GetEffectiveFrom() != mine[j].GetEffectiveFrom() {
			return mine[i].GetEffectiveFrom() < mine[j].GetEffectiveFrom()
		}
		return mine[i].GetId() < mine[j].GetId()
	})
	for _, t := range mine {
		if Covers(t, from, to) {
			return t, nil
		}
	}
	return nil, errTermBoundaryCrossed
}

// ListBySubscription lists the terms of one subscription through the (workspace-scoped) repository.
func ListBySubscription(ctx context.Context, repo agreementlinetermpb.AgreementLineTermDomainServiceServer, subscriptionID string) ([]*agreementlinetermpb.AgreementLineTerm, error) {
	if repo == nil {
		return nil, fmt.Errorf("agreement_line_term: repository unavailable")
	}
	resp, err := repo.ListAgreementLineTerms(ctx, &agreementlinetermpb.ListAgreementLineTermsRequest{
		Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
			Field: "subscription_id",
			FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
				Value: subscriptionID, Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true,
			}},
		}}},
	})
	if err != nil {
		return nil, err
	}
	out := make([]*agreementlinetermpb.AgreementLineTerm, 0, len(resp.GetData()))
	for _, t := range resp.GetData() {
		if t.GetSubscriptionId() == subscriptionID { // never trust an adapter that ignored the filter
			out = append(out, t)
		}
	}
	return out, nil
}
