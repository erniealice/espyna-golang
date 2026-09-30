package charge_component

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/registry/entityid"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
)

// GetChargeComponentListPageDataUseCase returns a page of rows with pagination.
type GetChargeComponentListPageDataUseCase struct {
	repos Repositories
	svc   Services
}

func (uc *GetChargeComponentListPageDataUseCase) Execute(ctx context.Context, req *chargecomponentpb.GetChargeComponentListPageDataRequest) (*chargecomponentpb.GetChargeComponentListPageDataResponse, error) {
	if err := uc.svc.gate(ctx, entityid.ActionList); err != nil {
		return nil, err
	}
	if uc.repos.ChargeComponent == nil {
		return nil, fmt.Errorf("charge_component: repository unavailable")
	}
	if req == nil {
		req = &chargecomponentpb.GetChargeComponentListPageDataRequest{}
	}
	return uc.repos.ChargeComponent.GetChargeComponentListPageData(ctx, req)
}
