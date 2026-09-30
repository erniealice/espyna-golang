package recovery_document

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"google.golang.org/protobuf/proto"

	chargeeffectpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_effect"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
	documentseriespb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/document_series"
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
	billablechargepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billable_charge"
	chargecomponentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/charge_component"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
)

const active = documentseriespb.DocumentSeriesStatus_DOCUMENT_SERIES_STATUS_ACTIVE

func code(t *testing.T, err error, want string) {
	t.Helper()
	var ec interface{ ErrorCode() string }
	if !errors.As(err, &ec) || ec.ErrorCode() != want {
		t.Fatalf("want error code %q, got %v", want, err)
	}
}

func issueReq(key string, ids ...string) *recoverydocumentpb.IssueRecoveryDocumentsRequest {
	return &recoverydocumentpb.IssueRecoveryDocumentsRequest{BillableChargeIds: ids, DocumentSeriesId: "soa",
		IssueDate: proto.String("2026-10-01"), DueDate: proto.String("2026-10-15"), IssuanceKey: key}
}

func voidReq(id, reason string) *recoverydocumentpb.VoidRecoveryDocumentRequest {
	return &recoverydocumentpb.VoidRecoveryDocumentRequest{RecoveryDocumentId: id, Reason: reason}
}

// AC-UC-07: numbers are unique and gapless per series even with concurrent issues; issued rows
// have no update path through the use cases (a second issue of the same charge is refused not_open).
func TestReimbursementBillingSeriesAndImmutability(t *testing.T) {
	h := newHarness(false)
	h.addSeries("soa", active)
	const n = 8
	for i := 0; i < n; i++ {
		h.addCharge("c"+string(rune('a'+i)), "client-"+string(rune('a'+i)), "sub-1", 1000+int64(i), "PHP")
	}
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("k"+string(rune('a'+i)), "c"+string(rune('a'+i))))
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	seen := map[int64]bool{}
	numbers := map[string]bool{}
	for _, m := range h.docs.s.list(nil) {
		d := m.(*recoverydocumentpb.RecoveryDocument)
		seen[d.GetSequenceNumber()] = true
		numbers[d.GetDocumentNumber()] = true
	}
	for i := int64(1); i <= n; i++ {
		if !seen[i] {
			t.Fatalf("sequence %d missing: gap (%v)", i, seen)
		}
	}
	if len(numbers) != n || !numbers["SOA-0001"] || !numbers["SOA-0008"] {
		t.Fatalf("document numbers wrong: %v", numbers)
	}
	s, _ := h.series.s.read("soa")
	if s.(*documentseriespb.DocumentSeries).GetNextNumber() != n+1 {
		t.Fatalf("next_number = %d", s.(*documentseriespb.DocumentSeries).GetNextNumber())
	}
	_, err := h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("again", "ca"))
	code(t, err, "not_open")
}

// D8: same issuance_key returns the original documents, writes nothing new.
func TestIssueRecoveryDocumentsRetryReturnsOriginal(t *testing.T) {
	h := newHarness(false)
	h.addSeries("soa", active)
	h.addCharge("c1", "cl1", "s1", 500, "PHP")
	h.addCharge("c2", "cl2", "s1", 700, "PHP")
	first, err := h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("K", "c1", "c2"))
	if err != nil || len(first.Data) != 2 || first.Replayed {
		t.Fatalf("first issue: %+v %v", first, err)
	}
	effects := len(h.effects.s.list(nil))
	again, err := h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("K", "c1", "c2"))
	if err != nil || !again.Replayed || len(again.Data) != 2 {
		t.Fatalf("retry: %+v %v", again, err)
	}
	for i := range again.Data {
		if again.Data[i].GetId() != first.Data[i].GetId() || again.Data[i].GetDocumentNumber() != first.Data[i].GetDocumentNumber() {
			t.Fatal("retry must return the original documents")
		}
	}
	if len(h.docs.s.list(nil)) != 2 || len(h.effects.s.list(nil)) != effects {
		t.Fatal("retry must not write new documents or effects")
	}
	s, _ := h.series.s.read("soa")
	if s.(*documentseriespb.DocumentSeries).GetNextNumber() != 3 {
		t.Fatal("retry must not consume numbers")
	}
	h.series.s.update(&documentseriespb.DocumentSeries{Id: "soa", Status: documentseriespb.DocumentSeriesStatus_DOCUMENT_SERIES_STATUS_RETIRED})
	if r, err := h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("K", "c1", "c2")); err != nil || !r.Replayed {
		t.Fatalf("retry after retire: %v", err)
	}
}

