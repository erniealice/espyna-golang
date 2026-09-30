package billable_charge

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	"google.golang.org/protobuf/proto"
)

// ListBillableChargesUseCase lists rows of the caller's workspace (billable_charge:list). The
// request's optional Status narrows to OPEN | ISSUED | CANCELLED (an unset status lists all), and
// its Pagination is forwarded to the adapter, whose response carries the pagination back (C11);
// views that list charges page through it rather than relying on the adapter's default row cap.
type ListBillableChargesUseCase struct {
	repos Repositories
	svc   Services
}

func (uc *ListBillableChargesUseCase) Execute(ctx context.Context, req *billablechargepb.ListBillableChargesRequest) (*billablechargepb.ListBillableChargesResponse, error) {
	if err := uc.svc.gate(ctx, entityid.ActionList); err != nil {
		return nil, err
	}
	if uc.repos.BillableCharge == nil {
		return nil, fmt.Errorf("billable_charge: repository unavailable")
	}
	if req == nil {
		req = &billablechargepb.ListBillableChargesRequest{}
	}
	if req.Status != nil && req.GetStatus() != billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_UNSPECIFIED {
		req = proto.Clone(req).(*billablechargepb.ListBillableChargesRequest)
		if req.Filters == nil {
			req.Filters = &commonpb.FilterRequest{}
		}
		req.Filters.Filters = append(req.Filters.Filters, &commonpb.TypedFilter{
			Field: "status",
			FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
				Value: req.GetStatus().String(), Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true,
			}},
		})
	}
	return uc.repos.BillableCharge.ListBillableCharges(ctx, req)
}
