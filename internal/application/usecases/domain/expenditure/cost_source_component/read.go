package cost_source_component

import (
	"context"

	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
)

// ReadCostSourceComponentUseCase reads one row by id (expenditure:read). The repository response
// is returned as-is: a foreign-workspace or missing id fails inside the workspace-aware adapter.
type ReadCostSourceComponentUseCase struct{ c *core }

func (uc *ReadCostSourceComponentUseCase) Execute(ctx context.Context, req *pb.ReadCostSourceComponentRequest) (*pb.ReadCostSourceComponentResponse, error) {
	if err := uc.c.gate(ctx, entityid.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || blank(req.Data.GetId()) {
		return nil, uc.c.refuse(ctx, codeValidation, "id is required")
	}
	if uc.c.repos.CostSourceComponent == nil {
		return nil, uc.c.refuse(ctx, codeValidation, "repository unavailable")
	}
	return uc.c.repos.CostSourceComponent.ReadCostSourceComponent(ctx, req)
}
