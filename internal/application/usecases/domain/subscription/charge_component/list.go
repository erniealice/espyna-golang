package charge_component

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/registry/entityid"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
)

// ListChargeComponentsUseCase lists rows of the caller's workspace.
type ListChargeComponentsUseCase struct {
	repos Repositories
	svc   Services
}

func (uc *ListChargeComponentsUseCase) Execute(ctx context.Context, req *chargecomponentpb.ListChargeComponentsRequest) (*chargecomponentpb.ListChargeComponentsResponse, error) {
	if err := uc.svc.gate(ctx, entityid.ActionList); err != nil {
		return nil, err
	}
	if uc.repos.ChargeComponent == nil {
		return nil, fmt.Errorf("charge_component: repository unavailable")
	}
	if req == nil {
		req = &chargecomponentpb.ListChargeComponentsRequest{}
	}
	return uc.repos.ChargeComponent.ListChargeComponents(ctx, req)
}
