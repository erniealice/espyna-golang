package billable_charge

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/registry/entityid"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
)

// ReadBillableChargeUseCase reads one row by id (foreign-workspace and missing ids are both "not found").
type ReadBillableChargeUseCase struct {
	repos Repositories
	svc   Services
}

func (uc *ReadBillableChargeUseCase) Execute(ctx context.Context, req *billablechargepb.ReadBillableChargeRequest) (*billablechargepb.ReadBillableChargeResponse, error) {
	if err := uc.svc.gate(ctx, entityid.ActionRead); err != nil {
		return nil, err
	}
	if uc.repos.BillableCharge == nil {
		return nil, fmt.Errorf("billable_charge: repository unavailable")
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("billable_charge: id is required")
	}
	return uc.repos.BillableCharge.ReadBillableCharge(ctx, req)
}
