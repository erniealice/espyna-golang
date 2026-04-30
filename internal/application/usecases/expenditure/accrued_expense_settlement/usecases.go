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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		),
		ReadAccruedExpenseSettlement: NewReadAccruedExpenseSettlementUseCase(
			ReadAccruedExpenseSettlementRepositories{AccruedExpenseSettlement: repositories.AccruedExpenseSettlement},
			ReadAccruedExpenseSettlementServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		UpdateAccruedExpenseSettlement: NewUpdateAccruedExpenseSettlementUseCase(
			UpdateAccruedExpenseSettlementRepositories{AccruedExpenseSettlement: repositories.AccruedExpenseSettlement},
			UpdateAccruedExpenseSettlementServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		DeleteAccruedExpenseSettlement: NewDeleteAccruedExpenseSettlementUseCase(
			DeleteAccruedExpenseSettlementRepositories{AccruedExpenseSettlement: repositories.AccruedExpenseSettlement},
			DeleteAccruedExpenseSettlementServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		ListAccruedExpenseSettlements: NewListAccruedExpenseSettlementsUseCase(
			ListAccruedExpenseSettlementsRepositories{AccruedExpenseSettlement: repositories.AccruedExpenseSettlement},
			ListAccruedExpenseSettlementsServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
	}
}
