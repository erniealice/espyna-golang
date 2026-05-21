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
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	listRepos := ListEquityTransactionsRepositories(repositories)
	listServices := ListEquityTransactionsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetEquityTransactionListPageDataRepositories(repositories)
	getListPageDataServices := GetEquityTransactionListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateEquityTransaction:          NewCreateEquityTransactionUseCase(createRepos, createServices),
		ListEquityTransactions:           NewListEquityTransactionsUseCase(listRepos, listServices),
		GetEquityTransactionListPageData: NewGetEquityTransactionListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
