package group_attribute

import (
	"leapfor.xyz/espyna/internal/application/ports"
	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"
	grouppb "leapfor.xyz/esqyma/golang/v1/domain/entity/group"
	groupattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/group_attribute"
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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// NewUseCases creates a new collection of group attribute use cases
func NewUseCases(
	repositories GroupAttributeRepositories,
	services GroupAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateGroupAttributeRepositories(repositories)
	createServices := CreateGroupAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	readRepos := ReadGroupAttributeRepositories{
		GroupAttribute: repositories.GroupAttribute,
	}
	readServices := ReadGroupAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	updateRepos := UpdateGroupAttributeRepositories{
		GroupAttribute: repositories.GroupAttribute,
		Group:          repositories.Group,
		Attribute:      repositories.Attribute,
	}
	updateServices := UpdateGroupAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	deleteRepos := DeleteGroupAttributeRepositories{
		GroupAttribute: repositories.GroupAttribute,
	}
	deleteServices := DeleteGroupAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	listRepos := ListGroupAttributesRepositories{
		GroupAttribute: repositories.GroupAttribute,
	}
	listServices := ListGroupAttributesServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getListPageDataRepos := GetGroupAttributeListPageDataRepositories{
		GroupAttribute: repositories.GroupAttribute,
	}
	getListPageDataServices := GetGroupAttributeListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getItemPageDataRepos := GetGroupAttributeItemPageDataRepositories{
		GroupAttribute: repositories.GroupAttribute,
	}
	getItemPageDataServices := GetGroupAttributeItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
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
	authorizationService ports.AuthorizationService,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := GroupAttributeRepositories{
		GroupAttribute: groupAttributeRepo,
		Group:          groupRepo,
		Attribute:      attributeRepo,
	}

	services := GroupAttributeServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
