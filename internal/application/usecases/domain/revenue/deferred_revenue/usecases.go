package deferredrevenue

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	deferredrevenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/deferred_revenue"
)

// DeferredRevenueRepositories groups all repository dependencies for deferred revenue use cases
type DeferredRevenueRepositories struct {
	DeferredRevenue deferredrevenuepb.DeferredRevenueDomainServiceServer // Primary entity repository
}

// DeferredRevenueServices groups all business service dependencies for deferred revenue use cases
type DeferredRevenueServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all deferred revenue-related use cases
type UseCases struct {
	CreateDeferredRevenue          *CreateDeferredRevenueUseCase
	ListDeferredRevenues           *ListDeferredRevenuesUseCase
	GetDeferredRevenueListPageData *GetDeferredRevenueListPageDataUseCase
}

// NewUseCases creates a new collection of deferred revenue use cases
func NewUseCases(
	repositories DeferredRevenueRepositories,
	services DeferredRevenueServices,
) *UseCases {
	createRepos := CreateDeferredRevenueRepositories(repositories)
	createServices := CreateDeferredRevenueServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	listRepos := ListDeferredRevenuesRepositories(repositories)
	listServices := ListDeferredRevenuesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetDeferredRevenueListPageDataRepositories(repositories)
	getListPageDataServices := GetDeferredRevenueListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateDeferredRevenue:          NewCreateDeferredRevenueUseCase(createRepos, createServices),
		ListDeferredRevenues:           NewListDeferredRevenuesUseCase(listRepos, listServices),
		GetDeferredRevenueListPageData: NewGetDeferredRevenueListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
