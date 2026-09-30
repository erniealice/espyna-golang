//go:build postgresql

package expenditure

import (
	"context"
	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core/scopetest"
	"testing"

	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	allocationbatchpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_batch"
	allocationsharepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_share"
	costsourcecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
)

// CostSourceComponent: every by-id and list operation carries a trusted, non-empty workspace predicate. A
// foreign workspace id, or a missing workspace, sees "not found"; a caller-supplied workspace_id
// never wins over the trusted one. Needs migration 20260930110000 on the leasing_usage1 clone.
func TestCostSourceComponentWorkspaceScope(t *testing.T) {
	h := scopetest.New(t, "cost_source_component")
	repo := NewPostgresCostSourceComponentRepository(h.Ops, entityid.CostSourceComponent).(*PostgresCostSourceComponentRepository)
	const id = "s1scope-cost_source_component"
	h.Run(t, func(base, ctxA, ctxB, ctxNone context.Context) {
		if _, err := repo.CreateCostSourceComponent(ctxA, &costsourcecomponentpb.CreateCostSourceComponentRequest{Data: &costsourcecomponentpb.CostSourceComponent{
			Id: id, WorkspaceId: scopetest.WsB, // must be ignored in favour of the trusted workspace
			Active: true, ExpenditureId: "e1", ComponentKind: costsourcecomponentpb.CostSourceComponentKind_COST_SOURCE_COMPONENT_KIND_ENERGY, Amount: 100, Currency: "PHP", SourceVersion: 1, Description: scopetest.Str("orig")}}); err != nil {
			t.Fatalf("create in own workspace: %v", err)
		}
		got, err := repo.ReadCostSourceComponent(ctxA, &costsourcecomponentpb.ReadCostSourceComponentRequest{Data: &costsourcecomponentpb.CostSourceComponent{Id: id}})
		if err != nil || len(got.Data) != 1 || got.Data[0].GetWorkspaceId() != scopetest.WsA {
			t.Fatalf("own read must succeed with the trusted workspace stamped: %v %+v", err, got)
		}
		for name, ctx := range map[string]context.Context{"foreign workspace": ctxB, "no workspace": ctxNone} {
			if _, err := repo.ReadCostSourceComponent(ctx, &costsourcecomponentpb.ReadCostSourceComponentRequest{Data: &costsourcecomponentpb.CostSourceComponent{Id: id}}); err == nil {
				t.Errorf("%s: read must fail (not found / fail closed)", name)
			}
			if _, err := repo.GetCostSourceComponentItemPageData(ctx, &costsourcecomponentpb.GetCostSourceComponentItemPageDataRequest{CostSourceComponentId: id}); err == nil {
				t.Errorf("%s: item page data must fail", name)
			}
			if _, err := repo.UpdateCostSourceComponent(ctx, &costsourcecomponentpb.UpdateCostSourceComponentRequest{Data: &costsourcecomponentpb.CostSourceComponent{Id: id, Description: scopetest.Str("hijacked")}}); err == nil {
				t.Errorf("%s: update must fail", name)
			}
			if _, err := repo.DeleteCostSourceComponent(ctx, &costsourcecomponentpb.DeleteCostSourceComponentRequest{Data: &costsourcecomponentpb.CostSourceComponent{Id: id}}); err == nil {
				t.Errorf("%s: delete must fail", name)
			}
			if l, err := repo.ListCostSourceComponents(ctx, &costsourcecomponentpb.ListCostSourceComponentsRequest{}); err == nil {
				for _, r := range l.Data {
					if r.GetId() == id {
						t.Errorf("%s: list leaked a foreign row", name)
					}
				}
			}
			if l, err := repo.GetCostSourceComponentListPageData(ctx, &costsourcecomponentpb.GetCostSourceComponentListPageDataRequest{}); err == nil {
				for _, r := range l.CostSourceComponentList {
					if r.GetId() == id {
						t.Errorf("%s: list page data leaked a foreign row", name)
					}
				}
			}
			if _, err := repo.LockCostSourceComponentForUpdate(ctx, id); err == nil {
				t.Errorf("%s: LockCostSourceComponentForUpdate must report not found", name)
			}
			if rows, err := repo.LockCostSourceComponentsByExpenditure(ctx, "e1"); err == nil && len(rows) != 0 {
				t.Errorf("%s: LockCostSourceComponentsByExpenditure leaked a foreign row", name)
			}
		}
		// The stored row is untouched by the refused writes.
		still, err := repo.ReadCostSourceComponent(ctxA, &costsourcecomponentpb.ReadCostSourceComponentRequest{Data: &costsourcecomponentpb.CostSourceComponent{Id: id}})
		if err != nil || len(still.Data) != 1 || still.Data[0].GetDescription() != "orig" {
			t.Errorf("foreign write changed the row: %v %+v", err, still)
		}
		// An OR filter would let the tenant predicate be OR-ed away: refused outright.
		if _, err := repo.ListCostSourceComponents(ctxA, &costsourcecomponentpb.ListCostSourceComponentsRequest{Filters: &commonpb.FilterRequest{Logic: commonpb.FilterLogic_OR, Filters: []*commonpb.TypedFilter{{Field: "id"}}}}); err == nil {
			t.Error("OR filter logic must be refused")
		}
		if v, err := repo.LockCostSourceComponentForUpdate(ctxA, id); err != nil || v.GetId() != id {
			t.Errorf("own LockCostSourceComponentForUpdate: %v", err)
		}
		if rows, err := repo.LockCostSourceComponentsByExpenditure(ctxA, "e1"); err != nil || len(rows) != 1 || rows[0].GetId() != id {
			t.Errorf("own LockCostSourceComponentsByExpenditure: %v %v", err, rows)
		}
	})
	// Outside a transaction the lock fails closed (SELECT ... FOR UPDATE needs a tx).
	if _, err := repo.LockCostSourceComponentForUpdate(identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: scopetest.WsA}), id); err == nil {
		t.Error("LockCostSourceComponentForUpdate outside a transaction must fail closed")
	}
}

