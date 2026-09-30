package cost_source_component

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/cost_source_component"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
	expenditurelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
	taxtreatmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_treatment"
	"google.golang.org/protobuf/proto"
)

type fakeRepo struct {
	pb.UnimplementedCostSourceComponentDomainServiceServer
	rows map[string]*pb.CostSourceComponent
}

func (f *fakeRepo) CreateCostSourceComponent(_ context.Context, r *pb.CreateCostSourceComponentRequest) (*pb.CreateCostSourceComponentResponse, error) {
	f.rows[r.Data.Id] = proto.Clone(r.Data).(*pb.CostSourceComponent)
	return &pb.CreateCostSourceComponentResponse{Data: []*pb.CostSourceComponent{r.Data}, Success: true}, nil
}
func (f *fakeRepo) ReadCostSourceComponent(_ context.Context, r *pb.ReadCostSourceComponentRequest) (*pb.ReadCostSourceComponentResponse, error) {
	row, ok := f.rows[r.Data.Id]
	if !ok {
		return nil, errors.New("not found")
	}
	return &pb.ReadCostSourceComponentResponse{Data: []*pb.CostSourceComponent{proto.Clone(row).(*pb.CostSourceComponent)}}, nil
}
func (f *fakeRepo) UpdateCostSourceComponent(_ context.Context, r *pb.UpdateCostSourceComponentRequest) (*pb.UpdateCostSourceComponentResponse, error) {
	proto.Merge(f.rows[r.Data.Id], r.Data)
	return &pb.UpdateCostSourceComponentResponse{Data: []*pb.CostSourceComponent{f.rows[r.Data.Id]}, Success: true}, nil
}
func (f *fakeRepo) DeleteCostSourceComponent(_ context.Context, r *pb.DeleteCostSourceComponentRequest) (*pb.DeleteCostSourceComponentResponse, error) {
	delete(f.rows, r.Data.Id)
	return &pb.DeleteCostSourceComponentResponse{Success: true}, nil
}

// lockingRepo is the fake with the postgres locker capability (C4: fakes implement it).
type lockingRepo struct {
	*fakeRepo
	lockErr error // injected infrastructure error
}

func (l *lockingRepo) LockCostSourceComponentForUpdate(_ context.Context, id string) (*pb.CostSourceComponent, error) {
	if l.lockErr != nil {
		return nil, l.lockErr
	}
	row, ok := l.rows[id]
	if !ok {
		return nil, fmt.Errorf("cost_source_component lock: %w", domainports.ErrLockedRowNotFound)
	}
	return proto.Clone(row).(*pb.CostSourceComponent), nil
}
func (l *lockingRepo) LockCostSourceComponentsByExpenditure(context.Context, string) ([]*pb.CostSourceComponent, error) {
	return nil, nil
}

type fakeLineItems struct {
	expenditurelineitempb.UnimplementedExpenditureLineItemDomainServiceServer
	parent map[string]string // line item id -> expenditure id
}

func (f *fakeLineItems) ReadExpenditureLineItem(_ context.Context, r *expenditurelineitempb.ReadExpenditureLineItemRequest) (*expenditurelineitempb.ReadExpenditureLineItemResponse, error) {
	p, ok := f.parent[r.Data.Id]
	if !ok {
		return nil, errors.New("not found")
	}
	return &expenditurelineitempb.ReadExpenditureLineItemResponse{Data: []*expenditurelineitempb.ExpenditureLineItem{{Id: r.Data.Id, ExpenditureId: p}}}, nil
}

type fakeTax struct {
	taxtreatmentpb.UnimplementedTaxTreatmentDomainServiceServer
	active map[string]bool
}

func (f *fakeTax) ReadTaxTreatment(_ context.Context, r *taxtreatmentpb.ReadTaxTreatmentRequest) (*taxtreatmentpb.ReadTaxTreatmentResponse, error) {
	a, ok := f.active[r.Data.Id]
	if !ok {
		return nil, errors.New("not found")
	}
	return &taxtreatmentpb.ReadTaxTreatmentResponse{Data: []*taxtreatmentpb.TaxTreatment{{Id: r.Data.Id, Active: a}}}, nil
}

