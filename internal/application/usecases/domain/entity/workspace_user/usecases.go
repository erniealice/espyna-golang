package workspace_user

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
)

// UseCases contains all workspace user-related use cases
type UseCases struct {
	CreateWorkspaceUser          *CreateWorkspaceUserUseCase
	ReadWorkspaceUser            *ReadWorkspaceUserUseCase
	UpdateWorkspaceUser          *UpdateWorkspaceUserUseCase
	DeleteWorkspaceUser          *DeleteWorkspaceUserUseCase
	ListWorkspaceUsers           *ListWorkspaceUsersUseCase
	GetWorkspaceUserListPageData *GetWorkspaceUserListPageDataUseCase
	GetWorkspaceUserItemPageData *GetWorkspaceUserItemPageDataUseCase
}

// WorkspaceUserRepositories groups all repository dependencies for workspace user use cases
type WorkspaceUserRepositories struct {
	WorkspaceUser workspaceuserpb.WorkspaceUserDomainServiceServer // Primary entity repository
	Workspace     workspacepb.WorkspaceDomainServiceServer         // Entity reference validation
	User          userpb.UserDomainServiceServer                   // Entity reference validation
}

// WorkspaceUserServices groups all business service dependencies for workspace user use cases
type WorkspaceUserServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// NewUseCases creates a new collection of workspace user use cases
func NewUseCases(
	repositories WorkspaceUserRepositories,
	services WorkspaceUserServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateWorkspaceUserRepositories(repositories)
	createServices := CreateWorkspaceUserServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadWorkspaceUserRepositories(repositories)
	readServices := ReadWorkspaceUserServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateWorkspaceUserRepositories(repositories)
	updateServices := UpdateWorkspaceUserServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteWorkspaceUserRepositories(repositories)
	deleteServices := DeleteWorkspaceUserServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListWorkspaceUsersRepositories(repositories)
	listServices := ListWorkspaceUsersServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetWorkspaceUserListPageDataRepositories{
		WorkspaceUser: repositories.WorkspaceUser,
	}
	listPageDataServices := GetWorkspaceUserListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetWorkspaceUserItemPageDataRepositories{
		WorkspaceUser: repositories.WorkspaceUser,
	}
	itemPageDataServices := GetWorkspaceUserItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateWorkspaceUser:          NewCreateWorkspaceUserUseCase(createRepos, createServices),
		ReadWorkspaceUser:            NewReadWorkspaceUserUseCase(readRepos, readServices),
		UpdateWorkspaceUser:          NewUpdateWorkspaceUserUseCase(updateRepos, updateServices),
		DeleteWorkspaceUser:          NewDeleteWorkspaceUserUseCase(deleteRepos, deleteServices),
		ListWorkspaceUsers:           NewListWorkspaceUsersUseCase(listRepos, listServices),
		GetWorkspaceUserListPageData: NewGetWorkspaceUserListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetWorkspaceUserItemPageData: NewGetWorkspaceUserItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of workspace user use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(
	workspaceUserRepo workspaceuserpb.WorkspaceUserDomainServiceServer,
	workspaceRepo workspacepb.WorkspaceDomainServiceServer,
	userRepo userpb.UserDomainServiceServer,
	authorizationService ports.Authorizer,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := WorkspaceUserRepositories{
		WorkspaceUser: workspaceUserRepo,
		Workspace:     workspaceRepo,
		User:          userRepo,
	}

	services := WorkspaceUserServices{
		Authorizer: authorizationService,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewUseCases(repositories, services)
}
