package allocation_batch

import (
	"context"

	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	allocationbatchpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_batch"
	allocationsharepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_share"
)

// UpdateAllocationBatchSharesUseCase replaces a DRAFT batch's shares (allocation_batch:update).
// A published or superseded batch is immutable (already_published / not_draft). Request:
// AllocationBatchId + Shares; response: Data = the batch, Shares = the stored rows.
type UpdateAllocationBatchSharesUseCase struct{ c *writeCore }

func (uc *UpdateAllocationBatchSharesUseCase) Execute(ctx context.Context, req *allocationbatchpb.UpdateAllocationBatchSharesRequest) (*allocationbatchpb.UpdateAllocationBatchSharesResponse, error) {
	if err := uc.c.gate(ctx, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if _, err := contextutil.RequireUserIDFromContext(ctx); err != nil {
		return nil, uc.c.refuse(ctx, codeValidation, "authenticated user required")
	}
	if req == nil || req.GetAllocationBatchId() == "" {
		return nil, uc.c.refuse(ctx, codeValidation, "allocation batch id is required")
	}
	if err := uc.c.ready(); err != nil {
		return nil, err
	}
	var out *allocationbatchpb.UpdateAllocationBatchSharesResponse
	err := uc.c.inTx(ctx, func(txCtx context.Context) error {
		peek, err := uc.c.peekBatch(txCtx, req.GetAllocationBatchId())
		if err != nil {
			return err
		}
		// Same lock order as publish: component first, then the batch.
		comp, err := uc.c.lockComponent(txCtx, peek.GetCostSourceComponentId())
		if err != nil {
			return err
		}
		batch, err := uc.c.lockBatch(txCtx, req.GetAllocationBatchId())
		if err != nil {
			return err
		}
		if err := uc.c.requireDraft(txCtx, batch); err != nil {
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
		old, err := uc.c.sharesOfBatch(txCtx, batch.GetId())
		if err != nil {
			return err
		}
		// Shares of a DRAFT batch are working rows; the adapter refuses deleting a share once its
		// batch is PUBLISHED (C6).
		for _, s := range old {
			if _, err := uc.c.r.AllocationShare.DeleteAllocationShare(txCtx, &allocationsharepb.DeleteAllocationShareRequest{Data: &allocationsharepb.AllocationShare{Id: s.GetId()}}); err != nil {
				return err
			}
		}
		shares, err := uc.c.writeShares(txCtx, batch.GetId(), req.GetShares(), amounts)
		if err != nil {
			return err
		}
		ms, ts := stamp()
		if _, err := uc.c.r.AllocationBatch.UpdateAllocationBatch(txCtx, &allocationbatchpb.UpdateAllocationBatchRequest{Data: &allocationbatchpb.AllocationBatch{
			Id: batch.GetId(), TotalAmount: i64p(comp.GetAmount()), Currency: strp(comp.GetCurrency()),
			DateModified: i64p(ms), DateModifiedString: strp(ts),
		}}); err != nil {
			return err
		}
		batch.TotalAmount, batch.Currency = i64p(comp.GetAmount()), strp(comp.GetCurrency())
		out = &allocationbatchpb.UpdateAllocationBatchSharesResponse{Data: batch, Shares: shares, Success: true}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
