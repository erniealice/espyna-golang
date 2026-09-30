package allocation_batch

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/registry/entityid"
	allocationbatchpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_batch"
)

// ListAllocationBatchesUseCase lists rows of the caller's workspace.
type ListAllocationBatchesUseCase struct {
	repos Repositories
	svc   Services
}

func (uc *ListAllocationBatchesUseCase) Execute(ctx context.Context, req *allocationbatchpb.ListAllocationBatchesRequest) (*allocationbatchpb.ListAllocationBatchesResponse, error) {
	if err := uc.svc.gate(ctx, entityid.ActionList); err != nil {
		return nil, err
	}
	if uc.repos.AllocationBatch == nil {
		return nil, fmt.Errorf("allocation_batch: repository unavailable")
	}
	if req == nil {
		req = &allocationbatchpb.ListAllocationBatchesRequest{}
	}
	return uc.repos.AllocationBatch.ListAllocationBatches(ctx, req)
}