// AllocationBatch: every by-id and list operation carries a trusted, non-empty workspace predicate. A
// foreign workspace id, or a missing workspace, sees "not found"; a caller-supplied workspace_id
// never wins over the trusted one. Needs migration 20260930110000 on the leasing_usage1 clone.
func TestAllocationBatchWorkspaceScope(t *testing.T) {
	h := scopetest.New(t, "allocation_batch")
	repo := NewPostgresAllocationBatchRepository(h.Ops, entityid.AllocationBatch).(*PostgresAllocationBatchRepository)
	const id = "s1scope-allocation_batch"
	h.Run(t, func(base, ctxA, ctxB, ctxNone context.Context) {
		if _, err := repo.CreateAllocationBatch(ctxA, &allocationbatchpb.CreateAllocationBatchRequest{Data: &allocationbatchpb.AllocationBatch{
			Id: id, WorkspaceId: scopetest.WsB, // must be ignored in favour of the trusted workspace
			Active: true, CostSourceComponentId: "c1", Revision: 1, Status: allocationbatchpb.AllocationBatchStatus_ALLOCATION_BATCH_STATUS_DRAFT, RuleCode: "LARGEST_REMAINDER", RuleVersion: 1, PreparedBy: scopetest.Str("orig")}}); err != nil {
			t.Fatalf("create in own workspace: %v", err)
		}
		got, err := repo.ReadAllocationBatch(ctxA, &allocationbatchpb.ReadAllocationBatchRequest{Data: &allocationbatchpb.AllocationBatch{Id: id}})
		if err != nil || len(got.Data) != 1 || got.Data[0].GetWorkspaceId() != scopetest.WsA {
			t.Fatalf("own read must succeed with the trusted workspace stamped: %v %+v", err, got)
		}
		for name, ctx := range map[string]context.Context{"foreign workspace": ctxB, "no workspace": ctxNone} {
			if _, err := repo.ReadAllocationBatch(ctx, &allocationbatchpb.ReadAllocationBatchRequest{Data: &allocationbatchpb.AllocationBatch{Id: id}}); err == nil {
				t.Errorf("%s: read must fail (not found / fail closed)", name)
			}
			if _, err := repo.GetAllocationBatchItemPageData(ctx, &allocationbatchpb.GetAllocationBatchItemPageDataRequest{AllocationBatchId: id}); err == nil {
				t.Errorf("%s: item page data must fail", name)
			}
			if _, err := repo.UpdateAllocationBatch(ctx, &allocationbatchpb.UpdateAllocationBatchRequest{Data: &allocationbatchpb.AllocationBatch{Id: id, PreparedBy: scopetest.Str("hijacked")}}); err == nil {
				t.Errorf("%s: update must fail", name)
			}
			if _, err := repo.DeleteAllocationBatch(ctx, &allocationbatchpb.DeleteAllocationBatchRequest{Data: &allocationbatchpb.AllocationBatch{Id: id}}); err == nil {
				t.Errorf("%s: delete must fail", name)
			}
			if l, err := repo.ListAllocationBatches(ctx, &allocationbatchpb.ListAllocationBatchesRequest{}); err == nil {
				for _, r := range l.Data {
					if r.GetId() == id {
						t.Errorf("%s: list leaked a foreign row", name)
					}
				}
			}
			if l, err := repo.GetAllocationBatchListPageData(ctx, &allocationbatchpb.GetAllocationBatchListPageDataRequest{}); err == nil {
				for _, r := range l.AllocationBatchList {
					if r.GetId() == id {
						t.Errorf("%s: list page data leaked a foreign row", name)
					}
				}
			}
			if _, err := repo.LockAllocationBatchForUpdate(ctx, id); err == nil {
				t.Errorf("%s: LockAllocationBatchForUpdate must report not found", name)
			}
			if rows, err := repo.LockAllocationBatchesByComponent(ctx, "c1"); err == nil && len(rows) != 0 {
				t.Errorf("%s: LockAllocationBatchesByComponent leaked a foreign row", name)
			}
		}
		// The stored row is untouched by the refused writes.
		still, err := repo.ReadAllocationBatch(ctxA, &allocationbatchpb.ReadAllocationBatchRequest{Data: &allocationbatchpb.AllocationBatch{Id: id}})
		if err != nil || len(still.Data) != 1 || still.Data[0].GetPreparedBy() != "orig" {
			t.Errorf("foreign write changed the row: %v %+v", err, still)
		}
		// An OR filter would let the tenant predicate be OR-ed away: refused outright.
		if _, err := repo.ListAllocationBatches(ctxA, &allocationbatchpb.ListAllocationBatchesRequest{Filters: &commonpb.FilterRequest{Logic: commonpb.FilterLogic_OR, Filters: []*commonpb.TypedFilter{{Field: "id"}}}}); err == nil {
			t.Error("OR filter logic must be refused")
		}
		if v, err := repo.LockAllocationBatchForUpdate(ctxA, id); err != nil || v.GetId() != id {
			t.Errorf("own LockAllocationBatchForUpdate: %v", err)
		}
		if rows, err := repo.LockAllocationBatchesByComponent(ctxA, "c1"); err != nil || len(rows) != 1 || rows[0].GetId() != id {
			t.Errorf("own LockAllocationBatchesByComponent: %v %v", err, rows)
		}
	})
	// Outside a transaction the lock fails closed (SELECT ... FOR UPDATE needs a tx).
	if _, err := repo.LockAllocationBatchForUpdate(identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: scopetest.WsA}), id); err == nil {
		t.Error("LockAllocationBatchForUpdate outside a transaction must fail closed")
	}
}

