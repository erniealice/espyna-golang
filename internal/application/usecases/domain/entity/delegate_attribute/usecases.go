package delegate_attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
	delegateattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_attribute"
)

// UseCases contains all delegate attribute-related use cases
type UseCases struct {
	CreateDelegateAttribute          *CreateDelegateAttributeUseCase
	ReadDelegateAttribute            *ReadDelegateAttributeUseCase
	UpdateDelegateAttribute          *UpdateDelegateAttributeUseCase
	DeleteDelegateAttribute          *DeleteDelegateAttributeUseCase
	ListDelegateAttributes           *ListDelegateAttributesUseCase
	GetDelegateAttributeListPageData *GetDelegateAttributeListPageDataUseCase
	GetDelegateAttributeItemPageData *GetDelegateAttributeItemPageDataUseCase
}

// DelegateAttributeRepositories groups all repository dependencies for delegate attribute use cases
type DelegateAttributeRepositories struct {
	DelegateAttribute delegateattributepb.DelegateAttributeDomainServiceServer // Primary entity repository
	Delegate          delegatepb.DelegateDomainServiceServer                   // Entity reference validation
	Attribute         attributepb.AttributeDomainServiceServer                 // Entity reference validation
}

// DelegateAttributeServices groups all business service dependencies for delegate attribute use cases
type DelegateAttributeServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// NewUseCases creates a new collection of delegate attribute use cases
func NewUseCases(
	repositories DelegateAttributeRepositories,
	services DelegateAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateDelegateAttributeRepositories(repositories)
	createServices := CreateDelegateAttributeServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadDelegateAttributeRepositories{
		DelegateAttribute: repositories.DelegateAttribute,
	}
	readServices := ReadDelegateAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateDelegateAttributeRepositories{
		DelegateAttribute: repositories.DelegateAttribute,
		Delegate:          repositories.Delegate,
		Attribute:         repositories.Attribute,
	}
	updateServices := UpdateDelegateAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteDelegateAttributeRepositories{
		DelegateAttribute: repositories.DelegateAttribute,
	}
	deleteServices := DeleteDelegateAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListDelegateAttributesRepositories{
		DelegateAttribute: repositories.DelegateAttribute,
	}
	listServices := ListDelegateAttributesServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getListPageDataRepos := GetDelegateAttributeListPageDataRepositories{
		DelegateAttribute: repositories.DelegateAttribute,
	}
	getListPageDataServices := GetDelegateAttributeListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getItemPageDataRepos := GetDelegateAttributeItemPageDataRepositories{
		DelegateAttribute: repositories.DelegateAttribute,
	}
	getItemPageDataServices := GetDelegateAttributeItemPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		CreateDelegateAttribute:          NewCreateDelegateAttributeUseCase(createRepos, createServices),
		ReadDelegateAttribute:            NewReadDelegateAttributeUseCase(readRepos, readServices),
		UpdateDelegateAttribute:          NewUpdateDelegateAttributeUseCase(updateRepos, updateServices),
		DeleteDelegateAttribute:          NewDeleteDelegateAttributeUseCase(deleteRepos, deleteServices),
		ListDelegateAttributes:           NewListDelegateAttributesUseCase(listRepos, listServices),
		GetDelegateAttributeListPageData: NewGetDelegateAttributeListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetDelegateAttributeItemPageData: NewGetDelegateAttributeItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of delegate attribute use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(
	delegateAttributeRepo delegateattributepb.DelegateAttributeDomainServiceServer,
	delegateRepo delegatepb.DelegateDomainServiceServer,
	attributeRepo attributepb.AttributeDomainServiceServer,
	authorizationService ports.Authorizer,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := DelegateAttributeRepositories{
		DelegateAttribute: delegateAttributeRepo,
		Delegate:          delegateRepo,
		Attribute:         attributeRepo,
	}

	services := DelegateAttributeServices{
		Authorizer: authorizationService,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewUseCases(repositories, services)
}
