package cost_source_component

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	allocationbatchpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_batch"
	allocationsharepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_share"
	costsourcecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
)

// ReconcileCostSourceUseCase is the read-only ReconcileCostSource RPC of
// CostSourceComponentDomainService (build-spec §6.3; moved from ledger/recovery_reporting, §7c C27).
// Per component it checks that amount = Σ shares (all kinds) and that Σ recoverable shares = Σ
// original charges; corrections are reported separately (net). Permission: expenditure:read.
// It needs the AllocationBatch, AllocationShare and BillableCharge repositories (fail closed when any
// is missing).
type ReconcileCostSourceUseCase struct{ c *core }

// Execute reconciles one component (component_id) or every component of one expenditure.
func (uc *ReconcileCostSourceUseCase) Execute(ctx context.Context, req *costsourcecomponentpb.ReconcileCostSourceRequest) (*costsourcecomponentpb.ReconcileCostSourceResponse, error) {
	return uc.execute(ctx, req)
}

func (uc *ReconcileCostSourceUseCase) execute(ctx context.Context, req *costsourcecomponentpb.ReconcileCostSourceRequest) (*costsourcecomponentpb.ReconcileCostSourceResponse, error) {
	if err := uc.c.gate(ctx, entityid.ActionRead); err != nil {
		return nil, err
	}
	r := uc.c.repos
	if r.CostSourceComponent == nil || r.AllocationBatch == nil || r.AllocationShare == nil || r.BillableCharge == nil {
		return nil, fmt.Errorf("cost_source_component: reconciliation repositories not wired")
	}
	if req == nil || (blank(req.GetComponentId()) && blank(req.GetExpenditureId())) {
		return nil, uc.c.refuse(ctx, codeValidation)
	}
	var comps []*costsourcecomponentpb.CostSourceComponent
	if !blank(req.GetComponentId()) {
		resp, err := r.CostSourceComponent.ReadCostSourceComponent(ctx, &costsourcecomponentpb.ReadCostSourceComponentRequest{Data: &costsourcecomponentpb.CostSourceComponent{Id: req.GetComponentId()}})
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "not found") {
			return nil, usecaseerr.RepoErr("cost_source_component", "read cost_source_component", err, req.GetComponentId())
		}
		if err != nil || len(resp.GetData()) == 0 {
			return nil, uc.c.refuse(ctx, codeNotFound)
		}
		comps = resp.Data[:1]
	} else {
		var err error
		comps, err = listdata.ListAll(func(p *commonpb.PaginationRequest, sort *commonpb.SortRequest) ([]*costsourcecomponentpb.CostSourceComponent, error) {
			resp, err := r.CostSourceComponent.ListCostSourceComponents(ctx, &costsourcecomponentpb.ListCostSourceComponentsRequest{Filters: listdata.EqFilter("expenditure_id", req.GetExpenditureId()), Sort: sort, Pagination: p})
			return resp.GetData(), err
		})
		if err != nil {
			return nil, usecaseerr.RepoErr("cost_source_component", "list components", err, req.GetExpenditureId())
		}
	}
	sort.Slice(comps, func(i, j int) bool { return comps[i].GetId() < comps[j].GetId() })

	out := &costsourcecomponentpb.ReconcileCostSourceResponse{Success: true}
	for _, c := range comps {
		rec, err := uc.reconcileOne(ctx, c)
		if err != nil {
			return nil, err
		}
		out.Data = append(out.Data, rec)
	}
	return out, nil
}

func (uc *ReconcileCostSourceUseCase) reconcileOne(ctx context.Context, c *costsourcecomponentpb.CostSourceComponent) (*costsourcecomponentpb.CostSourceComponentReconciliation, error) {
	r := uc.c.repos
	rec := &costsourcecomponentpb.CostSourceComponentReconciliation{ComponentId: c.GetId(), Currency: c.GetCurrency(), Amount: c.GetAmount(), SharesByKind: map[string]int64{}}

	batches, err := listdata.ListAll(func(p *commonpb.PaginationRequest, sort *commonpb.SortRequest) ([]*allocationbatchpb.AllocationBatch, error) {
		resp, err := r.AllocationBatch.ListAllocationBatches(ctx, &allocationbatchpb.ListAllocationBatchesRequest{Filters: listdata.EqFilter("cost_source_component_id", c.GetId()), Sort: sort, Pagination: p})
		return resp.GetData(), err
	})
	if err != nil {
		return nil, usecaseerr.RepoErr("cost_source_component", "list batches", err, c.GetId())
	}
	var batch *allocationbatchpb.AllocationBatch
	for _, b := range batches {
		if b.GetStatus() == allocationbatchpb.AllocationBatchStatus_ALLOCATION_BATCH_STATUS_PUBLISHED {
			batch = b
			break
		}
	}
	if batch == nil {
		return rec, nil // nothing published: not reconciled, all zero
	}
	rec.HasPublishedBatch = true
	rec.BatchId = &batch.Id

	shares, err := listdata.ListAll(func(p *commonpb.PaginationRequest, sort *commonpb.SortRequest) ([]*allocationsharepb.AllocationShare, error) {
		resp, err := r.AllocationShare.ListAllocationShares(ctx, &allocationsharepb.ListAllocationSharesRequest{Filters: listdata.EqFilter("allocation_batch_id", batch.GetId()), Sort: sort, Pagination: p})
		return resp.GetData(), err
	})
	if err != nil {
		return nil, usecaseerr.RepoErr("cost_source_component", "list shares", err, batch.GetId())
	}
	sort.Slice(shares, func(i, j int) bool { return shares[i].GetId() < shares[j].GetId() })
	for _, s := range shares {
		rec.SharesByKind[s.GetShareKind().String()] += s.GetAmount()
		rec.SharesTotal += s.GetAmount()
		if s.GetShareKind() != allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_RECOVERABLE {
			continue
		}
		rec.RecoverableTotal += s.GetAmount()
		charges, err := listdata.ListAll(func(p *commonpb.PaginationRequest, sort *commonpb.SortRequest) ([]*billablechargepb.BillableCharge, error) {
			resp, err := r.BillableCharge.ListBillableCharges(ctx, &billablechargepb.ListBillableChargesRequest{Filters: listdata.EqFilter("allocation_share_id", s.GetId()), Sort: sort, Pagination: p})
			return resp.GetData(), err
		})
		if err != nil {
			return nil, usecaseerr.RepoErr("cost_source_component", "list charges", err, s.GetId())
		}
		for _, ch := range charges {
			if ch.GetChargeKind() == billablechargepb.BillableChargeKind_BILLABLE_CHARGE_KIND_CORRECTION {
				if ch.GetStatus() == billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_ISSUED {
					rec.Corrections += ch.GetAmount()
					rec.IssuedNet += ch.GetAmount()
				}
				continue
			}
			switch ch.GetStatus() {
			case billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_OPEN:
				rec.ChargesOpen += ch.GetAmount()
			case billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_ISSUED:
				rec.ChargesIssued += ch.GetAmount()
				rec.IssuedNet += ch.GetAmount()
			case billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_CANCELLED:
				rec.ChargesCancelled += ch.GetAmount()
			}
		}
	}
	rec.ShareVariance = rec.Amount - rec.SharesTotal
	rec.ChargeVariance = rec.RecoverableTotal - (rec.ChargesOpen + rec.ChargesIssued + rec.ChargesCancelled)
	rec.Reconciled = rec.ShareVariance == 0 && rec.ChargeVariance == 0
	return rec, nil
}
