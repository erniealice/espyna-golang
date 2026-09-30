package allocation_share

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/registry/entityid"
	allocationsharepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_share"
)

// ReadAllocationShareUseCase reads one row by id (foreign-workspace and missing ids are both "not found").
type ReadAllocationShareUseCase struct {
	repos Repositories
	svc   Services
}

func (uc *ReadAllocationShareUseCase) Execute(ctx context.Context, req *allocationsharepb.ReadAllocationShareRequest) (*allocationsharepb.ReadAllocationShareResponse, error) {
	if err := uc.svc.gate(ctx, entityid.ActionRead); err != nil {
		return nil, err
	}
	if uc.repos.AllocationShare == nil {
		return nil, fmt.Errorf("allocation_share: repository unavailable")
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("allocation_share: id is required")
	}
	return uc.repos.AllocationShare.ReadAllocationShare(ctx, req)
}
