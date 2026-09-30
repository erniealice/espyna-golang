package allocation_batch

import (
	"context"

	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	allocationbatchpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_batch"
)

// CreateAllocationBatchUseCase creates the next revision as a DRAFT (allocation_batch:create).
// Request: Data.CostSourceComponentId + Shares (share_kind, subscription_id, client_id, basis
// weights). Under the component row lock it refuses a claimed source, numbers the revision
// (max+1), and stores the shares with their exact largest-remainder amounts. Response: Data[0] =
// the batch, Shares = the stored rows.
type CreateAllocationBatchUseCase struct{ c *writeCore }

func (uc *CreateAllocationBatchUseCase) Execute(ctx context.Context, req *allocationbatchpb.CreateAllocationBatchRequest) (*allocationbatchpb.CreateAllocationBatchResponse, error) {
	if err := uc.c.gate(ctx, entityid.ActionCreate); err != nil {
		return nil, err
	}
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		return nil, uc.c.refuse(ctx, codeValidation, "authenticated user required")
	}
	if req == nil || req.GetData().GetCostSourceComponentId() == "" {
		return nil, uc.c.refuse(ctx, codeValidation, "cost source component id is required")
	}
	if err := uc.c.ready(); err != nil {
		return nil, err
	}
	var out *allocationbatchpb.CreateAllocationBatchResponse
	err = uc.c.inTx(ctx, func(txCtx context.Context) error {
		comp, err := uc.c.lockComponent(txCtx, req.GetData().GetCostSourceComponentId())
		if err != nil {
			return err
		}
		if err := uc.c.requireUnclaimed(txCtx, comp); err != nil {
			return err
		}
		amounts, err := uc.c.validateShares(txCtx, comp.GetAmount(), req.GetShares())
		if err != nil {
			return err
		}
		if err := uc.c.verifyRecoverableTerms(txCtx, req.GetShares()); err != nil {
			return err
		}
		existing, err := uc.c.batchesOfComponent(txCtx, comp.GetId())
		if err != nil {
			return err
		}
		var revision int32
		for _, b := range existing {
			if b.GetRevision() > revision {
				revision = b.GetRevision()
			}
		}
		ms, ts := stamp()
		row := &allocationbatchpb.AllocationBatch{
			Id:                    uc.c.s.IDGenerator.GenerateID(),
			CostSourceComponentId: comp.GetId(),
			Revision:              revision + 1,
			Status:                allocationbatchpb.AllocationBatchStatus_ALLOCATION_BATCH_STATUS_DRAFT,
			RuleCode:              "LARGEST_REMAINDER",
			RuleVersion:           1,
			TotalAmount:           i64p(comp.GetAmount()),
			Currency:              strp(comp.GetCurrency()),
			PreparedBy:            strp(userID),
			Active:                true,
			DateCreated:           i64p(ms),
			DateCreatedString:     strp(ts),
			DateModified:          i64p(ms),
			DateModifiedString:    strp(ts),
		}
		resp, err := uc.c.r.AllocationBatch.CreateAllocationBatch(txCtx, &allocationbatchpb.CreateAllocationBatchRequest{Data: row})
		if err != nil {
			return err
		}
		if d := resp.GetData(); len(d) > 0 && d[0] != nil {
			row = d[0]
		}
		shares, err := uc.c.writeShares(txCtx, row.GetId(), req.GetShares(), amounts)
		if err != nil {
			return err
		}
		out = &allocationbatchpb.CreateAllocationBatchResponse{Data: []*allocationbatchpb.AllocationBatch{row}, Shares: shares, Success: true}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
