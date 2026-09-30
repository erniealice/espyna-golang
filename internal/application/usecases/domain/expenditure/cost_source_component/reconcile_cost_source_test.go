package cost_source_component

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	allocationbatchpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_batch"
	allocationsharepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/allocation_share"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ReconcileCostSource (moved from ledger/recovery_reporting, build-spec §7c C27).

// rows is a tiny in-memory table with string-equality filters.
type rows map[string]proto.Message

func (r rows) list(f *commonpb.FilterRequest) []proto.Message {
	var ids []string
	for id := range r {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var out []proto.Message
	for _, id := range ids {
		m, ok := r[id], true
		for _, tf := range f.GetFilters() {
			fd := m.ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(tf.GetField()))
			ok = ok && fd != nil && m.ProtoReflect().Get(fd).String() == tf.GetStringFilter().GetValue()
		}
		if ok {
			out = append(out, m)
		}
	}
	return out
}

type reconComps struct {
	pb.UnimplementedCostSourceComponentDomainServiceServer
	r rows
}

func (f reconComps) ReadCostSourceComponent(_ context.Context, q *pb.ReadCostSourceComponentRequest) (*pb.ReadCostSourceComponentResponse, error) {
	m, ok := f.r[q.GetData().GetId()]
	if !ok {
		return nil, errors.New("not found")
	}
	return &pb.ReadCostSourceComponentResponse{Data: []*pb.CostSourceComponent{m.(*pb.CostSourceComponent)}}, nil
}
func (f reconComps) ListCostSourceComponents(_ context.Context, q *pb.ListCostSourceComponentsRequest) (*pb.ListCostSourceComponentsResponse, error) {
	out := &pb.ListCostSourceComponentsResponse{Success: true}
	for _, m := range f.r.list(q.GetFilters()) {
		out.Data = append(out.Data, m.(*pb.CostSourceComponent))
	}
	return out, nil
}

type reconBatches struct {
	allocationbatchpb.UnimplementedAllocationBatchDomainServiceServer
	r rows
}

func (f reconBatches) ListAllocationBatches(_ context.Context, q *allocationbatchpb.ListAllocationBatchesRequest) (*allocationbatchpb.ListAllocationBatchesResponse, error) {
	out := &allocationbatchpb.ListAllocationBatchesResponse{Success: true}
	for _, m := range f.r.list(q.GetFilters()) {
		out.Data = append(out.Data, m.(*allocationbatchpb.AllocationBatch))
	}
	return out, nil
}

type reconShares struct {
	allocationsharepb.UnimplementedAllocationShareDomainServiceServer
	r rows
}

func (f reconShares) ListAllocationShares(_ context.Context, q *allocationsharepb.ListAllocationSharesRequest) (*allocationsharepb.ListAllocationSharesResponse, error) {
	out := &allocationsharepb.ListAllocationSharesResponse{Success: true}
	for _, m := range f.r.list(q.GetFilters()) {
		out.Data = append(out.Data, m.(*allocationsharepb.AllocationShare))
	}
	return out, nil
}

type reconCharges struct {
	billablechargepb.UnimplementedBillableChargeDomainServiceServer
	r rows
}

func (f reconCharges) ListBillableCharges(_ context.Context, q *billablechargepb.ListBillableChargesRequest) (*billablechargepb.ListBillableChargesResponse, error) {
	out := &billablechargepb.ListBillableChargesResponse{Success: true}
	for _, m := range f.r.list(q.GetFilters()) {
		out.Data = append(out.Data, m.(*billablechargepb.BillableCharge))
	}
	return out, nil
}

func reconHarness(deny bool, comps, batches, shares, charges rows) *UseCases {
	return NewUseCases(Repositories{
		CostSourceComponent: reconComps{r: comps}, AllocationBatch: reconBatches{r: batches},
		AllocationShare: reconShares{r: shares}, BillableCharge: reconCharges{r: charges},
	}, Services{
		Authorizer: ports.NewNoOpAuthorizer(), Transactor: tx{}, Translator: ports.NewNoOpTranslator(), IDGenerator: &ids{},
		ActionGatekeeper: actiongate.NewActionGatekeeper(&authz{deny: deny}, ports.NewNoOpTranslator()),
	})
}