func TestIssueRefusals(t *testing.T) {
	h := newHarness(false)
	h.addSeries("soa", active)
	h.addSeries("old", documentseriespb.DocumentSeriesStatus_DOCUMENT_SERIES_STATUS_RETIRED)
	h.addCharge("c1", "cl1", "s1", 500, "PHP")
	h.addCharge("c2", "cl1", "s1", 500, "USD")

	_, err := h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("k0"))
	code(t, err, "nothing_selected")
	_, err = h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("k1", "c1", "c2"))
	code(t, err, "currency_mismatch")
	r := issueReq("k2", "c1")
	r.DocumentSeriesId = "old"
	_, err = h.uc.IssueRecoveryDocuments.Execute(h.ctx(), r)
	code(t, err, "series_retired")
	h.charges.s.update(&billablechargepb.BillableCharge{Id: "c1", Status: billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_CANCELLED})
	_, err = h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("k3", "c1"))
	code(t, err, "not_open")
	if len(h.docs.s.list(nil)) != 0 {
		t.Fatal("refusals must not create documents")
	}
	d := newHarness(true)
	if _, err := d.uc.IssueRecoveryDocuments.Execute(d.ctx(), issueReq("x", "c1")); err == nil {
		t.Fatal("issue must be denied")
	}
}

// D9: issuing writes balanced ISSUE effects (DR RECEIVABLE / CR CLEARING); a version without
// mapping refuses missing_posting.
func TestIssueWritesIssueEffects(t *testing.T) {
	h := newHarness(false)
	h.addSeries("soa", active)
	h.addCharge("c1", "cl1", "s1", 500, "PHP")
	h.addCharge("c2", "cl1", "s1", 250, "PHP")
	h.addCharge("c3", "cl2", "s1", 100, "PHP")
	resp, err := h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("K", "c1", "c2", "c3"))
	if err != nil || len(resp.Data) != 2 {
		t.Fatalf("want 2 documents (one per client+subscription), got %v %v", resp, err)
	}
	var dr, cr int64
	for _, m := range h.effects.s.list(nil) {
		e := m.(*chargeeffectpb.ChargeEffect)
		if e.GetDirection() == chargeeffectpb.EffectDirection_EFFECT_DIRECTION_DEBIT {
			dr += e.GetAmount()
		} else {
			cr += e.GetAmount()
		}
	}
	if dr != 850 || cr != 850 {
		t.Fatalf("effects unbalanced dr=%d cr=%d", dr, cr)
	}
	for _, id := range []string{"c1", "c2", "c3"} {
		if h.charge(id).GetStatus() != billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_ISSUED {
			t.Fatalf("%s not ISSUED", id)
		}
	}
	if n := len(h.lines.s.list(nil)); n != 3 {
		t.Fatalf("lines=%d want 3", n)
	}
	h.addCharge("c4", "cl3", "s1", 100, "PHP")
	h.charges.s.update(&billablechargepb.BillableCharge{Id: "c4", ChargePolicyVersionId: strp("v-unmapped")})
	_, err = h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("K2", "c4"))
	code(t, err, "missing_posting")
}

