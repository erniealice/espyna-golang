package collection_application

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"google.golang.org/protobuf/proto"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	chargeeffectpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_effect"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenuepaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_payment"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
)

const (
	tRev = collectionapplicationpb.ApplicationTargetKind_APPLICATION_TARGET_KIND_REVENUE
	tDoc = collectionapplicationpb.ApplicationTargetKind_APPLICATION_TARGET_KIND_RECOVERY_DOCUMENT
)

func (h *harness) appsOf() (out []*collectionapplicationpb.CollectionApplication) {
	for _, m := range h.apps.list(nil) {
		out = append(out, m.(*collectionapplicationpb.CollectionApplication))
	}
	return
}

func (h *harness) effectSums() (debit, credit map[enumspb.ChargePostingRole]int64) {
	debit, credit = map[enumspb.ChargePostingRole]int64{}, map[enumspb.ChargePostingRole]int64{}
	for _, m := range h.effects.list(nil) {
		e := m.(*chargeeffectpb.ChargeEffect)
		if e.GetDirection() == chargeeffectpb.EffectDirection_EFFECT_DIRECTION_DEBIT {
			debit[e.GetPostingRole()] += e.GetAmount()
		} else {
			credit[e.GetPostingRole()] += e.GetAmount()
		}
	}
	return
}

func req(client string, amount int64, cur string) *collectionapplicationpb.ReceiveAndApplyCollectionRequest {
	return &collectionapplicationpb.ReceiveAndApplyCollectionRequest{ClientId: client, Amount: amount, Currency: cur,
		PaymentDate: proto.String("2026-09-10"), ReferenceNumber: proto.String("OR-1")}
}

func preview(client string, amount int64, cur string) *collectionapplicationpb.PreviewCollectionApplicationRequest {
	return &collectionapplicationpb.PreviewCollectionApplicationRequest{ClientId: client, Amount: amount, Currency: cur,
		PaymentDate: proto.String("2026-09-10"), ReferenceNumber: proto.String("OR-1")}
}

func reverseReq(id string) *collectionapplicationpb.ReverseCollectionApplicationRequest {
	return &collectionapplicationpb.ReverseCollectionApplicationRequest{CollectionApplicationId: id}
}

// planOf runs the preview and returns its plan.
func (h *harness) planOf(client string, amount int64, cur string) (*collectionapplicationpb.CollectionApplicationPlan, error) {
	r, err := h.uc.PreviewCollectionApplication.Execute(h.ctx(), preview(client, amount, cur))
	if err != nil {
		return nil, err
	}
	return r.Plan, nil
}

// AC-UC-10: 100 received; rent invoice 70 and recovery statement 60 share a due date; rent first (N9),
// the rest (30) goes to the recovery document, whose application writes DR CASH / CR RECEIVABLE.
func TestReceiveAndApplySplitsAcrossInvoiceAndRecoveryDocument(t *testing.T) {
	h := newHarness(false)
	h.revenue("rent1", "c1", 70, "2026-09-05", "PHP")
	h.statement("doc1", "c1", 60, "2026-09-05", "PHP")
	got, err := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 100, "PHP"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Applications) != 2 || got.Unapplied != 0 {
		t.Fatalf("applications=%d unapplied=%d", len(got.Applications), got.Unapplied)
	}
	a0, a1 := got.Applications[0], got.Applications[1]
	if a0.GetTargetKind() != tRev || a0.GetRevenueId() != "rent1" || a0.GetAmount() != 70 ||
		a1.GetTargetKind() != tDoc || a1.GetRecoveryDocumentId() != "doc1" || a1.GetAmount() != 30 {
		t.Fatalf("allocation = %v / %v", a0, a1)
	}
	c := got.Collection
	if c.GetRevenueId() != "" || c.GetClientId() != "c1" || c.GetWorkspaceId() != "ws1" || c.GetCurrency() != "PHP" || c.GetAmount() != 100 || c.GetCollectionType() != ReceiptCollectionType {
		t.Fatalf("receipt = %v", c)
	}
	// effects only for the recovery-document application: DR CASH 30 / CR RECEIVABLE 30
	d, cr := h.effectSums()
	if d[enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_CASH] != 30 || cr[enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_RECEIVABLE] != 30 || len(h.effects.rows) != 2 {
		t.Fatalf("effects debit=%v credit=%v rows=%d", d, cr, len(h.effects.rows))
	}
}

