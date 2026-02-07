package client

import (
	"leapfor.xyz/espyna/internal/application/ports"
	clientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/client"
	userpb "leapfor.xyz/esqyma/golang/v1/domain/entity/user"
)

// ClientRepositories groups all repository dependencies for client use cases
type ClientRepositories struct {
	Client clientpb.ClientDomainServiceServer // Primary entity repository
	User   userpb.UserDomainServiceServer     // User repository for embedded user data
}

// ClientServices groups all business service dependencies for client use cases
type ClientServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadClientRepositories{
		Client: repositories.Client,
	}
	readServices := ReadClientServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateClientRepositories{
		Client: repositories.Client,
	}
	updateServices := UpdateClientServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteClientRepositories{
		Client: repositories.Client,
	}
	deleteServices := DeleteClientServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListClientsRepositories{
		Client: repositories.Client,
	}
	listServices := ListClientsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetClientListPageDataRepositories{
		Client: repositories.Client,
	}
	getListPageDataServices := GetClientListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getItemPageDataRepos := GetClientItemPageDataRepositories{
		Client: repositories.Client,
	}
	getItemPageDataServices := GetClientItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	findOrCreateRepos := FindOrCreateClientRepositories{
		Client: repositories.Client,
		User:   repositories.User,
	}
	findOrCreateServices := FindOrCreateClientServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	getByEmailRepos := GetClientByEmailRepositories{
		Client: repositories.Client,
		User:   repositories.User,
	}
	getByEmailServices := GetClientByEmailServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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
		AuthorizationService: nil, // Will be injected later by container
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewUseCases(repositories, services)
}
