package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/treasury"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeTreasury creates all treasury use cases from provider repositories
func InitializeTreasury(
	repos *domain.TreasuryRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idSvc ports.IDGenerator,
) (*treasury.TreasuryUseCases, error) {
	return treasury.NewUseCases(
		treasury.TreasuryRepositories{
			// Existing treasury repositories
			Collection:           repos.Collection,
			Disbursement:         repos.Disbursement,
			DisbursementSchedule: repos.DisbursementSchedule,

			// Treasury-domain-rebuild Stage 1 — method management templates.
			CollectionMethod:   repos.CollectionMethod,
			DisbursementMethod: repos.DisbursementMethod,

			// Loans & Petty Cash repositories
			Loan:                   repos.Loan,
			LoanPayment:            repos.LoanPayment,
			SecurityDeposit:        repos.SecurityDeposit,
			PettyCashFund:          repos.PettyCashFund,
			PettyCashVoucher:       repos.PettyCashVoucher,
			PettyCashReplenishment: repos.PettyCashReplenishment,

			// Tax extension
			WithholdingCertificate: repos.WithholdingCertificate,

			// 20260517-advance-cash-events Plan B Phase 2 — cross-domain.
			Revenue:            repos.Revenue,
			ExpenseRecognition: repos.ExpenseRecognition,

			// 20260517-advance-cash-events Plan B Phase 7 — MILESTONE recognize.
			// BillingEvent is wired post-construction (its provider lives under
			// the subscription domain); the three repositories below come from
			// the treasury + expenditure provider blocks.
			BillingEvent:                     repos.BillingEvent,
			SupplierBillingEvent:             repos.SupplierBillingEvent,
			CollectionBillingEvent:           repos.CollectionBillingEvent,
			DisbursementSupplierBillingEvent: repos.DisbursementSupplierBillingEvent,
		},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
	), nil
}
