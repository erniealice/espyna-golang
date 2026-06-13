package balance

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
)

// BalanceRepositories groups all repository dependencies for balance use cases
type BalanceRepositories struct {
	Balance balancepb.BalanceDomainServiceServer // Primary entity repository
}

// BalanceServices groups all business service dependencies for balance use cases
type BalanceServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator // Only for CreateBalance
}

// UseCases contains all balance-related use cases
type UseCases struct {
	CreateBalance          *CreateBalanceUseCase
	ReadBalance            *ReadBalanceUseCase
	UpdateBalance          *UpdateBalanceUseCase
	DeleteBalance          *DeleteBalanceUseCase
	ListBalances           *ListBalancesUseCase
	GetBalanceListPageData *GetBalanceListPageDataUseCase
	GetBalanceItemPageData *GetBalanceItemPageDataUseCase
}

// NewUseCases creates a new collection of balance use cases
func NewUseCases(
	repositories BalanceRepositories,
	services BalanceServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateBalanceRepositories(repositories)
	createServices := CreateBalanceServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadBalanceRepositories(repositories)
	readServices := ReadBalanceServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateBalanceRepositories(repositories)
	updateServices := UpdateBalanceServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteBalanceRepositories(repositories)
	deleteServices := DeleteBalanceServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListBalancesRepositories(repositories)
	listServices := ListBalancesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetBalanceListPageDataRepositories(repositories)
	getListPageDataServices := GetBalanceListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getItemPageDataRepos := GetBalanceItemPageDataRepositories(repositories)
	getItemPageDataServices := GetBalanceItemPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateBalance:          NewCreateBalanceUseCase(createRepos, createServices),
		ReadBalance:            NewReadBalanceUseCase(readRepos, readServices),
		UpdateBalance:          NewUpdateBalanceUseCase(updateRepos, updateServices),
		DeleteBalance:          NewDeleteBalanceUseCase(deleteRepos, deleteServices),
		ListBalances:           NewListBalancesUseCase(listRepos, listServices),
		GetBalanceListPageData: NewGetBalanceListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetBalanceItemPageData: NewGetBalanceItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}
