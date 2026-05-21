package accruedexpensesettlement

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	accruedexpensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/accrued_expense"
)

// AccruedExpenseSettlementRepositories groups all repository dependencies.
type AccruedExpenseSettlementRepositories struct {
	AccruedExpenseSettlement accruedexpensepb.AccruedExpenseSettlementDomainServiceServer
}

// AccruedExpenseSettlementServices groups all service dependencies.
type AccruedExpenseSettlementServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all accrued-expense-settlement use cases.
type UseCases struct {
	CreateAccruedExpenseSettlement *CreateAccruedExpenseSettlementUseCase
	ReadAccruedExpenseSettlement   *ReadAccruedExpenseSettlementUseCase
	UpdateAccruedExpenseSettlement *UpdateAccruedExpenseSettlementUseCase
	DeleteAccruedExpenseSettlement *DeleteAccruedExpenseSettlementUseCase
	ListAccruedExpenseSettlements  *ListAccruedExpenseSettlementsUseCase
}

// NewUseCases creates a new collection of accrued expense settlement use cases.
func NewUseCases(
	repositories AccruedExpenseSettlementRepositories,
	services AccruedExpenseSettlementServices,
) *UseCases {
	return &UseCases{
		CreateAccruedExpenseSettlement: NewCreateAccruedExpenseSettlementUseCase(
			CreateAccruedExpenseSettlementRepositories{AccruedExpenseSettlement: repositories.AccruedExpenseSettlement},
			CreateAccruedExpenseSettlementServices{
				Authorizer:  services.Authorizer,
				Translator:  services.Translator,
				IDGenerator: services.IDGenerator,
			},
		),
		ReadAccruedExpenseSettlement: NewReadAccruedExpenseSettlementUseCase(
			ReadAccruedExpenseSettlementRepositories{AccruedExpenseSettlement: repositories.AccruedExpenseSettlement},
			ReadAccruedExpenseSettlementServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		UpdateAccruedExpenseSettlement: NewUpdateAccruedExpenseSettlementUseCase(
			UpdateAccruedExpenseSettlementRepositories{AccruedExpenseSettlement: repositories.AccruedExpenseSettlement},
			UpdateAccruedExpenseSettlementServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		DeleteAccruedExpenseSettlement: NewDeleteAccruedExpenseSettlementUseCase(
			DeleteAccruedExpenseSettlementRepositories{AccruedExpenseSettlement: repositories.AccruedExpenseSettlement},
			DeleteAccruedExpenseSettlementServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		ListAccruedExpenseSettlements: NewListAccruedExpenseSettlementsUseCase(
			ListAccruedExpenseSettlementsRepositories{AccruedExpenseSettlement: repositories.AccruedExpenseSettlement},
			ListAccruedExpenseSettlementsServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
	}
}
