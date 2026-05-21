package equitytransaction

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	equitytransactionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/equity_transaction"
)

// EquityTransactionRepositories groups all repository dependencies for equity transaction use cases.
type EquityTransactionRepositories struct {
	EquityTransaction equitytransactionpb.EquityTransactionDomainServiceServer
}

// EquityTransactionServices groups all business service dependencies for equity transaction use cases.
type EquityTransactionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all equity transaction-related use cases.
type UseCases struct {
	CreateEquityTransaction          *CreateEquityTransactionUseCase
	ListEquityTransactions           *ListEquityTransactionsUseCase
	GetEquityTransactionListPageData *GetEquityTransactionListPageDataUseCase
}

// NewUseCases creates a new collection of equity transaction use cases.
func NewUseCases(
	repositories EquityTransactionRepositories,
	services EquityTransactionServices,
) *UseCases {
	createRepos := CreateEquityTransactionRepositories(repositories)
	createServices := CreateEquityTransactionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	listRepos := ListEquityTransactionsRepositories(repositories)
	listServices := ListEquityTransactionsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetEquityTransactionListPageDataRepositories(repositories)
	getListPageDataServices := GetEquityTransactionListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateEquityTransaction:          NewCreateEquityTransactionUseCase(createRepos, createServices),
		ListEquityTransactions:           NewListEquityTransactionsUseCase(listRepos, listServices),
		GetEquityTransactionListPageData: NewGetEquityTransactionListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
