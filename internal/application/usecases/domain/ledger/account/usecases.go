package account

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	accountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account"
)

// AccountRepositories groups all repository dependencies for account use cases
type AccountRepositories struct {
	Account accountpb.AccountDomainServiceServer // Primary entity repository
}

// AccountServices groups all business service dependencies for account use cases
type AccountServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadAccountRepositories(repositories)
	readServices := ReadAccountServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateAccountRepositories(repositories)
	updateServices := UpdateAccountServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteAccountRepositories(repositories)
	deleteServices := DeleteAccountServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListAccountsRepositories(repositories)
	listServices := ListAccountsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetAccountListPageDataRepositories(repositories)
	getListPageDataServices := GetAccountListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
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