// AC-UC-10: every application targets exactly one document, matching target_kind.
func TestCollectionTypedTargetExactlyOne(t *testing.T) {
	h := newHarness(false)
	h.revenue("rent1", "c1", 50, "2026-09-01", "PHP")
	h.statement("doc1", "c1", 50, "2026-09-02", "PHP")
	if _, err := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 100, "PHP")); err != nil {
		t.Fatal(err)
	}
	for _, a := range h.appsOf() {
		hasRev, hasDoc := a.GetRevenueId() != "", a.GetRecoveryDocumentId() != ""
		if hasRev == hasDoc {
			t.Fatalf("application %s has revenue=%v doc=%v", a.GetId(), hasRev, hasDoc)
		}
		if (a.GetTargetKind() == tRev) != hasRev || a.GetApplicationKind() != collectionapplicationpb.ApplicationKind_APPLICATION_KIND_CASH || a.GetAmount() <= 0 {
			t.Fatalf("application %v violates target/kind/amount rules", a)
		}
	}
}

// N9: oldest due first; same due date -> rent (revenue) before recovery; then by number.
func TestReceiveAndApplyPaymentOrder(t *testing.T) {
	h := newHarness(false)
	h.statement("docA", "c1", 50, "2026-07-01", "PHP")
	h.revenue("rentLate", "c1", 50, "2026-08-01", "PHP")
	h.revenue("rentEarly", "c1", 50, "2026-07-01", "PHP")
	h.statement("docB", "c1", 50, "2026-07-01", "PHP")
	got, err := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 120, "PHP"))
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	var amounts []int64
	for _, a := range got.Applications {
		if a.GetTargetKind() == tRev {
			order = append(order, a.GetRevenueId())
		} else {
			order = append(order, a.GetRecoveryDocumentId())
		}
		amounts = append(amounts, a.GetAmount())
	}
	want := []string{"rentEarly", "docA", "docB"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
	if amounts[0] != 50 || amounts[1] != 50 || amounts[2] != 20 {
		t.Fatalf("amounts = %v", amounts)
	}
	// rank is the N9 position
	if got.Applications[2].GetOrderRank() != 3 {
		t.Fatalf("rank = %d", got.Applications[2].GetOrderRank())
	}
}

// N15: open items only in another currency -> refused, nothing written.
func TestReceiveAndApplyCurrencyMismatch(t *testing.T) {
	h := newHarness(false)
	h.revenue("rent1", "c1", 50, "2026-07-01", "PHP")
	_, err := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 100, "USD"))
	if !usecaseerr.IsCode(err, "currency_mismatch") {
		t.Fatalf("err = %v, want currency_mismatch", err)
	}
	if len(h.cols.rows) != 0 || len(h.apps.rows) != 0 {
		t.Fatal("a refused receipt must write nothing")
	}
	// mixed currencies: only the same-currency item is applied
	h.revenue("rentUSD", "c1", 40, "2026-07-01", "USD")
	got, err := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 100, "USD"))
	if err != nil || len(got.Applications) != 1 || got.Applications[0].GetRevenueId() != "rentUSD" || got.Unapplied != 60 {
		t.Fatalf("mixed = %v err=%v", got, err)
	}
}