// AllocationShare: every by-id and list operation carries a trusted, non-empty workspace predicate. A
// foreign workspace id, or a missing workspace, sees "not found"; a caller-supplied workspace_id
// never wins over the trusted one. Needs migration 20260930110000 on the leasing_usage1 clone.
func TestAllocationShareWorkspaceScope(t *testing.T) {
	h := scopetest.New(t, "allocation_share")
	repo := NewPostgresAllocationShareRepository(h.Ops, entityid.AllocationShare).(*PostgresAllocationShareRepository)
	const id = "s1scope-allocation_share"
	h.Run(t, func(base, ctxA, ctxB, ctxNone context.Context) {
		if _, err := repo.CreateAllocationShare(ctxA, &allocationsharepb.CreateAllocationShareRequest{Data: &allocationsharepb.AllocationShare{
			Id: id, WorkspaceId: scopetest.WsB, // must be ignored in favour of the trusted workspace
			Active: true, AllocationBatchId: "b1", ShareKind: allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_OWN_USE, BasisNumerator: 1, BasisDenominator: 2, Amount: 10, SequenceOrder: 1}}); err != nil {
			t.Fatalf("create in own workspace: %v", err)
		}
		got, err := repo.ReadAllocationShare(ctxA, &allocationsharepb.ReadAllocationShareRequest{Data: &allocationsharepb.AllocationShare{Id: id}})
		if err != nil || len(got.Data) != 1 || got.Data[0].GetWorkspaceId() != scopetest.WsA {
			t.Fatalf("own read must succeed with the trusted workspace stamped: %v %+v", err, got)
		}
		for name, ctx := range map[string]context.Context{"foreign workspace": ctxB, "no workspace": ctxNone} {
			if _, err := repo.ReadAllocationShare(ctx, &allocationsharepb.ReadAllocationShareRequest{Data: &allocationsharepb.AllocationShare{Id: id}}); err == nil {
				t.Errorf("%s: read must fail (not found / fail closed)", name)
			}
			if _, err := repo.GetAllocationShareItemPageData(ctx, &allocationsharepb.GetAllocationShareItemPageDataRequest{AllocationShareId: id}); err == nil {
				t.Errorf("%s: item page data must fail", name)
			}
			if _, err := repo.UpdateAllocationShare(ctx, &allocationsharepb.UpdateAllocationShareRequest{Data: &allocationsharepb.AllocationShare{Id: id, Amount: 999}}); err == nil {
				t.Errorf("%s: update must fail", name)
			}
			if _, err := repo.DeleteAllocationShare(ctx, &allocationsharepb.DeleteAllocationShareRequest{Data: &allocationsharepb.AllocationShare{Id: id}}); err == nil {
				t.Errorf("%s: delete must fail", name)
			}
			if l, err := repo.ListAllocationShares(ctx, &allocationsharepb.ListAllocationSharesRequest{}); err == nil {
				for _, r := range l.Data {
					if r.GetId() == id {
						t.Errorf("%s: list leaked a foreign row", name)
					}
				}
			}
			if l, err := repo.GetAllocationShareListPageData(ctx, &allocationsharepb.GetAllocationShareListPageDataRequest{}); err == nil {
				for _, r := range l.AllocationShareList {
					if r.GetId() == id {
						t.Errorf("%s: list page data leaked a foreign row", name)
					}
				}
			}
		}
		// The stored row is untouched by the refused writes.
		still, err := repo.ReadAllocationShare(ctxA, &allocationsharepb.ReadAllocationShareRequest{Data: &allocationsharepb.AllocationShare{Id: id}})
		if err != nil || len(still.Data) != 1 || still.Data[0].GetAmount() != int64(10) {
			t.Errorf("foreign write changed the row: %v %+v", err, still)
		}
		// An OR filter would let the tenant predicate be OR-ed away: refused outright.
		if _, err := repo.ListAllocationShares(ctxA, &allocationsharepb.ListAllocationSharesRequest{Filters: &commonpb.FilterRequest{Logic: commonpb.FilterLogic_OR, Filters: []*commonpb.TypedFilter{{Field: "id"}}}}); err == nil {
			t.Error("OR filter logic must be refused")
		}
	})
}