// AC-UC-31 (issuance half): issuing a CORRECTION charge (the row AdjustBillableCharge writes; that
// use case is tested in billable_charge) makes a CREDIT_NOTE pointing at the original document; the
// original stays untouched. A correction whose original is not ISSUED cannot be issued.
func TestCorrectionIssuesCreditNote(t *testing.T) {
	h := newHarness(false)
	h.addSeries("soa", active)
	h.addCharge("c1", "cl1", "s1", 1000, "PHP")
	h.addCorrection("corr", "c1", -600)
	_, err := h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("K-EARLY", "corr"))
	code(t, err, "not_issued")

	first, err := h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("K", "c1"))
	if err != nil {
		t.Fatal(err)
	}
	origDoc := first.Data[0]
	resp, err := h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("K-CN", "corr"))
	if err != nil || len(resp.Data) != 1 {
		t.Fatalf("issue correction: %v %v", resp, err)
	}
	cn := resp.Data[0]
	if cn.GetDocumentType() != recoverydocumentpb.RecoveryDocumentType_RECOVERY_DOCUMENT_TYPE_CREDIT_NOTE ||
		cn.GetCorrectsDocumentId() != origDoc.GetId() || cn.GetTotalAmount() != -600 {
		t.Fatalf("credit note wrong: %v", cn)
	}
	if h.charge("c1").GetAmount() != 1000 {
		t.Fatal("original amount must stay immutable")
	}
	for _, m := range h.effects.s.list(nil) {
		e := m.(*chargeeffectpb.ChargeEffect)
		if e.GetEventId() == cn.GetId() && e.GetAccountId() == "acc-recv" &&
			(e.GetDirection() != chargeeffectpb.EffectDirection_EFFECT_DIRECTION_CREDIT || e.GetAmount() != 600) {
			t.Fatalf("credit note receivable leg wrong: %v", e)
		}
	}
	det, err := h.uc.ReadRecoveryDocument.Execute(h.ctx(), &recoverydocumentpb.ReadRecoveryDocumentRequest{Data: &recoverydocumentpb.RecoveryDocument{Id: origDoc.GetId()}})
	if err != nil || len(det.Data) != 1 || len(det.Lines) != 1 || len(det.CreditNotes) != 1 || det.CreditNotes[0].GetId() != cn.GetId() {
		t.Fatalf("detail: %+v %v", det, err)
	}
}

// C12: issuance checks the series document_kind and every component's document_kind.
func TestIssueRefusesWrongDocumentKinds(t *testing.T) {
	h := newHarness(false)
	h.addSeries("soa", active)
	h.series.s.create(&documentseriespb.DocumentSeries{Id: "inv", Code: "INV", IssuerName: "Acme", DocumentKind: enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_INVOICE,
		NextNumber: 1, NumberPadding: 4, Status: active})
	h.addCharge("c1", "cl1", "s1", 500, "PHP")
	r := issueReq("k1", "c1")
	r.DocumentSeriesId = "inv"
	_, err := h.uc.IssueRecoveryDocuments.Execute(h.ctx(), r)
	code(t, err, "series_kind_mismatch")

	h.comps.s.update(&chargecomponentpb.ChargeComponent{Id: "cc-c1", DocumentKind: enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_INVOICE})
	_, err = h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("k2", "c1"))
	code(t, err, "component_kind_mismatch")
	if len(h.docs.s.list(nil)) != 0 {
		t.Fatal("refusals must not create documents")
	}
}

// C12: the same issuance key with a different request is a conflict, never the old documents.
func TestIssuanceKeyFingerprint(t *testing.T) {
	h := newHarness(false)
	h.addSeries("soa", active)
	h.addSeries("soa2", active)
	h.addCharge("c1", "cl1", "s1", 500, "PHP")
	h.addCharge("c2", "cl1", "s1", 700, "PHP")
	if _, err := h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("K", "c1")); err != nil {
		t.Fatal(err)
	}
	_, err := h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("K", "c2"))
	code(t, err, "issuance_conflict")
	_, err = h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("K", "c1", "c2"))
	code(t, err, "issuance_conflict")
	other := issueReq("K", "c1")
	other.DocumentSeriesId = "soa2"
	_, err = h.uc.IssueRecoveryDocuments.Execute(h.ctx(), other)
	code(t, err, "issuance_conflict")
	if r, err := h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("K", "c1")); err != nil || !r.Replayed {
		t.Fatalf("identical retry must replay: %v", err)
	}
	_, err = h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("bad#key", "c2"))
	code(t, err, "validation")
}

// C3: issue and void use the strict gate; an authorizer whose shadow check allows but whose strict
// check denies must refuse both, writing nothing.
func TestIssueAndVoidUseStrictGate(t *testing.T) {
	h := newHarnessWith(strictAuthz{})
	h.addSeries("soa", active)
	h.addCharge("c1", "cl1", "s1", 500, "PHP")
	if _, err := h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("K", "c1")); err == nil {
		t.Fatal("issue must use the strict gate")
	}
	if _, err := h.uc.VoidRecoveryDocument.Execute(h.ctx(), voidReq("d", "r")); err == nil {
		t.Fatal("void must use the strict gate")
	}
	if len(h.docs.s.list(nil)) != 0 || h.charge("c1").GetStatus() != billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_OPEN {
		t.Fatal("denied calls must write nothing")
	}
}