func TestReceiveWithNothingOpenLeavesReceiptUnapplied(t *testing.T) {
	h := newHarness(false)
	got, err := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 500, "PHP"))
	if err != nil || len(got.Applications) != 0 || got.Unapplied != 500 || got.Collection.GetAmount() != 500 {
		t.Fatalf("got=%v err=%v", got, err)
	}
	for _, bad := range []*collectionapplicationpb.ReceiveAndApplyCollectionRequest{req("", 5, "PHP"), req("c1", 0, "PHP"), req("c1", -5, "PHP")} {
		if _, err := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), bad); err == nil {
			t.Fatalf("request %+v must be refused", bad)
		}
	}
}

func TestLegacyReceiptsAndPaymentsReduceOpenBalance(t *testing.T) {
	h := newHarness(false)
	h.revenue("rent1", "c1", 100, "2026-07-01", "PHP")
	st := "completed"
	h.cols.put(&collectionpb.Collection{Id: "old1", RevenueId: "rent1", Amount: 40, Status: st, Active: true})
	h.pays.put(&revenuepaymentpb.RevenuePayment{Id: "pay1", RevenueId: "rent1", Amount: 10, Active: true})
	plan, err := h.planOf("c1", 500, "PHP")
	if err != nil || len(plan.Allocations) != 1 || plan.Allocations[0].Balance != 50 || plan.Unapplied != 450 {
		t.Fatalf("plan = %+v err=%v", plan, err)
	}
}

func TestPreviewWritesNothingAndMatchesReceive(t *testing.T) {
	h := newHarness(false)
	h.revenue("rent1", "c1", 70, "2026-09-05", "PHP")
	h.statement("doc1", "c1", 60, "2026-09-05", "PHP")
	plan, err := h.planOf("c1", 100, "PHP")
	if err != nil {
		t.Fatal(err)
	}
	if len(h.cols.rows) != 0 || len(h.apps.rows) != 0 || len(h.effects.rows) != 0 {
		t.Fatal("preview must not write")
	}
	got, _ := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 100, "PHP"))
	if len(plan.Allocations) != len(got.Applications) || plan.Applied != 100 {
		t.Fatalf("preview %+v vs receive %d apps", plan, len(got.Applications))
	}
	for i, al := range plan.Allocations {
		if al.Apply != got.Applications[i].GetAmount() || al.TargetId != got.Plan.Allocations[i].TargetId {
			t.Fatalf("line %d differs", i)
		}
	}
}

// A second receipt sees the balances consumed by the first (no double application).
func TestSecondReceiptSeesReducedBalances(t *testing.T) {
	h := newHarness(false)
	h.statement("doc1", "c1", 100, "2026-09-05", "PHP")
	if _, err := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 60, "PHP")); err != nil {
		t.Fatal(err)
	}
	got, err := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 60, "PHP"))
	if err != nil || got.Applications[0].GetAmount() != 40 || got.Unapplied != 20 {
		t.Fatalf("second = %v err=%v", got, err)
	}
}

