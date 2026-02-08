package delegate_client

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
	delegateclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_client"
)

// DelegateClientRepositories groups all repository dependencies for delegate client use cases
type DelegateClientRepositories struct {
	DelegateClient delegateclientpb.DelegateClientDomainServiceServer // Primary entity repository
	Delegate       delegatepb.DelegateDomainServiceServer             // Entity reference validation
	Client         clientpb.ClientDomainServiceServer                 // Entity reference validation
}

// DelegateClientServices groups all business service dependencies for delegate client use cases
type DelegateClientServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all delegate client-related use cases
type UseCases struct {
	CreateDelegateClient *CreateDelegateClientUseCase
	ReadDelegateClient   *ReadDelegateClientUseCase
	UpdateDelegateClient *UpdateDelegateClientUseCase
	DeleteDelegateClient *DeleteDelegateClientUseCase
	ListDelegateClients  *ListDelegateClientsUseCase
}

// NewUseCases creates a new collection of delegate client use cases
func NewUseCases(
	repositories DelegateClientRepositories,
	services DelegateClientServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateDelegateClientRepositories(repositories)
	createServices := CreateDelegateClientServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadDelegateClientRepositories(repositories)
	readServices := ReadDelegateClientServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateDelegateClientRepositories(repositories)
	updateServices := UpdateDelegateClientServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteDelegateClientRepositories(repositories)
	deleteServices := DeleteDelegateClientServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListDelegateClientsRepositories(repositories)
	listServices := ListDelegateClientsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateDelegateClient: NewCreateDelegateClientUseCase(createRepos, createServices),
		ReadDelegateClient:   NewReadDelegateClientUseCase(readRepos, readServices),
		UpdateDelegateClient: NewUpdateDelegateClientUseCase(updateRepos, updateServices),
		DeleteDelegateClient: NewDeleteDelegateClientUseCase(deleteRepos, deleteServices),
		ListDelegateClients:  NewListDelegateClientsUseCase(listRepos, listServices),
	}
}

// NewUseCasesUngrouped creates a new collection of delegate client use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(
	delegateClientRepo delegateclientpb.DelegateClientDomainServiceServer,
	delegateRepo delegatepb.DelegateDomainServiceServer,
	clientRepo clientpb.ClientDomainServiceServer,
	authorizationService ports.AuthorizationService,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := DelegateClientRepositories{
		DelegateClient: delegateClientRepo,
		Delegate:       delegateRepo,
		Client:         clientRepo,
	}

	services := DelegateClientServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
