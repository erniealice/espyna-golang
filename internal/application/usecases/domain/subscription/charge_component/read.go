package charge_component

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/registry/entityid"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
)

// ReadChargeComponentUseCase reads one row by id (foreign-workspace and missing ids are both "not found").
type ReadChargeComponentUseCase struct {
	repos Repositories
	svc   Services
}

func (uc *ReadChargeComponentUseCase) Execute(ctx context.Context, req *chargecomponentpb.ReadChargeComponentRequest) (*chargecomponentpb.ReadChargeComponentResponse, error) {
	if err := uc.svc.gate(ctx, entityid.ActionRead); err != nil {
		return nil, err
	}
	if uc.repos.ChargeComponent == nil {
		return nil, fmt.Errorf("charge_component: repository unavailable")
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("charge_component: id is required")
	}
	return uc.repos.ChargeComponent.ReadChargeComponent(ctx, req)
}
