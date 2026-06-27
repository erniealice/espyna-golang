package equitytransaction

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
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
	ActionGatekeeper *actiongate.ActionGatekeeper
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
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	listRepos := ListEquityTransactionsRepositories(repositories)
	listServices := ListEquityTransactionsServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetEquityTransactionListPageDataRepositories(repositories)
	getListPageDataServices := GetEquityTransactionListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
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
