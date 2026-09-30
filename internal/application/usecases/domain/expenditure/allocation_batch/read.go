package allocation_batch

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/registry/entityid"
	allocationbatchpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_batch"
)

// ReadAllocationBatchUseCase reads one row by id (foreign-workspace and missing ids are both "not found").
type ReadAllocationBatchUseCase struct {
	repos Repositories
	svc   Services
}

func (uc *ReadAllocationBatchUseCase) Execute(ctx context.Context, req *allocationbatchpb.ReadAllocationBatchRequest) (*allocationbatchpb.ReadAllocationBatchResponse, error) {
	if err := uc.svc.gate(ctx, entityid.ActionRead); err != nil {
		return nil, err
	}
	if uc.repos.AllocationBatch == nil {
		return nil, fmt.Errorf("allocation_batch: repository unavailable")
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("allocation_batch: id is required")
	}
	return uc.repos.AllocationBatch.ReadAllocationBatch(ctx, req)
}
