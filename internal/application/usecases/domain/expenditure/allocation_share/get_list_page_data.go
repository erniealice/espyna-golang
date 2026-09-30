package allocation_share

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/registry/entityid"
	allocationsharepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_share"
)

// GetAllocationShareListPageDataUseCase returns a page of rows with pagination.
type GetAllocationShareListPageDataUseCase struct {
	repos Repositories
	svc   Services
}

func (uc *GetAllocationShareListPageDataUseCase) Execute(ctx context.Context, req *allocationsharepb.GetAllocationShareListPageDataRequest) (*allocationsharepb.GetAllocationShareListPageDataResponse, error) {
	if err := uc.svc.gate(ctx, entityid.ActionList); err != nil {
		return nil, err
	}
	if uc.repos.AllocationShare == nil {
		return nil, fmt.Errorf("allocation_share: repository unavailable")
	}
	if req == nil {
		req = &allocationsharepb.GetAllocationShareListPageDataRequest{}
	}
	return uc.repos.AllocationShare.GetAllocationShareListPageData(ctx, req)
}
