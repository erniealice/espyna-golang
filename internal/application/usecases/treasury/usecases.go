package treasury

import (
	// Collection use cases
	collectionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/collection"
	// Disbursement use cases
	disbursementUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/disbursement"
	// DisbursementSchedule use cases
	disbursementscheduleUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/disbursement_schedule"
	// PettyCash use cases
	pettyCashUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/petty_cash"
	// SecurityDeposit use cases
	securityDepositUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/security_deposit"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Dashboard use cases
	cashdashboard "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/collection/dashboard"
	loandashboard "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/dashboard"

	// Protobuf domain services for treasury repositories
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
	disbursementschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement_schedule"
	loanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan"
	loanpaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan_payment"
	pettycashfundpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/petty_cash_fund"
	pettycashreplenishmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/petty_cash_replenishment"
	pettycashvoucherpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/petty_cash_voucher"
	securitydepositpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/security_deposit"
)

// TreasuryRepositories contains all treasury domain repositories
type TreasuryRepositories struct {
	// Existing treasury repositories
	Collection           collectionpb.CollectionDomainServiceServer
	Disbursement         disbursementpb.DisbursementDomainServiceServer
	DisbursementSchedule disbursementschedulepb.DisbursementScheduleDomainServiceServer

	// Loans & Petty Cash repositories
	Loan                   loanpb.LoanDomainServiceServer
	LoanPayment            loanpaymentpb.LoanPaymentDomainServiceServer
	SecurityDeposit        securitydepositpb.SecurityDepositDomainServiceServer
	PettyCashFund          pettycashfundpb.PettyCashFundDomainServiceServer
	PettyCashVoucher       pettycashvoucherpb.PettyCashVoucherDomainServiceServer
	PettyCashReplenishment pettycashreplenishmentpb.PettyCashReplenishmentDomainServiceServer
}

// TreasuryUseCases contains all treasury-related use cases
type TreasuryUseCases struct {
	Collection           *collectionUseCases.UseCases
	Disbursement         *disbursementUseCases.UseCases
	DisbursementSchedule *disbursementscheduleUseCases.UseCases
	SecurityDeposit      *securityDepositUseCases.UseCases
	PettyCash            *pettyCashUseCases.UseCases
	// Loans — use cases to be created in future iterations
	// Loan, LoanPayment, PettyCashVoucher, PettyCashReplenishment

	// Dashboard use cases (nil when postgres build tag is inactive).
	LoanDashboard *loandashboard.GetLoanDashboardPageDataUseCase
	CashDashboard *cashdashboard.GetCashDashboardPageDataUseCase
}

// NewUseCases creates all treasury use cases with proper constructor injection
func NewUseCases(
	repos TreasuryRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
) *TreasuryUseCases {
	collectionUC := collectionUseCases.NewUseCases(
		collectionUseCases.CollectionRepositories{
			Collection: repos.Collection,
		},
		collectionUseCases.CollectionServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	disbursementUC := disbursementUseCases.NewUseCases(
		disbursementUseCases.DisbursementRepositories{
			Disbursement: repos.Disbursement,
		},
		disbursementUseCases.DisbursementServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	var disbursementScheduleUC *disbursementscheduleUseCases.UseCases
	if repos.DisbursementSchedule != nil {
		disbursementScheduleUC = disbursementscheduleUseCases.NewUseCases(
			disbursementscheduleUseCases.DisbursementScheduleRepositories{
				DisbursementSchedule: repos.DisbursementSchedule,
			},
			disbursementscheduleUseCases.DisbursementScheduleServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
				IDService:            idService,
			},
		)
	}

	securityDepositUC := securityDepositUseCases.NewUseCases(
		securityDepositUseCases.SecurityDepositRepositories{
			SecurityDeposit: repos.SecurityDeposit,
		},
		securityDepositUseCases.SecurityDepositServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	pettyCashUC := pettyCashUseCases.NewUseCases(
		pettyCashUseCases.PettyCashRepositories{
			PettyCashFund: repos.PettyCashFund,
		},
		pettyCashUseCases.PettyCashServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	// Wire loan dashboard via type assertions on loan repos.
	var loanDash *loandashboard.GetLoanDashboardPageDataUseCase
	if repos.Loan != nil && repos.LoanPayment != nil {
		loanQ, lOK := repos.Loan.(loandashboard.LoanDashboardQueries)
		pmtQ, pOK := repos.LoanPayment.(loandashboard.LoanPaymentDashboardQueries)
		if lOK && pOK {
			loanDash = loandashboard.NewGetLoanDashboardPageDataUseCase(loanQ, pmtQ)
		}
	}

	// Wire cash dashboard via type assertion on collection repo.
	var cashDash *cashdashboard.GetCashDashboardPageDataUseCase
	if repos.Collection != nil {
		if collQ, ok := repos.Collection.(cashdashboard.CollectionDashboardQueries); ok {
			cashDash = cashdashboard.NewGetCashDashboardPageDataUseCase(collQ)
		}
	}

	return &TreasuryUseCases{
		Collection:           collectionUC,
		Disbursement:         disbursementUC,
		DisbursementSchedule: disbursementScheduleUC,
		SecurityDeposit:      securityDepositUC,
		PettyCash:            pettyCashUC,
		LoanDashboard:        loanDash,
		CashDashboard:        cashDash,
	}
}