// D10: reversing restores the balance and mirrors the effects; a second reversal is refused.
func TestReverseApplicationRestoresBalance(t *testing.T) {
	h := newHarness(false)
	h.statement("doc1", "c1", 100, "2026-09-05", "PHP")
	got, err := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 100, "PHP"))
	if err != nil {
		t.Fatal(err)
	}
	if p, _ := h.planOf("c1", 100, "PHP"); len(p.Allocations) != 0 {
		t.Fatal("document must be fully applied")
	}
	revResp, err := h.uc.ReverseCollectionApplication.Execute(h.ctx(), reverseReq(got.Applications[0].GetId()))
	if err != nil {
		t.Fatal(err)
	}
	rev := revResp.Data
	if rev.GetReversesApplicationId() != got.Applications[0].GetId() || rev.GetStatus() != collectionapplicationpb.ApplicationStatus_APPLICATION_STATUS_REVERSED {
		t.Fatalf("reversal row = %v", rev)
	}
	p, _ := h.planOf("c1", 100, "PHP")
	if len(p.Allocations) != 1 || p.Allocations[0].Balance != 100 {
		t.Fatalf("balance not restored: %+v", p)
	}
	// effects net to zero per role
	d, c := h.effectSums()
	for _, role := range []enumspb.ChargePostingRole{enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_CASH, enumspb.ChargePostingRole_CHARGE_POSTING_ROLE_RECEIVABLE} {
		if d[role] != c[role] || d[role] != 100 {
			t.Fatalf("role %v debit=%d credit=%d", role, d[role], c[role])
		}
	}
	if len(h.effects.rows) != 4 {
		t.Fatalf("expected 4 effect rows (2 application + 2 reversal), got %d", len(h.effects.rows))
	}
	if _, err := h.uc.ReverseCollectionApplication.Execute(h.ctx(), reverseReq(got.Applications[0].GetId())); !usecaseerr.IsCode(err, "already_reversed") {
		t.Fatalf("second reversal err = %v", err)
	}
	if _, err := h.uc.ReverseCollectionApplication.Execute(h.ctx(), reverseReq(rev.GetId())); !usecaseerr.IsCode(err, "already_reversed") {
		t.Fatalf("reversing the reversal row err = %v", err)
	}
	if _, err := h.uc.ReverseCollectionApplication.Execute(h.ctx(), reverseReq("nope")); !usecaseerr.IsCode(err, "not_found") {
		t.Fatalf("unknown err = %v", err)
	}
}

func TestReverseRevenueApplicationRestoresRentBalance(t *testing.T) {
	h := newHarness(false)
	h.revenue("rent1", "c1", 100, "2026-09-05", "PHP")
	got, _ := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 100, "PHP"))
	if _, err := h.uc.ReverseCollectionApplication.Execute(h.ctx(), reverseReq(got.Applications[0].GetId())); err != nil {
		t.Fatal(err)
	}
	if len(h.effects.rows) != 0 {
		t.Fatal("revenue applications write no effects")
	}
	p, _ := h.planOf("c1", 100, "PHP")
	if len(p.Allocations) != 1 || p.Allocations[0].Balance != 100 {
		t.Fatalf("plan = %+v", p)
	}
}

func TestMissingApplicationPostingRefuses(t *testing.T) {
	h := newHarness(false)
	h.posts.rows = map[string]proto.Message{} // version has no mappings
	h.statement("doc1", "c1", 100, "2026-09-05", "PHP")
	if _, err := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 100, "PHP")); !usecaseerr.IsCode(err, "missing_posting") {
		t.Fatalf("err = %v, want missing_posting", err)
	}
}

func TestCashUseCasesFailClosedWhenDenied(t *testing.T) {
	h := newHarness(true)
	h.revenue("rent1", "c1", 100, "2026-09-05", "PHP")
	if _, err := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 100, "PHP")); err == nil {
		t.Fatal("receive must be denied")
	}
	if _, err := h.uc.PreviewCollectionApplication.Execute(h.ctx(), preview("c1", 100, "PHP")); err == nil {
		t.Fatal("preview must be denied")
	}
	if _, err := h.uc.ReverseCollectionApplication.Execute(h.ctx(), reverseReq("x")); err == nil {
		t.Fatal("reverse must be denied")
	}
	if len(h.cols.rows)+len(h.apps.rows) != 0 {
		t.Fatal("denied calls must write nothing")
	}
}

// Concurrent receipts for the same client serialise on the document row lock: the document is never
// over-applied.
func TestConcurrentReceiptsNeverOverApplyDocument(t *testing.T) {
	h := newHarness(false)
	h.statement("doc1", "c1", 100, "2026-09-05", "PHP")
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 40, "PHP"))
		}()
	}
	wg.Wait()
	var applied int64
	for _, a := range h.appsOf() {
		applied += a.GetAmount()
	}
	if applied != 100 {
		t.Fatalf("applied %d, want exactly 100 (never over-applied)", applied)
	}
}

