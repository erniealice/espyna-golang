package client

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// ClientRepositories groups all repository dependencies for client use cases
type ClientRepositories struct {
	Client clientpb.ClientDomainServiceServer // Primary entity repository
	User   userpb.UserDomainServiceServer     // User repository for embedded user data
}

// ClientServices groups all business service dependencies for client use cases
type ClientServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all client-related use cases
type UseCases struct {
	CreateClient          *CreateClientUseCase
	ReadClient            *ReadClientUseCase
	UpdateClient          *UpdateClientUseCase
	DeleteClient          *DeleteClientUseCase
	ListClients           *ListClientsUseCase
	GetClientListPageData *GetClientListPageDataUseCase
	GetClientItemPageData *GetClientItemPageDataUseCase
	FindOrCreateClient    *FindOrCreateClientUseCase
	GetClientByEmail      *GetClientByEmailUseCase
	SearchClientsByName   *SearchClientsByNameUseCase
}

// NewUseCases creates a new collection of client use cases
func NewUseCases(
	repositories ClientRepositories,
	services ClientServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	// Note: Using explicit struct initialization instead of type conversion
	// because CreateClientRepositories has additional fields (User)
	createRepos := CreateClientRepositories{
		Client: repositories.Client,
		User:   repositories.User,
	}
	createServices := CreateClientServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadClientRepositories{
		Client: repositories.Client,
	}
	readServices := ReadClientServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateClientRepositories{
		Client: repositories.Client,
	}
	updateServices := UpdateClientServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteClientRepositories{
		Client: repositories.Client,
	}
	deleteServices := DeleteClientServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListClientsRepositories{
		Client: repositories.Client,
	}
	listServices := ListClientsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getListPageDataRepos := GetClientListPageDataRepositories{
		Client: repositories.Client,
	}
	getListPageDataServices := GetClientListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getItemPageDataRepos := GetClientItemPageDataRepositories{
		Client: repositories.Client,
	}
	getItemPageDataServices := GetClientItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	findOrCreateRepos := FindOrCreateClientRepositories{
		Client: repositories.Client,
		User:   repositories.User,
	}
	findOrCreateServices := FindOrCreateClientServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	getByEmailRepos := GetClientByEmailRepositories{
		Client: repositories.Client,
		User:   repositories.User,
	}
	getByEmailServices := GetClientByEmailServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	searchByNameRepos := SearchClientsByNameRepositories{
		Client: repositories.Client,
	}
	searchByNameServices := SearchClientsByNameServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		CreateClient:          NewCreateClientUseCase(createRepos, createServices),
		ReadClient:            NewReadClientUseCase(readRepos, readServices),
		UpdateClient:          NewUpdateClientUseCase(updateRepos, updateServices),
		DeleteClient:          NewDeleteClientUseCase(deleteRepos, deleteServices),
		ListClients:           NewListClientsUseCase(listRepos, listServices),
		GetClientListPageData: NewGetClientListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetClientItemPageData: NewGetClientItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
		FindOrCreateClient:    NewFindOrCreateClientUseCase(findOrCreateRepos, findOrCreateServices),
		GetClientByEmail:      NewGetClientByEmailUseCase(getByEmailRepos, getByEmailServices),
		SearchClientsByName:   NewSearchClientsByNameUseCase(searchByNameRepos, searchByNameServices),
	}
}

// NewUseCasesUngrouped creates a new collection of client use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(clientRepo clientpb.ClientDomainServiceServer) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := ClientRepositories{
		Client: clientRepo,
	}

	services := ClientServices{
		Authorizer:  nil, // Will be injected later by container
		Transactor:  ports.NewNoOpTransactor(),
		Translator:  ports.NewNoOpTranslator(),
		IDGenerator: ports.NewNoOpIDGenerator(),
	}

	return NewUseCases(repositories, services)
}
