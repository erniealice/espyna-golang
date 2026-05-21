package equityaccount

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	equityaccountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/equity_account"
)

// EquityAccountRepositories groups all repository dependencies for equity account use cases.
type EquityAccountRepositories struct {
	EquityAccount equityaccountpb.EquityAccountDomainServiceServer
}

// EquityAccountServices groups all business service dependencies for equity account use cases.
type EquityAccountServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all equity account-related use cases.
type UseCases struct {
	CreateEquityAccount          *CreateEquityAccountUseCase
	ReadEquityAccount            *ReadEquityAccountUseCase
	ListEquityAccounts           *ListEquityAccountsUseCase
	GetEquityAccountListPageData *GetEquityAccountListPageDataUseCase
}

// NewUseCases creates a new collection of equity account use cases.
func NewUseCases(
	repositories EquityAccountRepositories,
	services EquityAccountServices,
) *UseCases {
	createRepos := CreateEquityAccountRepositories(repositories)
	createServices := CreateEquityAccountServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadEquityAccountRepositories(repositories)
	readServices := ReadEquityAccountServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListEquityAccountsRepositories(repositories)
	listServices := ListEquityAccountsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetEquityAccountListPageDataRepositories(repositories)
	getListPageDataServices := GetEquityAccountListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateEquityAccount:          NewCreateEquityAccountUseCase(createRepos, createServices),
		ReadEquityAccount:            NewReadEquityAccountUseCase(readRepos, readServices),
		ListEquityAccounts:           NewListEquityAccountsUseCase(listRepos, listServices),
		GetEquityAccountListPageData: NewGetEquityAccountListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