// C5: the request's client and collection method are read in the actor's workspace first.
func TestReceiveVerifiesReferencesInWorkspace(t *testing.T) {
	h := newHarness(false)
	h.revenue("rent1", "ghost", 100, "2026-09-05", "PHP")
	if _, err := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("ghost", 100, "PHP")); !usecaseerr.IsCode(err, "client_required") {
		t.Fatalf("unknown/foreign client err = %v", err)
	}
	bad := req("c1", 100, "PHP")
	bad.CollectionMethodId = proto.String("nope")
	if _, err := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), bad); !usecaseerr.IsCode(err, "not_found") {
		t.Fatalf("unknown collection method err = %v", err)
	}
	good := req("c1", 100, "PHP")
	good.CollectionMethodId = proto.String("m1")
	if _, err := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), good); err != nil {
		t.Fatal(err)
	}
	if len(h.cols.rows) != 1 {
		t.Fatalf("only the valid receipt may be written, got %d", len(h.cols.rows))
	}
}

// C4: a repository that cannot lock the document row fails the receipt closed (no unlocked read).
func TestReceiveFailsClosedWithoutDocumentLocker(t *testing.T) {
	h := newHarness(false)
	h.statement("doc1", "c1", 100, "2026-09-05", "PHP")
	r := h.repos
	r.RecoveryDocument = &noLockDocs{RecoveryDocumentDomainServiceServer: &fakeDocs{s: h.docs}}
	uc := NewUseCases(r, h.svc)
	if _, err := uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 100, "PHP")); err == nil || usecaseerr.IsCode(err, "not_found") {
		t.Fatalf("err = %v, want a lock-capability failure", err)
	}
	if len(h.apps.rows) != 0 {
		t.Fatal("nothing may be applied without the lock")
	}
}

type failingApps struct {
	collectionapplicationpb.CollectionApplicationDomainServiceServer
}

// the lock succeeds so the read failure is the one under test
func (failingApps) LockCollectionApplicationForUpdate(context.Context, string) (*collectionapplicationpb.CollectionApplication, error) {
	return &collectionapplicationpb.CollectionApplication{}, nil
}

func (failingApps) ReadCollectionApplication(context.Context, *collectionapplicationpb.ReadCollectionApplicationRequest) (*collectionapplicationpb.ReadCollectionApplicationResponse, error) {
	return nil, errors.New("connection reset by peer")
}

// C9: a repository failure is never reported as not_found.
func TestRepositoryErrorIsNotMaskedAsNotFound(t *testing.T) {
	h := newHarness(false)
	r := h.repos
	r.CollectionApplication = failingApps{}
	uc := NewUseCases(r, h.svc)
	_, err := uc.ReverseCollectionApplication.Execute(h.ctx(), reverseReq("a1"))
	if err == nil || usecaseerr.IsCode(err, "not_found") {
		t.Fatalf("err = %v, want the repository error, not not_found", err)
	}
}

// C3: receive and reverse use the strict gate; a shadow-allow / strict-deny authorizer is refused.
func TestReceiveAndReverseUseStrictGate(t *testing.T) {
	h := newHarness(false)
	h.revenue("rent1", "c1", 100, "2026-09-05", "PHP")
	strict := NewUseCases(h.repos, Services{Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(strictAuthz{}, ports.NewNoOpTranslator()), Transactor: fakeTx{}, IDGenerator: &seqIDs{}})
	if _, err := strict.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 100, "PHP")); err == nil {
		t.Fatal("receive must use the strict gate")
	}
	if _, err := strict.ReverseCollectionApplication.Execute(h.ctx(), reverseReq("a")); err == nil {
		t.Fatal("reverse must use the strict gate")
	}
	if len(h.cols.rows)+len(h.apps.rows) != 0 {
		t.Fatal("denied calls must write nothing")
	}
}

