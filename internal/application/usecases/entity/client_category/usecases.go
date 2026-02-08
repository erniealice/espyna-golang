package client_category

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	clientcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_category"
)

// ClientCategoryRepositories groups all repository dependencies for client_category use cases
type ClientCategoryRepositories struct {
	ClientCategory clientcategorypb.ClientCategoryDomainServiceServer
}

// ClientCategoryServices groups all business service dependencies for client_category use cases
type ClientCategoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all client_category-related use cases
type UseCases struct {
	CreateClientCategory          *CreateClientCategoryUseCase
	ReadClientCategory            *ReadClientCategoryUseCase
	UpdateClientCategory          *UpdateClientCategoryUseCase
	DeleteClientCategory          *DeleteClientCategoryUseCase
	ListClientCategories          *ListClientCategoriesUseCase
	GetClientCategoryListPageData *GetClientCategoryListPageDataUseCase
	GetClientCategoryItemPageData *GetClientCategoryItemPageDataUseCase
}

// NewUseCases creates a new collection of client_category use cases
func NewUseCases(
	repositories ClientCategoryRepositories,
	services ClientCategoryServices,
) *UseCases {
	createRepos := CreateClientCategoryRepositories(repositories)
	createServices := CreateClientCategoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadClientCategoryRepositories(repositories)
	readServices := ReadClientCategoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateClientCategoryRepositories(repositories)
	updateServices := UpdateClientCategoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	deleteRepos := DeleteClientCategoryRepositories(repositories)
	deleteServices := DeleteClientCategoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListClientCategoriesRepositories(repositories)
	listServices := ListClientCategoriesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetClientCategoryListPageDataRepositories{
		ClientCategory: repositories.ClientCategory,
	}
	getListPageDataServices := GetClientCategoryListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getItemPageDataRepos := GetClientCategoryItemPageDataRepositories{
		ClientCategory: repositories.ClientCategory,
	}
	getItemPageDataServices := GetClientCategoryItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	return &UseCases{
		CreateClientCategory:          NewCreateClientCategoryUseCase(createRepos, createServices),
		ReadClientCategory:            NewReadClientCategoryUseCase(readRepos, readServices),
		UpdateClientCategory:          NewUpdateClientCategoryUseCase(updateRepos, updateServices),
		DeleteClientCategory:          NewDeleteClientCategoryUseCase(deleteRepos, deleteServices),
		ListClientCategories:          NewListClientCategoriesUseCase(listRepos, listServices),
		GetClientCategoryListPageData: NewGetClientCategoryListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetClientCategoryItemPageData: NewGetClientCategoryItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of client_category use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(clientCategoryRepo clientcategorypb.ClientCategoryDomainServiceServer) *UseCases {
	repositories := ClientCategoryRepositories{
		ClientCategory: clientCategoryRepo,
	}

	services := ClientCategoryServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewUseCases(repositories, services)
}
