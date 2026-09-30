package cost_source_component

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
)

// ListCostSourceComponentsUseCase lists components; filter by expenditure_id for the
// expenditure detail "recoverable costs" tab. The request's Pagination is forwarded to the
// adapter (C11); callers that show the list page through it.
type ListCostSourceComponentsUseCase struct{ c *core }

func (uc *ListCostSourceComponentsUseCase) Execute(ctx context.Context, req *pb.ListCostSourceComponentsRequest) (*pb.ListCostSourceComponentsResponse, error) {
	if err := uc.c.gate(ctx, entityid.ActionRead); err != nil {
		return nil, err
	}
	if uc.c.repos.CostSourceComponent == nil {
		return nil, fmt.Errorf("cost_source_component: repository unavailable")
	}
	if req == nil {
		req = &pb.ListCostSourceComponentsRequest{}
	}
	return uc.c.repos.CostSourceComponent.ListCostSourceComponents(ctx, req)
}