// C16 (unit): concurrent receipts against one INVOICE (no recovery document) serialise on the invoice
// lock and never over-apply it.
func TestConcurrentReceiptsNeverOverApplyRevenue(t *testing.T) {
	h := newHarness(false)
	h.revenue("rent1", "c1", 100, "2026-09-05", "PHP")
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 40, "PHP"))
		}()
	}
	wg.Wait()
	var applied int64
	for _, a := range h.appsOf() {
		applied += a.GetAmount()
	}
	if applied != 100 {
		t.Fatalf("applied %d, want exactly 100 (never over-applied)", applied)
	}
}

// noLockRevenues hides the invoice locker (C4).
type noLockRevenues struct {
	revenuepb.RevenueDomainServiceServer
}

func TestReceiveFailsClosedWithoutRevenueLocker(t *testing.T) {
	h := newHarness(false)
	h.revenue("rent1", "c1", 100, "2026-09-05", "PHP")
	r := h.repos
	r.Revenue = noLockRevenues{h.repos.Revenue}
	uc := NewUseCases(r, h.svc)
	if _, err := uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 100, "PHP")); err == nil || usecaseerr.IsCode(err, "not_found") {
		t.Fatalf("err = %v, want a lock-capability failure", err)
	}
	if len(h.apps.rows) != 0 {
		t.Fatal("nothing may be applied without the lock")
	}
}

// Two concurrent reversals of the same application: exactly one wins, the other is already_reversed.
func TestConcurrentReversalsWriteOneReversingRow(t *testing.T) {
	h := newHarness(false)
	h.revenue("rent1", "c1", 100, "2026-09-05", "PHP")
	got, err := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 100, "PHP"))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := h.uc.ReverseCollectionApplication.Execute(h.ctx(), reverseReq(got.Applications[0].GetId()))
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	ok := 0
	for err := range results {
		if err == nil {
			ok++
		} else if !usecaseerr.IsCode(err, "already_reversed") {
			t.Fatalf("unexpected error %v", err)
		}
	}
	reversing := 0
	for _, a := range h.appsOf() {
		if a.GetReversesApplicationId() != "" {
			reversing++
		}
	}
	if ok != 1 || reversing != 1 {
		t.Fatalf("succeeded=%d reversing rows=%d, want exactly one", ok, reversing)
	}
}

// Void racing apply: an application of a document that a concurrent void already took (VOID) is not
// planned; a document with an APPLIED application cannot be voided (covered in recovery_document).
func TestReceiveSkipsVoidedDocument(t *testing.T) {
	h := newHarness(false)
	h.statement("doc1", "c1", 100, "2026-09-05", "PHP")
	h.docs.put(&recoverydocumentpb.RecoveryDocument{Id: "doc1", ClientId: "c1", Status: recoverydocumentpb.RecoveryDocumentStatus_RECOVERY_DOCUMENT_STATUS_VOID, Active: true})
	got, err := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 100, "PHP"))
	if err != nil || len(got.Applications) != 0 || got.Unapplied != 100 {
		t.Fatalf("voided document must not be applied: %+v %v", got, err)
	}
}

// listCounter counts every list read a preview makes against the workspace-scoped repositories.
type listCounter struct{ n int }

type countRevs struct {
	revenuepb.RevenueDomainServiceServer
	c *listCounter
}

func (f countRevs) ListRevenues(ctx context.Context, r *revenuepb.ListRevenuesRequest) (*revenuepb.ListRevenuesResponse, error) {
	f.c.n++
	return f.RevenueDomainServiceServer.ListRevenues(ctx, r)
}

type countDocs struct {
	recoverydocumentpb.RecoveryDocumentDomainServiceServer
	c *listCounter
}

