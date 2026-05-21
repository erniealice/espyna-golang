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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadEquityAccountRepositories(repositories)
	readServices := ReadEquityAccountServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListEquityAccountsRepositories(repositories)
	listServices := ListEquityAccountsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetEquityAccountListPageDataRepositories(repositories)
	getListPageDataServices := GetEquityAccountListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateEquityAccount:          NewCreateEquityAccountUseCase(createRepos, createServices),
		ReadEquityAccount:            NewReadEquityAccountUseCase(readRepos, readServices),
		ListEquityAccounts:           NewListEquityAccountsUseCase(listRepos, listServices),
		GetEquityAccountListPageData: NewGetEquityAccountListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
