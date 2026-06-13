package group_attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	grouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/group"
	groupattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/group_attribute"
)

// UseCases contains all group attribute-related use cases
type UseCases struct {
	CreateGroupAttribute          *CreateGroupAttributeUseCase
	ReadGroupAttribute            *ReadGroupAttributeUseCase
	UpdateGroupAttribute          *UpdateGroupAttributeUseCase
	DeleteGroupAttribute          *DeleteGroupAttributeUseCase
	ListGroupAttributes           *ListGroupAttributesUseCase
	GetGroupAttributeListPageData *GetGroupAttributeListPageDataUseCase
	GetGroupAttributeItemPageData *GetGroupAttributeItemPageDataUseCase
}

// GroupAttributeRepositories groups all repository dependencies for group attribute use cases
type GroupAttributeRepositories struct {
	GroupAttribute groupattributepb.GroupAttributeDomainServiceServer // Primary entity repository
	Group          grouppb.GroupDomainServiceServer                   // Entity reference validation
	Attribute      attributepb.AttributeDomainServiceServer           // Entity reference validation
}

// GroupAttributeServices groups all business service dependencies for group attribute use cases
type GroupAttributeServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// NewUseCases creates a new collection of group attribute use cases
func NewUseCases(
	repositories GroupAttributeRepositories,
	services GroupAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateGroupAttributeRepositories(repositories)
	createServices := CreateGroupAttributeServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadGroupAttributeRepositories{
		GroupAttribute: repositories.GroupAttribute,
	}
	readServices := ReadGroupAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateGroupAttributeRepositories{
		GroupAttribute: repositories.GroupAttribute,
		Group:          repositories.Group,
		Attribute:      repositories.Attribute,
	}
	updateServices := UpdateGroupAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteGroupAttributeRepositories{
		GroupAttribute: repositories.GroupAttribute,
	}
	deleteServices := DeleteGroupAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListGroupAttributesRepositories{
		GroupAttribute: repositories.GroupAttribute,
	}
	listServices := ListGroupAttributesServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getListPageDataRepos := GetGroupAttributeListPageDataRepositories{
		GroupAttribute: repositories.GroupAttribute,
	}
	getListPageDataServices := GetGroupAttributeListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getItemPageDataRepos := GetGroupAttributeItemPageDataRepositories{
		GroupAttribute: repositories.GroupAttribute,
	}
	getItemPageDataServices := GetGroupAttributeItemPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		CreateGroupAttribute:          NewCreateGroupAttributeUseCase(createRepos, createServices),
		ReadGroupAttribute:            NewReadGroupAttributeUseCase(readRepos, readServices),
		UpdateGroupAttribute:          NewUpdateGroupAttributeUseCase(updateRepos, updateServices),
		DeleteGroupAttribute:          NewDeleteGroupAttributeUseCase(deleteRepos, deleteServices),
		ListGroupAttributes:           NewListGroupAttributesUseCase(listRepos, listServices),
		GetGroupAttributeListPageData: NewGetGroupAttributeListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetGroupAttributeItemPageData: NewGetGroupAttributeItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of group attribute use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(
	groupAttributeRepo groupattributepb.GroupAttributeDomainServiceServer,
	groupRepo grouppb.GroupDomainServiceServer,
	attributeRepo attributepb.AttributeDomainServiceServer,
	authorizationService ports.Authorizer,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := GroupAttributeRepositories{
		GroupAttribute: groupAttributeRepo,
		Group:          groupRepo,
		Attribute:      attributeRepo,
	}

	services := GroupAttributeServices{
		Authorizer: authorizationService,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewUseCases(repositories, services)
}
