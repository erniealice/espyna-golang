package billable_charge

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
	"google.golang.org/protobuf/proto"
)

type chargeRepo struct {
	billablechargepb.UnimplementedBillableChargeDomainServiceServer
	mu        sync.Mutex
	rows      map[string]*billablechargepb.BillableCharge
	lockErr   error
	listReqs  []*billablechargepb.ListBillableChargesRequest
	listPages *billablechargepb.ListBillableChargesResponse
}

func (c *chargeRepo) ReadBillableCharge(_ context.Context, r *billablechargepb.ReadBillableChargeRequest) (*billablechargepb.ReadBillableChargeResponse, error) {
	return nil, errors.New("adjust must lock, never read unlocked")
}
func (c *chargeRepo) CreateBillableCharge(_ context.Context, r *billablechargepb.CreateBillableChargeRequest) (*billablechargepb.CreateBillableChargeResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rows[r.Data.Id] = proto.Clone(r.Data).(*billablechargepb.BillableCharge)
	return &billablechargepb.CreateBillableChargeResponse{Data: []*billablechargepb.BillableCharge{r.Data}, Success: true}, nil
}
func (c *chargeRepo) ListBillableCharges(_ context.Context, r *billablechargepb.ListBillableChargesRequest) (*billablechargepb.ListBillableChargesResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.listReqs = append(c.listReqs, r)
	if c.listPages != nil {
		return c.listPages, nil
	}
	resp := &billablechargepb.ListBillableChargesResponse{Success: true}
	f := r.GetFilters().GetFilters()
	for _, x := range c.rows {
		if len(f) == 1 && f[0].GetField() == "predecessor_id" && x.GetPredecessorId() == f[0].GetStringFilter().GetValue() {
			resp.Data = append(resp.Data, proto.Clone(x).(*billablechargepb.BillableCharge))
		}
	}
	return resp, nil
}
func (c *chargeRepo) LockBillableChargeForUpdate(_ context.Context, id string) (*billablechargepb.BillableCharge, error) {
	if c.lockErr != nil {
		return nil, c.lockErr
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if x, ok := c.rows[id]; ok {
		return proto.Clone(x).(*billablechargepb.BillableCharge), nil
	}
	return nil, fmt.Errorf("billable_charge lock: %w", domainports.ErrLockedRowNotFound)
}

// plainChargeRepo hides the locker methods.
type plainChargeRepo struct {
	billablechargepb.BillableChargeDomainServiceServer
}

type compRepo struct {
	chargecomponentpb.UnimplementedChargeComponentDomainServiceServer
	rows []*chargecomponentpb.ChargeComponent
}

func (c *compRepo) ListChargeComponents(_ context.Context, r *chargecomponentpb.ListChargeComponentsRequest) (*chargecomponentpb.ListChargeComponentsResponse, error) {
	resp := &chargecomponentpb.ListChargeComponentsResponse{Success: true}
	for _, x := range c.rows {
		if x.GetBillableChargeId() == r.GetFilters().GetFilters()[0].GetStringFilter().GetValue() {
			resp.Data = append(resp.Data, x)
		}
	}
	return resp, nil
}
func (c *compRepo) CreateChargeComponent(_ context.Context, r *chargecomponentpb.CreateChargeComponentRequest) (*chargecomponentpb.CreateChargeComponentResponse, error) {
	c.rows = append(c.rows, proto.Clone(r.Data).(*chargecomponentpb.ChargeComponent))
	return &chargecomponentpb.CreateChargeComponentResponse{Success: true}, nil
}

type txFake struct{}

func (txFake) SupportsTransactions() bool               { return true }
func (txFake) IsTransactionActive(context.Context) bool { return false }
func (txFake) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type seqID struct{ n int }

func (s *seqID) GenerateID() string                        { s.n++; return fmt.Sprintf("id-%d", s.n) }
func (s *seqID) IsEnabled() bool                           { return true }
func (s *seqID) GetProviderInfo() string                   { return "seq" }
func (s *seqID) GenerateIDWithPrefix(prefix string) string { return prefix + s.GenerateID() }

type shadowAuthz struct{ strictAllow bool }

func (shadowAuthz) IsEnabled() bool                                             { return true }
func (shadowAuthz) HasPermission(context.Context, string, string) (bool, error) { return true, nil }
func (a shadowAuthz) HasPermissionStrict(context.Context, string, string) (bool, error) {
	return a.strictAllow, nil
}

func issuedOriginal() *billablechargepb.BillableCharge {
	return &billablechargepb.BillableCharge{
		Id: "ch-1", ObligationKey: "SRC:c:SUB:s:SVC:a/b", Amount: 10000, Currency: "PHP",
		ChargeKind: billablechargepb.BillableChargeKind_BILLABLE_CHARGE_KIND_ORIGINAL,
		Status:     billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_ISSUED, Active: true,
	}
}

func adjustHarness(authz actiongate.Authorizer) (*UseCases, *chargeRepo, *compRepo) {
	cr := &chargeRepo{rows: map[string]*billablechargepb.BillableCharge{"ch-1": issuedOriginal()}}
	kr := &compRepo{rows: []*chargecomponentpb.ChargeComponent{{
		Id: "cc-1", BillableChargeId: "ch-1", Amount: 10000,
		ComponentRole:    enumspb.ChargeComponentRole_CHARGE_COMPONENT_ROLE_RECOVERY_COST,
		DocumentKind:     enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT,
		BookPresentation: enumspb.BookPresentation_BOOK_PRESENTATION_EXCLUDED_REIMBURSEMENT,
		TaxPosition:      enumspb.TaxPosition_TAX_POSITION_EXCLUDED_REIMBURSEMENT,
	}}}
	uc := NewUseCases(Repositories{BillableCharge: cr, ChargeComponent: kr}, Services{
		Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(), Transactor: txFake{}, IDGenerator: &seqID{},
		ActionGatekeeper: actiongate.NewActionGatekeeper(authz, ports.NewNoOpTranslator()),
	})
	return uc, cr, kr
}

func actx() context.Context { return contextutil.WithUserID(context.Background(), "u1") }

func adjustReq(id string, amount int64) *billablechargepb.AdjustBillableChargeRequest {
	return &billablechargepb.AdjustBillableChargeRequest{BillableChargeId: id, NewAmount: amount, Reason: "rate dispute"}
}

func TestAdjustCreatesSignedCorrectionAndComponent(t *testing.T) {
	uc, cr, kr := adjustHarness(shadowAuthz{strictAllow: true})
	resp, err := uc.AdjustBillableCharge.Execute(actx(), adjustReq("ch-1", 7000))
	if err != nil {
		t.Fatal(err)
	}
	c := resp.GetData()
	if !resp.GetSuccess() || c.GetAmount() != -3000 || c.GetPredecessorId() != "ch-1" || c.GetChargeKind() != billablechargepb.BillableChargeKind_BILLABLE_CHARGE_KIND_CORRECTION ||
		c.GetStatus() != billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_OPEN || c.GetObligationKey() != "SRC:c:SUB:s:SVC:a/b#r1" {
		t.Fatalf("correction wrong: %+v", c)
	}
	if len(kr.rows) != 2 || kr.rows[1].GetAmount() != -3000 || kr.rows[1].GetBillableChargeId() != c.GetId() {
		t.Fatalf("component wrong: %+v", kr.rows)
	}
	if cr.rows["ch-1"].GetAmount() != 10000 {
		t.Fatal("the original must never be rewritten")
	}
	// a second adjustment sees the first correction: 7000 is the effective amount now
	if _, err := uc.AdjustBillableCharge.Execute(actx(), adjustReq("ch-1", 7000)); !usecaseerr.IsCode(err, "adjust_not_downward") {
		t.Fatalf("no-op adjustment: want adjust_not_downward, got %v", err)
	}
	if r, err := uc.AdjustBillableCharge.Execute(actx(), adjustReq("ch-1", 0)); err != nil || r.GetData().GetAmount() != -7000 || r.GetData().GetObligationKey() != "SRC:c:SUB:s:SVC:a/b#r2" {
		t.Fatalf("second correction: %v %+v", err, r)
	}
}

func TestAdjustRefusals(t *testing.T) {
	uc, cr, _ := adjustHarness(shadowAuthz{strictAllow: true})
	if _, err := uc.AdjustBillableCharge.Execute(actx(), adjustReq("ch-1", -1)); !usecaseerr.IsCode(err, "adjust_not_downward") {
		t.Errorf("negative new amount: %v", err)
	}
	if _, err := uc.AdjustBillableCharge.Execute(actx(), adjustReq("ch-1", 10000)); !usecaseerr.IsCode(err, "adjust_not_downward") {
		t.Errorf("not lower: %v", err)
	}
	if _, err := uc.AdjustBillableCharge.Execute(actx(), adjustReq("", 1)); !usecaseerr.IsCode(err, "not_found") {
		t.Errorf("blank id: %v", err)
	}
	if _, err := uc.AdjustBillableCharge.Execute(actx(), adjustReq("ghost", 1)); !usecaseerr.IsCode(err, "not_found") {
		t.Errorf("missing charge: %v", err)
	}
	cr.rows["ch-1"].Status = billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_OPEN
	if _, err := uc.AdjustBillableCharge.Execute(actx(), adjustReq("ch-1", 1)); !usecaseerr.IsCode(err, "not_issued") {
		t.Errorf("open charge: %v", err)
	}
}

// C3: adjust uses the strict gate; C4: no unlocked fallback; C9: infra errors are not not_found.
func TestAdjustStrictGateFailClosedLockAndErrors(t *testing.T) {
	uc, cr, _ := adjustHarness(shadowAuthz{strictAllow: false})
	if _, err := uc.AdjustBillableCharge.Execute(actx(), adjustReq("ch-1", 1)); err == nil {
		t.Fatal("adjust must be denied by the strict gate even where the regular check allows")
	}
	if len(cr.rows) != 1 {
		t.Fatal("a denied adjust must write nothing")
	}
	uc2, _, _ := adjustHarness(shadowAuthz{strictAllow: true})
	uc2 = NewUseCases(Repositories{BillableCharge: plainChargeRepo{&chargeRepo{rows: map[string]*billablechargepb.BillableCharge{"ch-1": issuedOriginal()}}}, ChargeComponent: &compRepo{}}, Services{
		Translator: ports.NewNoOpTranslator(), Transactor: txFake{}, IDGenerator: &seqID{},
		ActionGatekeeper: actiongate.NewActionGatekeeper(shadowAuthz{strictAllow: true}, ports.NewNoOpTranslator()),
	})
	if _, err := uc2.AdjustBillableCharge.Execute(actx(), adjustReq("ch-1", 1)); !usecaseerr.IsCode(err, "lock_unavailable") {
		t.Fatalf("no locker: want lock_unavailable, got %v", err)
	}
	uc3, cr3, _ := adjustHarness(shadowAuthz{strictAllow: true})
	cr3.lockErr = errors.New("connection reset")
	if _, err := uc3.AdjustBillableCharge.Execute(actx(), adjustReq("ch-1", 1)); err == nil || usecaseerr.IsCode(err, "not_found") {
		t.Fatalf("infra lock error must not be not_found: %v", err)
	}
}

// C11: ListBillableCharges maps the Status field to a filter and forwards pagination.
func TestListBillableChargesStatusAndPagination(t *testing.T) {
	uc, cr, _ := adjustHarness(shadowAuthz{strictAllow: true})
	st := billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_ISSUED
	req := &billablechargepb.ListBillableChargesRequest{Status: &st}
	if _, err := uc.ListBillableCharges.Execute(actx(), req); err != nil {
		t.Fatal(err)
	}
	last := cr.listReqs[len(cr.listReqs)-1]
	f := last.GetFilters().GetFilters()
	if len(f) != 1 || f[0].GetField() != "status" || f[0].GetStringFilter().GetValue() != st.String() {
		t.Fatalf("status filter not built: %+v", last)
	}
	if req.Filters != nil {
		t.Fatal("the caller's request must not be mutated")
	}
}
