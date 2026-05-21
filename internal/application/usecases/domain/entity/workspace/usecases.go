package workspace

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

// WorkspaceRepositories groups all repository dependencies for workspace use cases
type WorkspaceRepositories struct {
	Workspace workspacepb.WorkspaceDomainServiceServer // Primary entity repository
}

// WorkspaceServices groups all business service dependencies for workspace use cases
type WorkspaceServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all workspace-related use cases
type UseCases struct {
	CreateWorkspace          *CreateWorkspaceUseCase
	ReadWorkspace            *ReadWorkspaceUseCase
	UpdateWorkspace          *UpdateWorkspaceUseCase
	DeleteWorkspace          *DeleteWorkspaceUseCase
	ListWorkspaces           *ListWorkspacesUseCase
	GetWorkspaceListPageData *GetWorkspaceListPageDataUseCase
	GetWorkspaceItemPageData *GetWorkspaceItemPageDataUseCase
	SwitchWorkspace          *SwitchWorkspaceUseCase
	ListUserWorkspaces       *ListUserWorkspacesUseCase
}

// NewUseCases creates a new collection of workspace use cases
func NewUseCases(
	repositories WorkspaceRepositories,
	services WorkspaceServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateWorkspaceRepositories(repositories)
	createServices := CreateWorkspaceServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadWorkspaceRepositories(repositories)
	readServices := ReadWorkspaceServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateWorkspaceRepositories(repositories)
	updateServices := UpdateWorkspaceServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteWorkspaceRepositories(repositories)
	deleteServices := DeleteWorkspaceServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListWorkspacesRepositories(repositories)
	listServices := ListWorkspacesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetWorkspaceListPageDataRepositories(repositories)
	getListPageDataServices := GetWorkspaceListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getItemPageDataRepos := GetWorkspaceItemPageDataRepositories(repositories)
	getItemPageDataServices := GetWorkspaceItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	switchRepos := SwitchWorkspaceRepositories(repositories)
	switchServices := SwitchWorkspaceServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listUserRepos := ListUserWorkspacesRepositories(repositories)
	listUserServices := ListUserWorkspacesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateWorkspace:          NewCreateWorkspaceUseCase(createRepos, createServices),
		ReadWorkspace:            NewReadWorkspaceUseCase(readRepos, readServices),
		UpdateWorkspace:          NewUpdateWorkspaceUseCase(updateRepos, updateServices),
		DeleteWorkspace:          NewDeleteWorkspaceUseCase(deleteRepos, deleteServices),
		ListWorkspaces:           NewListWorkspacesUseCase(listRepos, listServices),
		GetWorkspaceListPageData: NewGetWorkspaceListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetWorkspaceItemPageData: NewGetWorkspaceItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
		SwitchWorkspace:          NewSwitchWorkspaceUseCase(switchRepos, switchServices),
		ListUserWorkspaces:       NewListUserWorkspacesUseCase(listUserRepos, listUserServices),
	}
}

// NewUseCasesUngrouped creates a new collection of workspace use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(workspaceRepo workspacepb.WorkspaceDomainServiceServer) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := WorkspaceRepositories{
		Workspace: workspaceRepo,
	}

	services := WorkspaceServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewUseCases(repositories, services)
}
