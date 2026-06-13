package revenuecategory

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_category"
)

// RevenueCategoryRepositories groups all repository dependencies
type RevenueCategoryRepositories struct {
	RevenueCategory pb.RevenueCategoryDomainServiceServer
}

// RevenueCategoryServices groups all business service dependencies
type RevenueCategoryServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all revenue category use cases
type UseCases struct {
	CreateRevenueCategory *CreateRevenueCategoryUseCase
	ReadRevenueCategory   *ReadRevenueCategoryUseCase
	UpdateRevenueCategory *UpdateRevenueCategoryUseCase
	DeleteRevenueCategory *DeleteRevenueCategoryUseCase
	ListRevenueCategories *ListRevenueCategoriesUseCase
}

// NewUseCases creates a new collection of revenue category use cases
func NewUseCases(
	repositories RevenueCategoryRepositories,
	services RevenueCategoryServices,
) *UseCases {
	createRepos := CreateRevenueCategoryRepositories(repositories)
	createServices := CreateRevenueCategoryServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadRevenueCategoryRepositories(repositories)
	readServices := ReadRevenueCategoryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateRevenueCategoryRepositories(repositories)
	updateServices := UpdateRevenueCategoryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteRevenueCategoryRepositories(repositories)
	deleteServices := DeleteRevenueCategoryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListRevenueCategoriesRepositories(repositories)
	listServices := ListRevenueCategoriesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateRevenueCategory: NewCreateRevenueCategoryUseCase(createRepos, createServices),
		ReadRevenueCategory:   NewReadRevenueCategoryUseCase(readRepos, readServices),
		UpdateRevenueCategory: NewUpdateRevenueCategoryUseCase(updateRepos, updateServices),
		DeleteRevenueCategory: NewDeleteRevenueCategoryUseCase(deleteRepos, deleteServices),
		ListRevenueCategories: NewListRevenueCategoriesUseCase(listRepos, listServices),
	}
}
