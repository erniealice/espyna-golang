package client_attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	clientattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_attribute"
)

// UseCases contains all client attribute-related use cases
type UseCases struct {
	CreateClientAttribute          *CreateClientAttributeUseCase
	ReadClientAttribute            *ReadClientAttributeUseCase
	UpdateClientAttribute          *UpdateClientAttributeUseCase
	DeleteClientAttribute          *DeleteClientAttributeUseCase
	ListClientAttributes           *ListClientAttributesUseCase
	GetClientAttributeListPageData *GetClientAttributeListPageDataUseCase
	GetClientAttributeItemPageData *GetClientAttributeItemPageDataUseCase
}

// ClientAttributeRepositories groups all repository dependencies for client attribute use cases
type ClientAttributeRepositories struct {
	ClientAttribute clientattributepb.ClientAttributeDomainServiceServer // Primary entity repository
	Client          clientpb.ClientDomainServiceServer                   // Entity reference validation
	Attribute       attributepb.AttributeDomainServiceServer             // Entity reference validation
}

// ClientAttributeServices groups all business service dependencies for client attribute use cases
type ClientAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// NewUseCases creates a new collection of client attribute use cases
func NewUseCases(
	repositories ClientAttributeRepositories,
	services ClientAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateClientAttributeRepositories(repositories)
	createServices := CreateClientAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadClientAttributeRepositories{
		ClientAttribute: repositories.ClientAttribute,
	}
	readServices := ReadClientAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	updateRepos := UpdateClientAttributeRepositories{
		ClientAttribute: repositories.ClientAttribute,
		Client:          repositories.Client,
		Attribute:       repositories.Attribute,
	}
	updateServices := UpdateClientAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	deleteRepos := DeleteClientAttributeRepositories{
		ClientAttribute: repositories.ClientAttribute,
	}
	deleteServices := DeleteClientAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	listRepos := ListClientAttributesRepositories{
		ClientAttribute: repositories.ClientAttribute,
	}
	listServices := ListClientAttributesServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getListPageDataRepos := GetClientAttributeListPageDataRepositories{
		ClientAttribute: repositories.ClientAttribute,
	}
	getListPageDataServices := GetClientAttributeListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getItemPageDataRepos := GetClientAttributeItemPageDataRepositories{
		ClientAttribute: repositories.ClientAttribute,
	}
	getItemPageDataServices := GetClientAttributeItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	return &UseCases{
		CreateClientAttribute:          NewCreateClientAttributeUseCase(createRepos, createServices),
		ReadClientAttribute:            NewReadClientAttributeUseCase(readRepos, readServices),
		UpdateClientAttribute:          NewUpdateClientAttributeUseCase(updateRepos, updateServices),
		DeleteClientAttribute:          NewDeleteClientAttributeUseCase(deleteRepos, deleteServices),
		ListClientAttributes:           NewListClientAttributesUseCase(listRepos, listServices),
		GetClientAttributeListPageData: NewGetClientAttributeListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetClientAttributeItemPageData: NewGetClientAttributeItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of client attribute use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(
	clientAttributeRepo clientattributepb.ClientAttributeDomainServiceServer,
	clientRepo clientpb.ClientDomainServiceServer,
	attributeRepo attributepb.AttributeDomainServiceServer,
	authorizationService ports.AuthorizationService,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := ClientAttributeRepositories{
		ClientAttribute: clientAttributeRepo,
		Client:          clientRepo,
		Attribute:       attributeRepo,
	}

	services := ClientAttributeServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
