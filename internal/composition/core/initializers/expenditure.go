package initializers

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/expenditure"
	"github.com/erniealice/espyna-golang/internal/application/usecases/treasury"
	treasurydisbursement "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/treasury_disbursement"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// treasuryAmortizeAdvanceDisbursement returns the AmortizeAdvanceDisbursement
// use case from the treasury aggregate, nil-safe on a nil aggregate.
func treasuryAmortizeAdvanceDisbursement(t *treasury.TreasuryUseCases) *treasurydisbursement.AmortizeAdvanceDisbursementUseCase {
	if t == nil {
		return nil
	}
	return t.AmortizeAdvanceDisbursement
}

// InitializeExpenditure creates all expenditure use cases from provider repositories.
//
// 20260517-expense-run Plan A Phase 4: GenerateExpenseRun composes the
// cross-domain treasury AmortizeAdvanceDisbursement use case. The treasury
// aggregate is built before expenditure (see usecases.go), so the caller
// passes the already-constructed pointer through.
func InitializeExpenditure(
	repos *domain.ExpenditureRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
	treasuryUseCases *treasury.TreasuryUseCases,
) (*expenditure.ExpenditureUseCases, error) {
	// AmortizeAdvanceDisbursement is the cross-domain composition target for
	// GenerateExpenseRun (Plan A Phase 4). Nil when treasury isn't initialized.
	var amortizeAdvDis = treasuryAmortizeAdvanceDisbursement(treasuryUseCases)
	return expenditure.NewUseCases(
		expenditure.ExpenditureRepositories{
			Expenditure:            repos.Expenditure,
			ExpenditureLineItem:    repos.ExpenditureLineItem,
			ExpenditureCategory:    repos.ExpenditureCategory,
			ExpenditureAttribute:   repos.ExpenditureAttribute,
			Prepayment:             repos.Prepayment,
			PurchaseOrder:          repos.PurchaseOrder,
			PurchaseOrderLineItem:  repos.PurchaseOrderLineItem,
			SupplierContract:       repos.SupplierContract,
			SupplierContractLine:   repos.SupplierContractLine,
			ProcurementRequest:     repos.ProcurementRequest,
			ProcurementRequestLine: repos.ProcurementRequestLine,
			// SPS Wave 2 (2026-04-30)
			SupplierContractPriceSchedule:     repos.SupplierContractPriceSchedule,
			SupplierContractPriceScheduleLine: repos.SupplierContractPriceScheduleLine,
			ExpenseRecognition:                repos.ExpenseRecognition,
			ExpenseRecognitionLine:            repos.ExpenseRecognitionLine,
			AccruedExpense:                    repos.AccruedExpense,
			AccruedExpenseSettlement:          repos.AccruedExpenseSettlement,
			PaymentTerm:                       repos.PaymentTerm,
			SupplierSubscription:              repos.SupplierSubscription,
			// 20260517-expense-run Plan A Phase 2 + Phase 4 — cross-domain repos.
			CostPlan:                repos.CostPlan,
			SupplierProductCostPlan: repos.SupplierProductCostPlan,
			TreasuryDisbursement:    repos.TreasuryDisbursement,
			// 20260517-expense-run Plan A Phase 4 — in-domain run repo.
			ExpenseRecognitionRun: repos.ExpenseRecognitionRun,
		},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
		amortizeAdvDis,
	), nil
}