func TestReconcileCostSourceBalances(t *testing.T) {
	comps := rows{"k1": &pb.CostSourceComponent{Id: "k1", ExpenditureId: "e1", Amount: 1000, Currency: "PHP", Active: true}}
	batches := rows{
		"b0": &allocationbatchpb.AllocationBatch{Id: "b0", CostSourceComponentId: "k1", Status: allocationbatchpb.AllocationBatchStatus_ALLOCATION_BATCH_STATUS_SUPERSEDED},
		"b1": &allocationbatchpb.AllocationBatch{Id: "b1", CostSourceComponentId: "k1", Status: allocationbatchpb.AllocationBatchStatus_ALLOCATION_BATCH_STATUS_PUBLISHED},
	}
	shares := rows{
		"s1": &allocationsharepb.AllocationShare{Id: "s1", AllocationBatchId: "b1", ShareKind: allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_RECOVERABLE, Amount: 600},
		"s2": &allocationsharepb.AllocationShare{Id: "s2", AllocationBatchId: "b1", ShareKind: allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_OWN_USE, Amount: 300},
		"s3": &allocationsharepb.AllocationShare{Id: "s3", AllocationBatchId: "b1", ShareKind: allocationsharepb.AllocationShareKind_ALLOCATION_SHARE_KIND_VACANCY, Amount: 100},
	}
	charges := rows{"ch1": &billablechargepb.BillableCharge{Id: "ch1", AllocationShareId: proto.String("s1"), Amount: 600, ChargeKind: billablechargepb.BillableChargeKind_BILLABLE_CHARGE_KIND_ORIGINAL, Status: billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_ISSUED}}
	uc := reconHarness(false, comps, batches, shares, charges)

	resp, err := uc.ReconcileCostSource.Execute(ctx(), &pb.ReconcileCostSourceRequest{ExpenditureId: proto.String("e1")})
	if err != nil || len(resp.Data) != 1 {
		t.Fatalf("resp=%v err=%v", resp, err)
	}
	r := resp.Data[0]
	if !r.Reconciled || r.ShareVariance != 0 || r.ChargeVariance != 0 || r.RecoverableTotal != 600 || r.SharesTotal != 1000 ||
		r.SharesByKind["ALLOCATION_SHARE_KIND_OWN_USE"] != 300 || r.IssuedNet != 600 {
		t.Fatalf("reconciliation = %+v", r)
	}

	// a credit note (-20) lowers the net issued amount but is not a variance of the originals
	charges["ch2"] = &billablechargepb.BillableCharge{Id: "ch2", AllocationShareId: proto.String("s1"), Amount: -20, ChargeKind: billablechargepb.BillableChargeKind_BILLABLE_CHARGE_KIND_CORRECTION, Status: billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_ISSUED}
	resp, _ = uc.ReconcileCostSource.Execute(ctx(), &pb.ReconcileCostSourceRequest{ComponentId: proto.String("k1")})
	if r := resp.Data[0]; !r.Reconciled || r.Corrections != -20 || r.IssuedNet != 580 {
		t.Fatalf("after correction = %+v", r)
	}

	// a missing charge is a variance
	delete(charges, "ch1")
	delete(charges, "ch2")
	resp, _ = uc.ReconcileCostSource.Execute(ctx(), &pb.ReconcileCostSourceRequest{ComponentId: proto.String("k1")})
	if r := resp.Data[0]; r.Reconciled || r.ChargeVariance != 600 {
		t.Fatalf("missing charge = %+v", r)
	}
	if _, err := uc.ReconcileCostSource.Execute(ctx(), &pb.ReconcileCostSourceRequest{ComponentId: proto.String("nope")}); !usecaseerr.IsCode(err, "not_found") {
		t.Fatalf("unknown component err = %v", err)
	}
	if _, err := uc.ReconcileCostSource.Execute(ctx(), &pb.ReconcileCostSourceRequest{}); !usecaseerr.IsCode(err, "validation") {
		t.Fatalf("no component or expenditure: err = %v", err)
	}
}

func TestReconcileCostSourceFailsClosed(t *testing.T) {
	if _, err := reconHarness(true, rows{}, rows{}, rows{}, rows{}).ReconcileCostSource.Execute(ctx(), &pb.ReconcileCostSourceRequest{ComponentId: proto.String("x")}); err == nil {
		t.Fatal("reconcile must be denied")
	}
	partial, _ := harness(false) // no reconciliation collaborators wired
	if _, err := partial.ReconcileCostSource.Execute(ctx(), &pb.ReconcileCostSourceRequest{ComponentId: proto.String("x")}); err == nil {
		t.Fatal("reconcile without its repositories must fail closed")
	}
}
