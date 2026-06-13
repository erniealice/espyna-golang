package client_category

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	clientcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_category"
)

// ClientCategoryRepositories groups all repository dependencies for client_category use cases
type ClientCategoryRepositories struct {
	ClientCategory clientcategorypb.ClientCategoryDomainServiceServer
}

// ClientCategoryServices groups all business service dependencies for client_category use cases
type ClientCategoryServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadClientCategoryRepositories(repositories)
	readServices := ReadClientCategoryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateClientCategoryRepositories(repositories)
	updateServices := UpdateClientCategoryServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	deleteRepos := DeleteClientCategoryRepositories(repositories)
	deleteServices := DeleteClientCategoryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListClientCategoriesRepositories(repositories)
	listServices := ListClientCategoriesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getListPageDataRepos := GetClientCategoryListPageDataRepositories{
		ClientCategory: repositories.ClientCategory,
	}
	getListPageDataServices := GetClientCategoryListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getItemPageDataRepos := GetClientCategoryItemPageDataRepositories{
		ClientCategory: repositories.ClientCategory,
	}
	getItemPageDataServices := GetClientCategoryItemPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
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
		Authorizer:  nil,
		Transactor:  ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
		IDGenerator: ports.NewNoOpIDGenerator(),
	}

	return NewUseCases(repositories, services)
}
