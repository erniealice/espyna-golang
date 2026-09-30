package recovery_document

import (
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
	collectionapplicationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_application"
	"google.golang.org/protobuf/proto"
)

// Report RPCs of RecoveryDocumentDomainService (GetRecoveryBalance, ListRecoverablesAging), moved
// here from ledger/recovery_reporting (build-spec §7c C27).

func (h *harness) reportDoc(id, client string, typ recoverydocumentpb.RecoveryDocumentType, total int64, due, corrects string) {
	d := &recoverydocumentpb.RecoveryDocument{
		Id: id, ClientId: client, DocumentType: typ, TotalAmount: total, Currency: "PHP", Active: true,
		Status: recoverydocumentpb.RecoveryDocumentStatus_RECOVERY_DOCUMENT_STATUS_ISSUED, IssueDate: proto.String("2026-08-01"), DueDate: proto.String(due),
	}
	if corrects != "" {
		d.CorrectsDocumentId = proto.String(corrects)
	}
	h.docs.s.create(d)
}

func (h *harness) reportCash(id, client, docID string, amount int64, status collectionapplicationpb.ApplicationStatus) {
	h.apps.s.create(&collectionapplicationpb.CollectionApplication{
		Id: id, ClientId: client, Amount: amount, Currency: "PHP", Active: true,
		TargetKind:         collectionapplicationpb.ApplicationTargetKind_APPLICATION_TARGET_KIND_RECOVERY_DOCUMENT,
		RecoveryDocumentId: proto.String(docID),
		ApplicationKind:    collectionapplicationpb.ApplicationKind_APPLICATION_KIND_CASH,
		Status:             status,
	})
}

const (
	statementDoc  = recoverydocumentpb.RecoveryDocumentType_RECOVERY_DOCUMENT_TYPE_STATEMENT
	creditNoteDoc = recoverydocumentpb.RecoveryDocumentType_RECOVERY_DOCUMENT_TYPE_CREDIT_NOTE
	appliedCash   = collectionapplicationpb.ApplicationStatus_APPLICATION_STATUS_APPLIED
	reversedCash  = collectionapplicationpb.ApplicationStatus_APPLICATION_STATUS_REVERSED
)

// AC-UC-31 (cash half): 100 issued and paid, credit note -20 => balance -20 (a credit), cash conserved.
func TestCorrectionAfterPaymentLeavesCredit(t *testing.T) {
	h := newHarness(false)
	h.reportDoc("d1", "c1", statementDoc, 100, "2026-08-31", "")
	h.reportCash("a1", "c1", "d1", 100, appliedCash)
	h.reportDoc("d1-cn", "c1", creditNoteDoc, -20, "", "d1")
	got, err := h.uc.GetRecoveryBalance.Execute(h.ctx(), &recoverydocumentpb.GetRecoveryBalanceRequest{ClientId: "c1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ByCurrency) != 1 || got.ByCurrency[0].Billed != 80 || got.ByCurrency[0].Applied != 100 || got.ByCurrency[0].Balance != -20 {
		t.Fatalf("balance = %+v, want billed 80 applied 100 balance -20", got.ByCurrency)
	}
	// the corrected document is not an open item any more
	ag, err := h.uc.ListRecoverablesAging.Execute(h.ctx(), &recoverydocumentpb.ListRecoverablesAgingRequest{AsOf: proto.String("2026-12-31")})
	if err != nil || len(ag.Rows) != 0 {
		t.Fatalf("aging = %+v err=%v; a fully-paid corrected document must not age", ag, err)
	}
}

func TestReverseApplicationsExcludedAndOtherClientIgnored(t *testing.T) {
	h := newHarness(false)
	h.reportDoc("d1", "c1", statementDoc, 100, "2026-08-31", "")
	h.reportDoc("d2", "c2", statementDoc, 50, "2026-08-31", "")
	h.reportCash("a1", "c1", "d1", 40, appliedCash)
	h.reportCash("a2", "c1", "d1", 60, reversedCash) // a reversed application no longer counts
	got, err := h.uc.GetRecoveryBalance.Execute(h.ctx(), &recoverydocumentpb.GetRecoveryBalanceRequest{ClientId: "c1"})
	if err != nil || got.ByCurrency[0].Balance != 60 {
		t.Fatalf("balance = %+v err=%v, want 60", got, err)
	}
}

func TestRecoverablesAgingBuckets(t *testing.T) {
	h := newHarness(false)
	// asOf 2026-12-31: due 2026-12-15 (16d), 2026-11-15 (46d), 2026-10-15 (77d), 2026-08-01 (152d)
	h.reportDoc("d1", "c1", statementDoc, 10, "2026-12-15", "")
	h.reportDoc("d2", "c1", statementDoc, 20, "2026-11-15", "")
	h.reportDoc("d3", "c1", statementDoc, 30, "2026-10-15", "")
	h.reportDoc("d4", "c1", statementDoc, 40, "2026-08-01", "")
	h.reportCash("a", "c1", "d4", 15, appliedCash) // partially paid: 25 left
	ag, err := h.uc.ListRecoverablesAging.Execute(h.ctx(), &recoverydocumentpb.ListRecoverablesAgingRequest{AsOf: proto.String("2026-12-31")})
	if err != nil {
		t.Fatal(err)
	}
	if len(ag.Rows) != 1 {
		t.Fatalf("rows = %+v", ag.Rows)
	}
	b := ag.Rows[0]
	if b.Days_0_30 != 10 || b.Days_31_60 != 20 || b.Days_61_90 != 30 || b.DaysOver_90 != 25 || b.Total != 85 || ag.Totals[0].Total != 85 || ag.Totals[0].ClientId != nil {
		t.Fatalf("buckets = %+v total=%d", b, b.Total)
	}
	// as-of before issue excludes the documents
	ag, _ = h.uc.ListRecoverablesAging.Execute(h.ctx(), &recoverydocumentpb.ListRecoverablesAgingRequest{AsOf: proto.String("2026-07-31")})
	if len(ag.Rows) != 0 {
		t.Fatalf("documents issued after as-of must be excluded: %+v", ag.Rows)
	}
}

func TestRecoveryReportsFailClosedWhenDenied(t *testing.T) {
	h := newHarness(true)
	if _, err := h.uc.GetRecoveryBalance.Execute(h.ctx(), &recoverydocumentpb.GetRecoveryBalanceRequest{ClientId: "c1"}); err == nil {
		t.Fatal("balance must be denied")
	}
	if _, err := h.uc.ListRecoverablesAging.Execute(h.ctx(), nil); err == nil {
		t.Fatal("aging must be denied")
	}
}

// C17: a client outside the caller's row scope (ReadClient reports it as not found) is refused by the
// balance and the client-filtered aging before any list is read.
func TestReportsRefuseClientOutsideRowScope(t *testing.T) {
	h := newHarness(false)
	h.reportDoc("d9", "c9", statementDoc, 100, "2026-08-31", "") // c9 has documents but is not readable by the actor
	if _, err := h.uc.GetRecoveryBalance.Execute(h.ctx(), &recoverydocumentpb.GetRecoveryBalanceRequest{ClientId: "c9"}); !usecaseerr.IsCode(err, "not_found") {
		t.Fatalf("balance err = %v, want not_found", err)
	}
	c9 := "c9"
	if _, err := h.uc.ListRecoverablesAging.Execute(h.ctx(), &recoverydocumentpb.ListRecoverablesAgingRequest{ClientId: &c9}); !usecaseerr.IsCode(err, "not_found") {
		t.Fatalf("aging err = %v, want not_found", err)
	}
}
