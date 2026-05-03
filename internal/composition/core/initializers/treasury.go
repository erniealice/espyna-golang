package initializers

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/treasury"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeTreasury creates all treasury use cases from provider repositories
func InitializeTreasury(
	repos *domain.TreasuryRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*treasury.TreasuryUseCases, error) {
	return treasury.NewUseCases(
		treasury.TreasuryRepositories{
			// Existing treasury repositories
			Collection:           repos.Collection,
			Disbursement:         repos.Disbursement,
			DisbursementSchedule: repos.DisbursementSchedule,

			// Loans & Petty Cash repositories
			Loan:                   repos.Loan,
			LoanPayment:            repos.LoanPayment,
			SecurityDeposit:        repos.SecurityDeposit,
			PettyCashFund:          repos.PettyCashFund,
			PettyCashVoucher:       repos.PettyCashVoucher,
			PettyCashReplenishment: repos.PettyCashReplenishment,
		},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
	), nil
}
