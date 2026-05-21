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
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
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
				Authorizer:  services.Authorizer,
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				IDGenerator: services.IDGenerator,
			},
		),
		ReadAccruedExpense: NewReadAccruedExpenseUseCase(
			ReadAccruedExpenseRepositories{AccruedExpense: repositories.AccruedExpense},
			ReadAccruedExpenseServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		UpdateAccruedExpense: NewUpdateAccruedExpenseUseCase(
			UpdateAccruedExpenseRepositories{AccruedExpense: repositories.AccruedExpense},
			UpdateAccruedExpenseServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		DeleteAccruedExpense: NewDeleteAccruedExpenseUseCase(
			DeleteAccruedExpenseRepositories{AccruedExpense: repositories.AccruedExpense},
			DeleteAccruedExpenseServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		ListAccruedExpenses: NewListAccruedExpensesUseCase(
			ListAccruedExpensesRepositories{AccruedExpense: repositories.AccruedExpense},
			ListAccruedExpensesServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		AccrueFromContract: NewAccrueFromContractUseCase(
			AccrueFromContractRepositories{AccruedExpense: repositories.AccruedExpense},
			AccrueFromContractServices{
				Authorizer:  services.Authorizer,
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				IDGenerator: services.IDGenerator,
			},
		),
		ReverseAccrual: NewReverseAccrualUseCase(
			ReverseAccrualRepositories{AccruedExpense: repositories.AccruedExpense},
			ReverseAccrualServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		),
		// SPS Wave 2 Opus: SettleAccrual lives in settle_accrual.go.
		SettleAccrual: NewSettleAccrualUseCase(
			SettleAccrualRepositories{
				AccruedExpense:           repositories.AccruedExpense,
				AccruedExpenseSettlement: repositories.AccruedExpenseSettlement,
			},
			SettleAccrualServices{
				Authorizer:  services.Authorizer,
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				IDGenerator: services.IDGenerator,
			},
		),
	}
}
