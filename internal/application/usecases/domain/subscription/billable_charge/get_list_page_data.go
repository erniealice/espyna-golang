package billable_charge

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/registry/entityid"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
)

// GetBillableChargeListPageDataUseCase returns a page of rows with pagination.
type GetBillableChargeListPageDataUseCase struct {
	repos Repositories
	svc   Services
}

func (uc *GetBillableChargeListPageDataUseCase) Execute(ctx context.Context, req *billablechargepb.GetBillableChargeListPageDataRequest) (*billablechargepb.GetBillableChargeListPageDataResponse, error) {
	if err := uc.svc.gate(ctx, entityid.ActionList); err != nil {
		return nil, err
	}
	if uc.repos.BillableCharge == nil {
		return nil, fmt.Errorf("billable_charge: repository unavailable")
	}
	if req == nil {
		req = &billablechargepb.GetBillableChargeListPageDataRequest{}
	}
	return uc.repos.BillableCharge.GetBillableChargeListPageData(ctx, req)
}
