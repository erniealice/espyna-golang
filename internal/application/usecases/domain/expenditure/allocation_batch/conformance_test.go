package allocation_batch

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	allocationbatchpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_batch"
	allocationsharepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_share"
	costsourcecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
)

// shadowAuthz emulates the rbac authorizer in SHADOW mode: the regular check allows a would-be deny
// (log only) while the strict verdict is the real one.
type shadowAuthz struct{ strictAllow bool }

func (shadowAuthz) IsEnabled() bool { return true }
func (shadowAuthz) HasPermission(context.Context, string, string) (bool, error) {
	return true, nil
}
func (a shadowAuthz) HasPermissionStrict(context.Context, string, string) (bool, error) {
	return a.strictAllow, nil
}

// C3: create / update / publish use CheckStrict, so a shadow-mode allow-on-deny cannot publish.
func TestWritesUseStrictGateInShadowMode(t *testing.T) {
	w := newWorld()
	d := w.draft(t, standardShares())
	w.uc = NewUseCases(Repositories{
		AllocationBatch: w.batches, AllocationShare: w.shares, CostSourceComponent: w.comps, AgreementLineTerm: w.terms,
		ChargePolicyVersion: nil, BillableCharge: w.charges, ChargeComponent: w.cComps,
	}, Services{
		Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(), Transactor: fakeTransactor{}, IDGenerator: w.ids,
		ActionGatekeeper: actiongate.NewActionGatekeeper(shadowAuthz{strictAllow: false}, ports.NewNoOpTranslator()),
	})
	if _, err := w.uc.CreateAllocationBatch.Execute(uctx(), createReq(standardShares())); err == nil {
		t.Error("create must be denied by the strict gate")
	}
	if _, err := w.uc.UpdateAllocationBatchShares.Execute(uctx(), updateReq(d.Batch.Id, standardShares())); err == nil {
		t.Error("update must be denied by the strict gate")
	}
	if _, err := w.uc.PublishAllocationBatch.Execute(uctx(), publishReq(d.Batch.Id)); err == nil {
		t.Error("publish must be denied by the strict gate")
	}
	if len(w.charges.rows) != 0 || w.comps.get("csc-1").ClaimKind != nil {
		t.Error("a denied publish must write nothing")
	}
}

// C4: a repository without the locker capability cannot create/update/publish (no unlocked fallback).
type plainBatches struct {
	allocationbatchpb.AllocationBatchDomainServiceServer // hides the locker methods of *fakeBatches
}

type plainComps struct {
	costsourcecomponentpb.CostSourceComponentDomainServiceServer // hides the locker methods of *fakeComps
}

func TestWritesFailClosedWithoutLocker(t *testing.T) {
	w := newWorld()
	d := w.draft(t, standardShares())
	w.uc = NewUseCases(Repositories{
		AllocationBatch: plainBatches{w.batches}, AllocationShare: w.shares, CostSourceComponent: w.comps, AgreementLineTerm: w.terms,
		BillableCharge: w.charges, ChargeComponent: w.cComps, ChargePolicyVersion: &fakeVersions{}, ChargePolicyComponent: &fakePolicyComps{},
	}, Services{
		Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(), Transactor: fakeTransactor{}, IDGenerator: w.ids,
		ActionGatekeeper: actiongate.NewActionGatekeeper(&w.authz, ports.NewNoOpTranslator()),
	})
	if _, err := w.uc.PublishAllocationBatch.Execute(uctx(), publishReq(d.Batch.Id)); code(err) != "lock_unavailable" {
		t.Fatalf("publish without batch locker: want lock_unavailable, got %v", err)
	}
	if _, err := w.uc.CreateAllocationBatch.Execute(uctx(), createReq(standardShares())); code(err) != "lock_unavailable" {
		t.Fatalf("create without batch locker: want lock_unavailable, got %v", err)
	}
	// and the component locker: hide it
	w2 := newWorld()
	w2.uc = NewUseCases(Repositories{
		AllocationBatch: w2.batches, AllocationShare: w2.shares, CostSourceComponent: plainComps{w2.comps}, AgreementLineTerm: w2.terms,
	}, Services{
		Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(), Transactor: fakeTransactor{}, IDGenerator: w2.ids,
		ActionGatekeeper: actiongate.NewActionGatekeeper(&w2.authz, ports.NewNoOpTranslator()),
	})
	if _, err := w2.uc.CreateAllocationBatch.Execute(uctx(), createReq(standardShares())); code(err) != "lock_unavailable" {
		t.Fatalf("create without component locker: want lock_unavailable, got %v", err)
	}
}

// C9: an infrastructure failure is never reported as not_found; a missing row is.
func TestRepositoryErrorsAreNotReportedAsNotFound(t *testing.T) {
	w := newWorld()
	d := w.draft(t, standardShares())
	if _, err := w.uc.PublishAllocationBatch.Execute(uctx(), publishReq("no-such-batch")); code(err) != "not_found" {
		t.Fatalf("missing batch: want not_found, got %v", err)
	}
	if _, err := w.uc.CreateAllocationBatch.Execute(uctx(), &allocationbatchpb.CreateAllocationBatchRequest{Data: &allocationbatchpb.AllocationBatch{CostSourceComponentId: "no-such-csc"}, Shares: standardShares()}); code(err) != "not_found" {
		t.Fatalf("missing component: want not_found, got %v", err)
	}
	w.batches.listErr = errors.New("connection reset")
	_, err := w.uc.PublishAllocationBatch.Execute(uctx(), publishReq(d.Batch.Id))
	if err == nil || code(err) == "not_found" {
		t.Fatalf("a failing batch read must not be not_found, got %v", err)
	}
	w.batches.listErr = nil
	w.batches.lockFail = errors.New("deadlock detected")
	_, err = w.uc.PublishAllocationBatch.Execute(uctx(), publishReq(d.Batch.Id))
	if err == nil || code(err) == "not_found" {
		t.Fatalf("a failing lock must not be not_found, got %v", err)
	}
}

// C7: a legal zero-weight share keeps a zero amount and produces no charge.
func TestZeroWeightShareStoresZeroAndCreatesNoCharge(t *testing.T) {
	w := newWorld()
	shares := []*allocationsharepb.AllocationShare{
		{ShareKind: allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_OWN_USE, BasisNumerator: 0, BasisDenominator: 4},
		{ShareKind: allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_RECOVERABLE, SubscriptionId: strp("sub-1"), ClientId: strp("cli-1"), BasisNumerator: 4, BasisDenominator: 4},
		{ShareKind: allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_RECOVERABLE, SubscriptionId: strp("sub-2"), ClientId: strp("cli-2"), BasisNumerator: 0, BasisDenominator: 4},
	}
	d := w.draft(t, shares)
	if d.Shares[0].Amount != 0 || d.Shares[1].Amount != 10000 || d.Shares[2].Amount != 0 {
		t.Fatalf("draft amounts: %d %d %d", d.Shares[0].Amount, d.Shares[1].Amount, d.Shares[2].Amount)
	}
	res, err := w.uc.PublishAllocationBatch.Execute(uctx(), publishReq(d.Batch.Id))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Charges) != 1 || res.Charges[0].Amount != 10000 || len(w.charges.rows) != 1 {
		t.Fatalf("only the positive recoverable share may produce a charge: %d", len(res.Charges))
	}
}
