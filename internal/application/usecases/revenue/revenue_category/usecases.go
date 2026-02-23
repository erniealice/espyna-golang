package revenuecategory

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_category"
)

// RevenueCategoryRepositories groups all repository dependencies
type RevenueCategoryRepositories struct {
	RevenueCategory pb.RevenueCategoryDomainServiceServer
}

// RevenueCategoryServices groups all business service dependencies
type RevenueCategoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadRevenueCategoryRepositories(repositories)
	readServices := ReadRevenueCategoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateRevenueCategoryRepositories(repositories)
	updateServices := UpdateRevenueCategoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteRevenueCategoryRepositories(repositories)
	deleteServices := DeleteRevenueCategoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListRevenueCategoriesRepositories(repositories)
	listServices := ListRevenueCategoriesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateRevenueCategory: NewCreateRevenueCategoryUseCase(createRepos, createServices),
		ReadRevenueCategory:   NewReadRevenueCategoryUseCase(readRepos, readServices),
		UpdateRevenueCategory: NewUpdateRevenueCategoryUseCase(updateRepos, updateServices),
		DeleteRevenueCategory: NewDeleteRevenueCategoryUseCase(deleteRepos, deleteServices),
		ListRevenueCategories: NewListRevenueCategoriesUseCase(listRepos, listServices),
	}
}