// hidden wraps a repository so its optional Locker capability is NOT visible.
type noLockCharges struct {
	billablechargepb.BillableChargeDomainServiceServer
}
type noLockSeries struct {
	documentseriespb.DocumentSeriesDomainServiceServer
}
type noLockDocs struct {
	recoverydocumentpb.RecoveryDocumentDomainServiceServer
}

// C4: a repository that cannot lock rows refuses the operation (no unlocked-read fallback).
func TestLockersFailClosed(t *testing.T) {
	for name, tamper := range map[string]func(r *Repositories){
		"charges": func(r *Repositories) { r.BillableCharge = noLockCharges{r.BillableCharge} },
		"series":  func(r *Repositories) { r.DocumentSeries = noLockSeries{r.DocumentSeries} },
	} {
		h := newHarness(false)
		h.addSeries("soa", active)
		h.addCharge("c1", "cl1", "s1", 500, "PHP")
		repos := Repositories{RecoveryDocument: h.docs, RecoveryDocumentLine: h.lines, DocumentSeries: h.series, BillableCharge: h.charges,
			ChargeComponent: h.comps, ChargePolicyPosting: h.posts, ChargeEffect: h.effects, CollectionApplication: h.apps}
		tamper(&repos)
		uc := NewUseCases(repos, h.svc)
		_, err := uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("K-"+name, "c1"))
		if err == nil || usecaseerr.IsCode(err, "not_found") {
			t.Fatalf("%s: unlockable repository must fail closed with a non-refusal error, got %v", name, err)
		}
		if h.charge("c1").GetStatus() != billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_OPEN {
			t.Fatalf("%s: nothing may be written", name)
		}
	}
	// void locks the document first
	h := newHarness(false)
	h.addSeries("soa", active)
	h.addCharge("c1", "cl1", "s1", 500, "PHP")
	resp, err := h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("K", "c1"))
	if err != nil {
		t.Fatal(err)
	}
	uc := NewUseCases(Repositories{RecoveryDocument: noLockDocs{h.docs}, RecoveryDocumentLine: h.lines, DocumentSeries: h.series, BillableCharge: h.charges,
		ChargeComponent: h.comps, ChargePolicyPosting: h.posts, ChargeEffect: h.effects, CollectionApplication: h.apps}, h.svc)
	if _, err := uc.VoidRecoveryDocument.Execute(h.ctx(), voidReq(resp.Data[0].GetId(), "r")); err == nil || usecaseerr.IsCode(err, "not_found") {
		t.Fatalf("void on an unlockable document repository must fail closed, got %v", err)
	}
}

// failingLocks makes every lock fail with a plain infrastructure error.
type failingCharges struct {
	billablechargepb.BillableChargeDomainServiceServer
}

func (failingCharges) LockBillableChargeForUpdate(context.Context, string) (*billablechargepb.BillableCharge, error) {
	return nil, errors.New("connection reset by peer")
}

// C9: a repository error is never reported as not_found.
func TestRepositoryErrorIsNotMaskedAsNotFound(t *testing.T) {
	h := newHarness(false)
	h.addSeries("soa", active)
	h.addCharge("c1", "cl1", "s1", 500, "PHP")
	uc := NewUseCases(Repositories{RecoveryDocument: h.docs, RecoveryDocumentLine: h.lines, DocumentSeries: h.series, BillableCharge: failingCharges{h.charges},
		ChargeComponent: h.comps, ChargePolicyPosting: h.posts, ChargeEffect: h.effects, CollectionApplication: h.apps}, h.svc)
	_, err := uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("K", "c1"))
	if err == nil || usecaseerr.IsCode(err, "not_found") {
		t.Fatalf("infrastructure failure must not surface as not_found: %v", err)
	}
	// a genuinely missing row still is not_found
	_, err = h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("K2", "ghost"))
	code(t, err, "not_found")
}

