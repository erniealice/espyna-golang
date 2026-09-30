package agreement_line_term

import (
	"context"
	"fmt"
	"sort"

	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	agreementlinetermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/agreement_line_term"
	"google.golang.org/protobuf/proto"
)

// ListAgreementLineTermsUseCase lists rows of the caller's workspace (subscription:read). With
// the request's subscription_id it backs the subscription "Charge terms" tab: the list is
// filtered to that subscription and ordered by effective_from, then id (the tab pages through the
// request's Pagination; the adapter returns it in the response, C11).
type ListAgreementLineTermsUseCase struct {
	repos Repositories
	svc   Services
}

func (uc *ListAgreementLineTermsUseCase) Execute(ctx context.Context, req *agreementlinetermpb.ListAgreementLineTermsRequest) (*agreementlinetermpb.ListAgreementLineTermsResponse, error) {
	if err := uc.svc.gate(ctx, entityid.ActionRead); err != nil {
		return nil, err
	}
	if uc.repos.AgreementLineTerm == nil {
		return nil, fmt.Errorf("agreement_line_term: repository unavailable")
	}
	if req == nil {
		req = &agreementlinetermpb.ListAgreementLineTermsRequest{}
	}
	subscriptionID := req.GetSubscriptionId()
	if req.SubscriptionId != nil {
		if subscriptionID == "" {
			return nil, fmt.Errorf("agreement_line_term: subscription id must not be blank")
		}
		req = proto.Clone(req).(*agreementlinetermpb.ListAgreementLineTermsRequest)
		if req.Filters == nil {
			req.Filters = &commonpb.FilterRequest{}
		}
		req.Filters.Filters = append(req.Filters.Filters, &commonpb.TypedFilter{
			Field: "subscription_id",
			FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
				Value: subscriptionID, Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true,
			}},
		})
		if req.Sort == nil {
			req.Sort = &commonpb.SortRequest{Fields: []*commonpb.SortField{
				{Field: "effective_from", Direction: commonpb.SortDirection_ASC},
				{Field: "id", Direction: commonpb.SortDirection_ASC},
			}}
		}
	}
	resp, err := uc.repos.AgreementLineTerm.ListAgreementLineTerms(ctx, req)
	if err != nil || subscriptionID == "" {
		return resp, err
	}
	// Never trust an adapter that ignored the filter; keep the documented order within the page.
	kept := resp.Data[:0]
	for _, t := range resp.GetData() {
		if t.GetSubscriptionId() == subscriptionID {
			kept = append(kept, t)
		}
	}
	resp.Data = kept
	sort.SliceStable(resp.Data, func(i, j int) bool {
		if resp.Data[i].GetEffectiveFrom() != resp.Data[j].GetEffectiveFrom() {
			return resp.Data[i].GetEffectiveFrom() < resp.Data[j].GetEffectiveFrom()
		}
		return resp.Data[i].GetId() < resp.Data[j].GetId()
	})
	return resp, nil
}
