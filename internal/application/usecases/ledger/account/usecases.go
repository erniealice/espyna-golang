package account

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	accountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account"
)

// AccountRepositories groups all repository dependencies for account use cases
type AccountRepositories struct {
	Account accountpb.AccountDomainServiceServer // Primary entity repository
}

// AccountServices groups all business service dependencies for account use cases
type AccountServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all account-related use cases
type UseCases struct {
	CreateAccount          *CreateAccountUseCase
	ReadAccount            *ReadAccountUseCase
	UpdateAccount          *UpdateAccountUseCase
	DeleteAccount          *DeleteAccountUseCase
	ListAccounts           *ListAccountsUseCase
	GetAccountListPageData *GetAccountListPageDataUseCase
}

// NewUseCases creates a new collection of account use cases
func NewUseCases(
	repositories AccountRepositories,
	services AccountServices,
) *UseCases {
	createRepos := CreateAccountRepositories(repositories)
	createServices := CreateAccountServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadAccountRepositories(repositories)
	readServices := ReadAccountServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateAccountRepositories(repositories)
	updateServices := UpdateAccountServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteAccountRepositories(repositories)
	deleteServices := DeleteAccountServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListAccountsRepositories(repositories)
	listServices := ListAccountsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetAccountListPageDataRepositories(repositories)
	getListPageDataServices := GetAccountListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateAccount:          NewCreateAccountUseCase(createRepos, createServices),
		ReadAccount:            NewReadAccountUseCase(readRepos, readServices),
		UpdateAccount:          NewUpdateAccountUseCase(updateRepos, updateServices),
		DeleteAccount:          NewDeleteAccountUseCase(deleteRepos, deleteServices),
		ListAccounts:           NewListAccountsUseCase(listRepos, listServices),
		GetAccountListPageData: NewGetAccountListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
