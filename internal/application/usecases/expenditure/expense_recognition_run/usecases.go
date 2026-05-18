package expenserecognitionrun

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	expenserecognitionrunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_run"
)

// ExpenseRecognitionRunRepositories groups the repository dependencies that
// can be wired from the expenditure domain repos alone. The cross-domain repos
// (SupplierSubscription / CostPlan / TreasuryDisbursement / Expenditure) plus
// the cross-domain composed use cases (RecognizeExpenseFromSupplierSubscription
// + AmortizeAdvanceDisbursement) are supplied by the parent expenditure
// aggregator at post-construction time — see expenditure/usecases.go where
// the two use cases below are assigned onto this struct after NewUseCases.
type ExpenseRecognitionRunRepositories struct {
	ExpenseRecognitionRun expenserecognitionrunpb.ExpenseRecognitionRunDomainServiceServer
}

// ExpenseRecognitionRunServices groups the infra services. Kept minimal here
// — the inner use cases pull what they need from the parent aggregator's
// constructor calls.
type ExpenseRecognitionRunServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all expense-recognition-run use cases.
//
// 20260518-hexagonal-strict-adherence Phase 3 — F6 closure. The two use cases
// below were previously carried as flat fields on ExpenditureUseCases
// (ListExpenseRunCandidates + GenerateExpenseRun). They are now nested under
// this entity sub-aggregate, mirroring the treasury/collection pattern.
//
// Both fields are nil-safe; consumers must check before invoking. The parent
// expenditure aggregator populates them when the required cross-domain repos
// are wired (see expenditure/usecases.go).
type UseCases struct {
	ListExpenseRunCandidates *ListExpenseRunCandidatesUseCase
	GenerateExpenseRun       *GenerateExpenseRunUseCase
}

// NewUseCases creates a new expense-recognition-run sub-aggregate. Returns an
// empty struct — the parent expenditure aggregator assigns the inner use
// cases post-construction because both require cross-domain dependencies
// (procurement.SupplierSubscription / treasury.Disbursement / etc.).
func NewUseCases(
	_ ExpenseRecognitionRunRepositories,
	_ ExpenseRecognitionRunServices,
) *UseCases {
	return &UseCases{}
}
