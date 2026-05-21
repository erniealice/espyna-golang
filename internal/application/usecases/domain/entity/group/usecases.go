package group

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	grouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/group"
)

// GroupRepositories groups all repository dependencies for group use cases
type GroupRepositories struct {
	Group grouppb.GroupDomainServiceServer // Primary entity repository
}

// GroupServices groups all business service dependencies for group use cases
type GroupServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all group-related use cases
type UseCases struct {
	CreateGroup          *CreateGroupUseCase
	ReadGroup            *ReadGroupUseCase
	UpdateGroup          *UpdateGroupUseCase
	DeleteGroup          *DeleteGroupUseCase
	ListGroups           *ListGroupsUseCase
	GetGroupListPageData *GetGroupListPageDataUseCase
	GetGroupItemPageData *GetGroupItemPageDataUseCase
}

// NewUseCases creates a new collection of group use cases
func NewUseCases(
	repositories GroupRepositories,
	services GroupServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateGroupRepositories(repositories)
	createServices := CreateGroupServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadGroupRepositories(repositories)
	readServices := ReadGroupServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateGroupRepositories(repositories)
	updateServices := UpdateGroupServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteGroupRepositories(repositories)
	deleteServices := DeleteGroupServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListGroupsRepositories(repositories)
	listServices := ListGroupsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetGroupListPageDataRepositories{
		Group: repositories.Group,
	}
	getListPageDataServices := GetGroupListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getItemPageDataRepos := GetGroupItemPageDataRepositories{
		Group: repositories.Group,
	}
	getItemPageDataServices := GetGroupItemPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateGroup:          NewCreateGroupUseCase(createRepos, createServices),
		ReadGroup:            NewReadGroupUseCase(readRepos, readServices),
		UpdateGroup:          NewUpdateGroupUseCase(updateRepos, updateServices),
		DeleteGroup:          NewDeleteGroupUseCase(deleteRepos, deleteServices),
		ListGroups:           NewListGroupsUseCase(listRepos, listServices),
		GetGroupListPageData: NewGetGroupListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetGroupItemPageData: NewGetGroupItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of group use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(groupRepo grouppb.GroupDomainServiceServer) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := GroupRepositories{
		Group: groupRepo,
	}

	services := GroupServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewUseCases(repositories, services)
}
