package deferredrevenue

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	deferredrevenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/deferred_revenue"
)

// DeferredRevenueRepositories groups all repository dependencies for deferred revenue use cases
type DeferredRevenueRepositories struct {
	DeferredRevenue deferredrevenuepb.DeferredRevenueDomainServiceServer // Primary entity repository
}

// DeferredRevenueServices groups all business service dependencies for deferred revenue use cases
type DeferredRevenueServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	listRepos := ListDeferredRevenuesRepositories(repositories)
	listServices := ListDeferredRevenuesServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetDeferredRevenueListPageDataRepositories(repositories)
	getListPageDataServices := GetDeferredRevenueListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateDeferredRevenue:          NewCreateDeferredRevenueUseCase(createRepos, createServices),
		ListDeferredRevenues:           NewListDeferredRevenuesUseCase(listRepos, listServices),
		GetDeferredRevenueListPageData: NewGetDeferredRevenueListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
