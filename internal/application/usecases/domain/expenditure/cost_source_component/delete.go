package cost_source_component

import (
	"context"

	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
)

// DeleteCostSourceComponentUseCase hard-deletes an UNCLAIMED component under its row lock
// (the component is source data, not an issued financial document; a claimed one is immutable).
type DeleteCostSourceComponentUseCase struct{ c *core }

func (uc *DeleteCostSourceComponentUseCase) Execute(ctx context.Context, req *pb.DeleteCostSourceComponentRequest) (*pb.DeleteCostSourceComponentResponse, error) {
	if err := uc.c.gate(ctx, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || blank(req.Data.GetId()) {
		return nil, uc.c.refuse(ctx, codeValidation, "id is required")
	}
	var out *pb.DeleteCostSourceComponentResponse
	err := uc.c.inTx(ctx, func(txCtx context.Context) error {
		cur, err := uc.c.lock(txCtx, req.Data.GetId())
		if err != nil {
			return err
		}
		if cur.ClaimKind != nil {
			return uc.c.refuse(txCtx, codeClaimed)
		}
		out, err = uc.c.repos.CostSourceComponent.DeleteCostSourceComponent(txCtx, &pb.DeleteCostSourceComponentRequest{Data: &pb.CostSourceComponent{Id: cur.GetId()}})
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
