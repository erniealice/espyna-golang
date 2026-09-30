package allocation_batch

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/registry/entityid"
	allocationbatchpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_batch"
)

// GetAllocationBatchListPageDataUseCase returns a page of rows with pagination.
type GetAllocationBatchListPageDataUseCase struct {
	repos Repositories
	svc   Services
}

func (uc *GetAllocationBatchListPageDataUseCase) Execute(ctx context.Context, req *allocationbatchpb.GetAllocationBatchListPageDataRequest) (*allocationbatchpb.GetAllocationBatchListPageDataResponse, error) {
	if err := uc.svc.gate(ctx, entityid.ActionList); err != nil {
		return nil, err
	}
	if uc.repos.AllocationBatch == nil {
		return nil, fmt.Errorf("allocation_batch: repository unavailable")
	}
	if req == nil {
		req = &allocationbatchpb.GetAllocationBatchListPageDataRequest{}
	}
	return uc.repos.AllocationBatch.GetAllocationBatchListPageData(ctx, req)
}