func (f countDocs) ListRecoveryDocuments(ctx context.Context, r *recoverydocumentpb.ListRecoveryDocumentsRequest) (*recoverydocumentpb.ListRecoveryDocumentsResponse, error) {
	f.c.n++
	return f.RecoveryDocumentDomainServiceServer.ListRecoveryDocuments(ctx, r)
}

type countApps struct {
	collectionapplicationpb.CollectionApplicationDomainServiceServer
	c *listCounter
}

func (f countApps) ListCollectionApplications(ctx context.Context, r *collectionapplicationpb.ListCollectionApplicationsRequest) (*collectionapplicationpb.ListCollectionApplicationsResponse, error) {
	f.c.n++
	return f.CollectionApplicationDomainServiceServer.ListCollectionApplications(ctx, r)
}

// C17: a staff actor whose row scope excludes the client (the row-scoped ReadClient reports it as
// not found) gets a refusal and the preview performs zero list reads; an in-scope client still previews.
func TestPreviewRefusesClientOutsideRowScope(t *testing.T) {
	h := newHarness(false)
	h.revenue("rent1", "c3", 70, "2026-09-05", "PHP") // c3 exists in the workspace but not in the actor's scope
	h.statement("doc1", "c3", 60, "2026-09-05", "PHP")
	c := &listCounter{}
	r := PreviewCollectionApplicationRepositories{
		Collection: h.repos.Collection, CollectionApplication: countApps{h.repos.CollectionApplication, c},
		Revenue: countRevs{h.repos.Revenue, c}, RevenuePayment: h.repos.RevenuePayment,
		RecoveryDocument: countDocs{h.repos.RecoveryDocument, c}, Client: h.repos.Client, CollectionMethod: h.repos.CollectionMethod,
	}
	uc := NewPreviewCollectionApplicationUseCase(r, PreviewCollectionApplicationServices{
		Authorizer: h.svc.Authorizer, Translator: h.svc.Translator, ActionGatekeeper: h.svc.ActionGatekeeper})
	if _, err := uc.Execute(h.ctx(), preview("c3", 100, "PHP")); !usecaseerr.IsCode(err, "client_required") {
		t.Fatalf("out-of-scope client err = %v, want client_required", err)
	}
	if c.n != 0 {
		t.Fatalf("preview read %d lists before the client check, want 0", c.n)
	}
	if _, err := uc.Execute(h.ctx(), preview("c1", 100, "PHP")); err != nil {
		t.Fatalf("in-scope client must still preview: %v", err)
	}
	if c.n == 0 {
		t.Fatal("in-scope preview should have listed open items")
	}
}

// A1 m3: a receipt date that is not an ISO calendar date is refused before anything is written.
func TestReceiveRefusesAMalformedPaymentDate(t *testing.T) {
	h := newHarness(false)
	h.revenue("rent1", "c1", 100, "2026-09-05", "PHP")
	for _, d := range []string{"2026-13-01", "10/09/2026", "2026-02-30", "0001-01-01", "2026-9-1", "yesterday"} {
		r := req("c1", 10, "PHP")
		r.PaymentDate = proto.String(d)
		if _, err := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), r); !usecaseerr.IsCode(err, "date_invalid") {
			t.Errorf("%q: want date_invalid, got %v", d, err)
		}
		p := preview("c1", 10, "PHP")
		p.PaymentDate = proto.String(d)
		if _, err := h.uc.PreviewCollectionApplication.Execute(h.ctx(), p); !usecaseerr.IsCode(err, "date_invalid") {
			t.Errorf("preview %q: want date_invalid, got %v", d, err)
		}
	}
	if len(h.cols.rows) != 0 || len(h.apps.rows) != 0 {
		t.Fatal("a refused receipt wrote rows")
	}
	r := req("c1", 10, "PHP")
	r.PaymentDate = proto.String("")
	if _, err := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), r); err != nil {
		t.Fatalf("a blank date defaults to today: %v", err)
	}
}

