package balance

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
)

// BalanceRepositories groups all repository dependencies for balance use cases
type BalanceRepositories struct {
	Balance balancepb.BalanceDomainServiceServer // Primary entity repository
}

// BalanceServices groups all business service dependencies for balance use cases
type BalanceServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService // Only for CreateBalance
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadBalanceRepositories(repositories)
	readServices := ReadBalanceServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateBalanceRepositories(repositories)
	updateServices := UpdateBalanceServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteBalanceRepositories(repositories)
	deleteServices := DeleteBalanceServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListBalancesRepositories(repositories)
	listServices := ListBalancesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetBalanceListPageDataRepositories(repositories)
	getListPageDataServices := GetBalanceListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getItemPageDataRepos := GetBalanceItemPageDataRepositories(repositories)
	getItemPageDataServices := GetBalanceItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
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