type fakeExp struct {
	expenditurepb.UnimplementedExpenditureDomainServiceServer
	known map[string]bool
}

func (f *fakeExp) ReadExpenditure(_ context.Context, r *expenditurepb.ReadExpenditureRequest) (*expenditurepb.ReadExpenditureResponse, error) {
	if !f.known[r.Data.Id] {
		return nil, errors.New("not found")
	}
	return &expenditurepb.ReadExpenditureResponse{Data: []*expenditurepb.Expenditure{{Id: r.Data.Id}}}, nil
}

type authz struct{ deny bool }

func (a *authz) IsEnabled() bool { return true }
func (a *authz) HasPermission(context.Context, string, string) (bool, error) {
	return !a.deny, nil
}

type tx struct{}

func (tx) SupportsTransactions() bool               { return true }
func (tx) IsTransactionActive(context.Context) bool { return false }
func (tx) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type ids struct{ n atomic.Int64 }

func (s *ids) GenerateID() string                        { return fmt.Sprintf("csc-%03d", s.n.Add(1)) }
func (s *ids) IsEnabled() bool                           { return true }
func (s *ids) GetProviderInfo() string                   { return "seq" }
func (s *ids) GenerateIDWithPrefix(prefix string) string { return prefix + s.GenerateID() }

func harness(deny bool) (*UseCases, *fakeRepo) {
	uc, repo, _ := harnessLock(deny)
	return uc, repo
}

func harnessLock(deny bool) (*UseCases, *fakeRepo, *lockingRepo) {
	repo := &fakeRepo{rows: map[string]*pb.CostSourceComponent{}}
	lr := &lockingRepo{fakeRepo: repo}
	uc := NewUseCases(Repositories{
		CostSourceComponent: lr, Expenditure: &fakeExp{known: map[string]bool{"exp-1": true}},
		ExpenditureLineItem: &fakeLineItems{parent: map[string]string{"li-1": "exp-1", "li-other": "exp-2"}},
		TaxTreatment:        &fakeTax{active: map[string]bool{"tax-1": true, "tax-off": false}},
	}, Services{
		Authorizer: ports.NewNoOpAuthorizer(), Transactor: tx{}, Translator: ports.NewNoOpTranslator(), IDGenerator: &ids{},
		ActionGatekeeper: actiongate.NewActionGatekeeper(&authz{deny: deny}, ports.NewNoOpTranslator()),
	})
	return uc, repo, lr
}

func ctx() context.Context { return contextutil.WithUserID(context.Background(), "u1") }

func valid() *pb.CostSourceComponent {
	return &pb.CostSourceComponent{ExpenditureId: "exp-1", ComponentKind: pb.CostSourceComponentKind_COST_SOURCE_COMPONENT_KIND_ENERGY, Amount: 1000, Currency: "PHP"}
}

func TestCreateValidationAndDefaults(t *testing.T) {
	uc, repo := harness(false)
	bad := map[string]func(*pb.CostSourceComponent){
		"zero amount":       func(c *pb.CostSourceComponent) { c.Amount = 0 },
		"no currency":       func(c *pb.CostSourceComponent) { c.Currency = "" },
		"no kind":           func(c *pb.CostSourceComponent) { c.ComponentKind = 0 },
		"unknown exp":       func(c *pb.CostSourceComponent) { c.ExpenditureId = "exp-x" },
		"no exp":            func(c *pb.CostSourceComponent) { c.ExpenditureId = "" },
		"inverted interval": func(c *pb.CostSourceComponent) { c.ServiceFrom, c.ServiceTo = strp("2026-02-01"), strp("2026-01-01") },
	}
	for name, mut := range bad {
		c := valid()
		mut(c)
		want := "validation"
		if name == "unknown exp" {
			want = "reference_invalid"
		}
		if _, err := uc.CreateCostSourceComponent.Execute(ctx(), &pb.CreateCostSourceComponentRequest{Data: c}); !usecaseerr.IsCode(err, want) {
			t.Errorf("%s: want %s, got %v", name, want, err)
		}
	}
	c := valid()
	c.ClaimKind = pb.SourceClaimKind_SOURCE_CLAIM_KIND_ALLOCATION.Enum() // caller-supplied claim must be dropped
	resp, err := uc.CreateCostSourceComponent.Execute(ctx(), &pb.CreateCostSourceComponentRequest{Data: c})
	if err != nil {
		t.Fatal(err)
	}
	row := repo.rows[resp.Data[0].Id]
	if row.ClaimKind != nil || row.SourceVersion != 1 || !row.Active {
		t.Fatalf("defaults wrong: %+v", row)
	}
}

