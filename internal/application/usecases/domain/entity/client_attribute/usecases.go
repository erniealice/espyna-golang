package client_attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
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
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// NewUseCases creates a new collection of client attribute use cases
func NewUseCases(
	repositories ClientAttributeRepositories,
	services ClientAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateClientAttributeRepositories(repositories)
	createServices := CreateClientAttributeServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadClientAttributeRepositories{
		ClientAttribute: repositories.ClientAttribute,
	}
	readServices := ReadClientAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateClientAttributeRepositories{
		ClientAttribute: repositories.ClientAttribute,
		Client:          repositories.Client,
		Attribute:       repositories.Attribute,
	}
	updateServices := UpdateClientAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteClientAttributeRepositories{
		ClientAttribute: repositories.ClientAttribute,
	}
	deleteServices := DeleteClientAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListClientAttributesRepositories{
		ClientAttribute: repositories.ClientAttribute,
	}
	listServices := ListClientAttributesServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getListPageDataRepos := GetClientAttributeListPageDataRepositories{
		ClientAttribute: repositories.ClientAttribute,
	}
	getListPageDataServices := GetClientAttributeListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getItemPageDataRepos := GetClientAttributeItemPageDataRepositories{
		ClientAttribute: repositories.ClientAttribute,
	}
	getItemPageDataServices := GetClientAttributeItemPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
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
	authorizationService ports.Authorizer,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := ClientAttributeRepositories{
		ClientAttribute: clientAttributeRepo,
		Client:          clientRepo,
		Attribute:       attributeRepo,
	}

	services := ClientAttributeServices{
		Authorizer: authorizationService,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewUseCases(repositories, services)
}
