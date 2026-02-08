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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	readRepos := ReadGroupRepositories(repositories)
	readServices := ReadGroupServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateGroupRepositories(repositories)
	updateServices := UpdateGroupServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteGroupRepositories(repositories)
	deleteServices := DeleteGroupServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListGroupsRepositories(repositories)
	listServices := ListGroupsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetGroupListPageDataRepositories{
		Group: repositories.Group,
	}
	getListPageDataServices := GetGroupListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getItemPageDataRepos := GetGroupItemPageDataRepositories{
		Group: repositories.Group,
	}
	getItemPageDataServices := GetGroupItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
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
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
