package allocation_share

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/registry/entityid"
	allocationsharepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_share"
)

// ListAllocationSharesUseCase lists rows of the caller's workspace.
type ListAllocationSharesUseCase struct {
	repos Repositories
	svc   Services
}

func (uc *ListAllocationSharesUseCase) Execute(ctx context.Context, req *allocationsharepb.ListAllocationSharesRequest) (*allocationsharepb.ListAllocationSharesResponse, error) {
	if err := uc.svc.gate(ctx, entityid.ActionList); err != nil {
		return nil, err
	}
	if uc.repos.AllocationShare == nil {
		return nil, fmt.Errorf("allocation_share: repository unavailable")
	}
	if req == nil {
		req = &allocationsharepb.ListAllocationSharesRequest{}
	}
	return uc.repos.AllocationShare.ListAllocationShares(ctx, req)
}
