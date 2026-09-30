package expenserecognition

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
)

// The recognition header has NOT NULL internal_id, name and recognition_date on the postgres
// schema; the contract-cycle recognizer must set them (same class as the from-expenditure header).
func TestRecognizeFromContractHeaderCarriesRequiredIdentity(t *testing.T) {
	recs := &recRepo{}
	uc := NewRecognizeFromContractUseCase(
		RecognizeFromContractRepositories{ExpenseRecognition: recs},
		RecognizeFromContractServices{
			Authorizer: ports.NewNoOpAuthorizer(), Transactor: &txRunner{}, Translator: ports.NewNoOpTranslator(), IDGenerator: &seq{},
			ActionGatekeeper: actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator()),
		})
	_, err := uc.Execute(contextutil.WithUserID(context.Background(), "u1"), &expenserecognitionpb.RecognizeFromContractRequest{
		SupplierContractId: "sc-1", CycleDate: "2026-02-15", Amount: func() *int64 { v := int64(1000); return &v }()})
	if err != nil {
		t.Fatal(err)
	}
	h := recs.created[0]
	if h.GetInternalId() == "" || h.GetName() == "" || h.GetRecognitionDate() == nil {
		t.Fatalf("header misses internal_id/name/recognition_date: %+v", h)
	}
	if got := h.GetRecognitionDate().AsTime().Format("2006-01-02"); got != "2026-02-15" {
		t.Fatalf("recognition_date = %s, want the cycle date 2026-02-15", got)
	}
	if h.GetInternalId() == h.GetId() {
		t.Fatalf("internal_id must be its own generated value, got id == internal_id")
	}
}