func TestVoidRefusedWithApplications(t *testing.T) {
	h := newHarness(false)
	h.addSeries("soa", active)
	h.addCharge("c1", "cl1", "s1", 1000, "PHP")
	resp, err := h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("K", "c1"))
	if err != nil {
		t.Fatal(err)
	}
	doc := resp.Data[0]
	_, err = h.uc.VoidRecoveryDocument.Execute(h.ctx(), voidReq(doc.GetId(), ""))
	code(t, err, "void_reason_required")

	h.apps.s.create(&collectionapplicationpb.CollectionApplication{Id: "a1", RecoveryDocumentId: strp(doc.GetId()), Status: collectionapplicationpb.ApplicationStatus_APPLICATION_STATUS_APPLIED})
	_, err = h.uc.VoidRecoveryDocument.Execute(h.ctx(), voidReq(doc.GetId(), "typo"))
	code(t, err, "has_applications")
	if d, _ := h.docs.s.read(doc.GetId()); d.(*recoverydocumentpb.RecoveryDocument).GetStatus() != recoverydocumentpb.RecoveryDocumentStatus_RECOVERY_DOCUMENT_STATUS_ISSUED {
		t.Fatal("refused void must leave the document ISSUED")
	}

	h.apps.s.update(&collectionapplicationpb.CollectionApplication{Id: "a1", Status: collectionapplicationpb.ApplicationStatus_APPLICATION_STATUS_REVERSED})
	out, err := h.uc.VoidRecoveryDocument.Execute(h.ctx(), voidReq(doc.GetId(), "typo"))
	if err != nil || out.GetData().GetStatus() != recoverydocumentpb.RecoveryDocumentStatus_RECOVERY_DOCUMENT_STATUS_VOID || out.GetData().GetVoidReason() != "typo" {
		t.Fatalf("void: %v %v", out, err)
	}
	if h.charge("c1").GetStatus() != billablechargepb.BillableChargeStatus_BILLABLE_CHARGE_STATUS_CANCELLED {
		t.Fatal("charge must be CANCELLED")
	}
	if len(h.docs.s.list(nil)) != 1 {
		t.Fatal("void must retain the row")
	}
	var dr, cr int64
	for _, m := range h.effects.s.list(nil) {
		e := m.(*chargeeffectpb.ChargeEffect)
		if e.GetEventKind().String() != "CHARGE_POSTING_EVENT_REVERSAL" {
			continue
		}
		if e.GetDirection() == chargeeffectpb.EffectDirection_EFFECT_DIRECTION_DEBIT {
			dr += e.GetAmount()
			if e.GetAccountId() != "acc-clear" {
				t.Fatalf("reversal must debit clearing: %v", e)
			}
		} else {
			cr += e.GetAmount()
		}
	}
	if dr != 1000 || cr != 1000 {
		t.Fatalf("reversal effects dr=%d cr=%d", dr, cr)
	}
	_, err = h.uc.VoidRecoveryDocument.Execute(h.ctx(), voidReq(doc.GetId(), "again"))
	code(t, err, "already_void")
	voided, err := h.uc.ListRecoveryDocuments.Execute(h.ctx(), &recoverydocumentpb.ListRecoveryDocumentsRequest{Status: recoverydocumentpb.RecoveryDocumentStatus_RECOVERY_DOCUMENT_STATUS_VOID.Enum()})
	if err != nil || len(voided.Data) != 1 {
		t.Fatalf("list docs by status: %v %v", voided, err)
	}
}

// C12: void is refused while an ISSUED credit note corrects the document; voiding the credit note
// first releases it.
func TestVoidRefusedWhileCreditNoteIssued(t *testing.T) {
	h := newHarness(false)
	h.addSeries("soa", active)
	h.addCharge("c1", "cl1", "s1", 1000, "PHP")
	first, err := h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("K", "c1"))
	if err != nil {
		t.Fatal(err)
	}
	h.addCorrection("corr", "c1", -300)
	cnResp, err := h.uc.IssueRecoveryDocuments.Execute(h.ctx(), issueReq("K-CN", "corr"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.uc.VoidRecoveryDocument.Execute(h.ctx(), voidReq(first.Data[0].GetId(), "typo"))
	code(t, err, "has_credit_notes")
	if _, err := h.uc.VoidRecoveryDocument.Execute(h.ctx(), voidReq(cnResp.Data[0].GetId(), "undo")); err != nil {
		t.Fatalf("voiding the credit note: %v", err)
	}
	if _, err := h.uc.VoidRecoveryDocument.Execute(h.ctx(), voidReq(first.Data[0].GetId(), "typo")); err != nil {
		t.Fatalf("void after the credit note is void: %v", err)
	}
}