func TestUpdateDeleteRefusedOnceClaimed(t *testing.T) {
	uc, repo := harness(false)
	resp, _ := uc.CreateCostSourceComponent.Execute(ctx(), &pb.CreateCostSourceComponentRequest{Data: valid()})
	id := resp.Data[0].Id
	// unclaimed update works, bumps source_version, keeps the parent
	if _, err := uc.UpdateCostSourceComponent.Execute(ctx(), &pb.UpdateCostSourceComponentRequest{Data: &pb.CostSourceComponent{Id: id, Amount: 2000}}); err != nil {
		t.Fatal(err)
	}
	if r := repo.rows[id]; r.Amount != 2000 || r.SourceVersion != 2 || r.ExpenditureId != "exp-1" {
		t.Fatalf("update wrong: %+v", r)
	}
	if _, err := uc.UpdateCostSourceComponent.Execute(ctx(), &pb.UpdateCostSourceComponentRequest{Data: &pb.CostSourceComponent{Id: id, ServiceFrom: strp("2026-03-01"), ServiceTo: strp("2026-02-01")}}); !usecaseerr.IsCode(err, "validation") {
		t.Fatalf("merged invalid interval must be refused, got %v", err)
	}
	repo.rows[id].ClaimKind = pb.SourceClaimKind_SOURCE_CLAIM_KIND_RECOGNITION.Enum()
	if _, err := uc.UpdateCostSourceComponent.Execute(ctx(), &pb.UpdateCostSourceComponentRequest{Data: &pb.CostSourceComponent{Id: id, Amount: 5}}); !usecaseerr.IsCode(err, "claimed") {
		t.Fatalf("update of claimed component: want claimed, got %v", err)
	}
	if _, err := uc.DeleteCostSourceComponent.Execute(ctx(), &pb.DeleteCostSourceComponentRequest{Data: &pb.CostSourceComponent{Id: id}}); !usecaseerr.IsCode(err, "claimed") {
		t.Fatalf("delete of claimed component: want claimed, got %v", err)
	}
	repo.rows[id].ClaimKind = nil
	if _, err := uc.DeleteCostSourceComponent.Execute(ctx(), &pb.DeleteCostSourceComponentRequest{Data: &pb.CostSourceComponent{Id: id}}); err != nil {
		t.Fatal(err)
	}
	if _, ok := repo.rows[id]; ok {
		t.Fatal("row must be deleted")
	}
}

func TestPermissionDeniedFailsClosed(t *testing.T) {
	uc, _ := harness(true)
	if _, err := uc.CreateCostSourceComponent.Execute(ctx(), &pb.CreateCostSourceComponentRequest{Data: valid()}); err == nil {
		t.Error("create must be denied")
	}
	if _, err := uc.ListCostSourceComponents.Execute(ctx(), &pb.ListCostSourceComponentsRequest{}); err == nil {
		t.Error("list must be denied")
	}
	if _, err := uc.ReadCostSourceComponent.Execute(ctx(), &pb.ReadCostSourceComponentRequest{Data: &pb.CostSourceComponent{Id: "x"}}); err == nil {
		t.Error("read must be denied")
	}
}

