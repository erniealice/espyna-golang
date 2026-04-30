package accruedexpense

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	accruedexpensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/accrued_expense"
)

// AccruedExpenseRepositories groups all repository dependencies.
type AccruedExpenseRepositories struct {
	AccruedExpense           accruedexpensepb.AccruedExpenseDomainServiceServer
	AccruedExpenseSettlement accruedexpensepb.AccruedExpenseSettlementDomainServiceServer
}

// AccruedExpenseServices groups all service dependencies.
type AccruedExpenseServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all accrued expense use cases.
type UseCases struct {
	CreateAccruedExpense *CreateAccruedExpenseUseCase
	ReadAccruedExpense   *ReadAccruedExpenseUseCase
	UpdateAccruedExpense *UpdateAccruedExpenseUseCase
	DeleteAccruedExpense *DeleteAccruedExpenseUseCase
	ListAccruedExpenses  *ListAccruedExpensesUseCase
	AccrueFromContract   *AccrueFromContractUseCase
	ReverseAccrual       *ReverseAccrualUseCase
	// SPS Wave 2 Opus: SettleAccrual registered separately
	SettleAccrual *SettleAccrualUseCase
}

// NewUseCases creates a new collection of accrued expense use cases.
func NewUseCases(
	repositories AccruedExpenseRepositories,
	services AccruedExpenseServices,
) *UseCases {
	return &UseCases{
		CreateAccruedExpense: NewCreateAccruedExpenseUseCase(
			CreateAccruedExpenseRepositories{AccruedExpense: repositories.AccruedExpense},
			CreateAccruedExpenseServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		),
		ReadAccruedExpense: NewReadAccruedExpenseUseCase(
			ReadAccruedExpenseRepositories{AccruedExpense: repositories.AccruedExpense},
			ReadAccruedExpenseServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		UpdateAccruedExpense: NewUpdateAccruedExpenseUseCase(
			UpdateAccruedExpenseRepositories{AccruedExpense: repositories.AccruedExpense},
			UpdateAccruedExpenseServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		DeleteAccruedExpense: NewDeleteAccruedExpenseUseCase(
			DeleteAccruedExpenseRepositories{AccruedExpense: repositories.AccruedExpense},
			DeleteAccruedExpenseServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		ListAccruedExpenses: NewListAccruedExpensesUseCase(
			ListAccruedExpensesRepositories{AccruedExpense: repositories.AccruedExpense},
			ListAccruedExpensesServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		AccrueFromContract: NewAccrueFromContractUseCase(
			AccrueFromContractRepositories{AccruedExpense: repositories.AccruedExpense},
			AccrueFromContractServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		),
		ReverseAccrual: NewReverseAccrualUseCase(
			ReverseAccrualRepositories{AccruedExpense: repositories.AccruedExpense},
			ReverseAccrualServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
			},
		),
		// SPS Wave 2 Opus: SettleAccrual lives in settle_accrual.go.
		SettleAccrual: NewSettleAccrualUseCase(
			SettleAccrualRepositories{
				AccruedExpense:           repositories.AccruedExpense,
				AccruedExpenseSettlement: repositories.AccruedExpenseSettlement,
			},
			SettleAccrualServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		),
	}
}