// recordingRevenues records every invoice lock and runs onLock before the first one.
type recordingRevenues struct {
	*fakeRevenues
	mu     sync.Mutex
	locked []string
	onLock func()
}

func (f *recordingRevenues) LockRevenueForUpdate(ctx context.Context, id string) (*revenuepb.Revenue, error) {
	f.mu.Lock()
	f.locked = append(f.locked, id)
	hook := f.onLock
	f.onLock = nil
	f.mu.Unlock()
	if hook != nil {
		hook()
	}
	return f.fakeRevenues.LockRevenueForUpdate(ctx, id)
}

// A1 m2: reversing a REVENUE-target application locks the invoice (it reopens its balance), and a
// repository that cannot lock refuses the reversal.
func TestReverseLocksTheRevenueTarget(t *testing.T) {
	h := newHarness(false)
	h.revenue("rent1", "c1", 100, "2026-09-05", "PHP")
	got, err := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 100, "PHP"))
	if err != nil {
		t.Fatal(err)
	}
	rec := &recordingRevenues{fakeRevenues: h.repos.Revenue.(*fakeRevenues)}
	r := h.repos
	r.Revenue = rec
	if _, err := NewUseCases(r, h.svc).ReverseCollectionApplication.Execute(h.ctx(), reverseReq(got.Applications[0].GetId())); err != nil {
		t.Fatal(err)
	}
	if len(rec.locked) != 1 || rec.locked[0] != "rent1" {
		t.Fatalf("reverse locked %v, want [rent1]", rec.locked)
	}

	h2 := newHarness(false)
	h2.revenue("rent1", "c1", 100, "2026-09-05", "PHP")
	got, _ = h2.uc.ReceiveAndApplyCollection.Execute(h2.ctx(), req("c1", 100, "PHP"))
	r2 := h2.repos
	r2.Revenue = noLockRevenues{h2.repos.Revenue}
	if _, err := NewUseCases(r2, h2.svc).ReverseCollectionApplication.Execute(h2.ctx(), reverseReq(got.Applications[0].GetId())); err == nil {
		t.Fatal("a revenue repository that cannot lock must refuse the reversal")
	}
}

// A1 m2: an invoice that a concurrent reversal reopens after the receipt's first open-item snapshot
// is locked too (the open set is recomputed after locking) before the plan applies to it.
func TestReceiveLocksAnItemReopenedWhileLocking(t *testing.T) {
	h := newHarness(false)
	h.revenue("rent2", "c1", 50, "2026-09-01", "PHP")
	h.revenue("rent1", "c1", 100, "2026-09-05", "PHP")
	first, err := h.uc.ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 50, "PHP"))
	if err != nil || len(first.Applications) != 1 || first.Applications[0].GetRevenueId() != "rent2" {
		t.Fatalf("setup: %v %+v", err, first)
	}
	rec := &recordingRevenues{fakeRevenues: h.repos.Revenue.(*fakeRevenues)}
	rec.onLock = func() { // a concurrent reversal of rent2's application commits here
		h.apps.patch(&collectionapplicationpb.CollectionApplication{Id: first.Applications[0].GetId(), Status: collectionapplicationpb.ApplicationStatus_APPLICATION_STATUS_REVERSED})
	}
	r := h.repos
	r.Revenue = rec
	got, err := NewUseCases(r, h.svc).ReceiveAndApplyCollection.Execute(h.ctx(), req("c1", 150, "PHP"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.locked) != 2 || rec.locked[0] != "rent1" || rec.locked[1] != "rent2" {
		t.Fatalf("locks = %v, want rent1 then the reopened rent2", rec.locked)
	}
	var toRent2 int64
	for _, a := range got.Applications {
		if a.GetRevenueId() == "rent2" {
			toRent2 += a.GetAmount()
		}
	}
	if toRent2 != 50 {
		t.Fatalf("the reopened invoice must be applied under its lock, applied %d", toRent2)
	}
}