func TestReferencesReadBeforeUse(t *testing.T) {
	uc, _ := harness(false)
	for name, mut := range map[string]func(*pb.CostSourceComponent){
		"line item of another expenditure": func(c *pb.CostSourceComponent) { c.ExpenditureLineItemId = strp("li-other") },
		"unknown line item":                func(c *pb.CostSourceComponent) { c.ExpenditureLineItemId = strp("li-x") },
		"unknown tax treatment":            func(c *pb.CostSourceComponent) { c.TaxTreatmentId = strp("tax-x") },
		"inactive tax treatment":           func(c *pb.CostSourceComponent) { c.TaxTreatmentId = strp("tax-off") },
	} {
		c := valid()
		mut(c)
		if _, err := uc.CreateCostSourceComponent.Execute(ctx(), &pb.CreateCostSourceComponentRequest{Data: c}); !usecaseerr.IsCode(err, "reference_invalid") {
			t.Errorf("%s: want reference_invalid, got %v", name, err)
		}
	}
	ok := valid()
	ok.ExpenditureLineItemId, ok.TaxTreatmentId = strp("li-1"), strp("tax-1")
	resp, err := uc.CreateCostSourceComponent.Execute(ctx(), &pb.CreateCostSourceComponentRequest{Data: ok})
	if err != nil {
		t.Fatalf("valid references: %v", err)
	}
	if _, err := uc.UpdateCostSourceComponent.Execute(ctx(), &pb.UpdateCostSourceComponentRequest{Data: &pb.CostSourceComponent{Id: resp.Data[0].Id, ExpenditureLineItemId: strp("li-other")}}); !usecaseerr.IsCode(err, "reference_invalid") {
		t.Errorf("update with a line item of another expenditure: want reference_invalid, got %v", err)
	}
}

func TestLockFailsClosedAndNeverMasksRepoErrors(t *testing.T) {
	// C4: a repository without the locker capability cannot update/delete.
	repo := &fakeRepo{rows: map[string]*pb.CostSourceComponent{"x": {Id: "x", ExpenditureId: "exp-1"}}}
	uc := NewUseCases(Repositories{CostSourceComponent: repo, Expenditure: &fakeExp{known: map[string]bool{"exp-1": true}}}, Services{
		Transactor: tx{}, Translator: ports.NewNoOpTranslator(), IDGenerator: &ids{},
		ActionGatekeeper: actiongate.NewActionGatekeeper(&authz{}, ports.NewNoOpTranslator()),
	})
	if _, err := uc.UpdateCostSourceComponent.Execute(ctx(), &pb.UpdateCostSourceComponentRequest{Data: &pb.CostSourceComponent{Id: "x", Amount: 5}}); !usecaseerr.IsCode(err, "lock_unavailable") {
		t.Errorf("update without locker: want lock_unavailable, got %v", err)
	}
	if _, err := uc.DeleteCostSourceComponent.Execute(ctx(), &pb.DeleteCostSourceComponentRequest{Data: &pb.CostSourceComponent{Id: "x"}}); !usecaseerr.IsCode(err, "lock_unavailable") {
		t.Errorf("delete without locker: want lock_unavailable, got %v", err)
	}
	// C9: a missing row is not_found; an infrastructure error is NOT.
	uc2, _, lr := harnessLock(false)
	if _, err := uc2.UpdateCostSourceComponent.Execute(ctx(), &pb.UpdateCostSourceComponentRequest{Data: &pb.CostSourceComponent{Id: "nope", Amount: 5}}); !usecaseerr.IsCode(err, "not_found") {
		t.Errorf("missing row: want not_found, got %v", err)
	}
	lr.lockErr = errors.New("connection reset by peer")
	_, err := uc2.UpdateCostSourceComponent.Execute(ctx(), &pb.UpdateCostSourceComponentRequest{Data: &pb.CostSourceComponent{Id: "nope", Amount: 5}})
	if err == nil || usecaseerr.IsCode(err, "not_found") {
		t.Errorf("infrastructure error must not be reported as not_found, got %v", err)
	}
}
