package workspace

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

// WorkspaceRepositories groups all repository dependencies for workspace use cases
type WorkspaceRepositories struct {
	Workspace workspacepb.WorkspaceDomainServiceServer // Primary entity repository
}

// WorkspaceServices groups all business service dependencies for workspace use cases
type WorkspaceServices struct {
	Authorizer    ports.Authorizer
	Transactor    ports.Transactor
	Translator    ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator   ports.IDGenerator
	ReservedSlugs ReservedSlugProvider // optional; nil disables the reserved-word slug check
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
	ValidateSlug             *ValidateSlugUseCase
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
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadWorkspaceRepositories(repositories)
	readServices := ReadWorkspaceServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateWorkspaceRepositories(repositories)
	updateServices := UpdateWorkspaceServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteWorkspaceRepositories(repositories)
	deleteServices := DeleteWorkspaceServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListWorkspacesRepositories(repositories)
	listServices := ListWorkspacesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getListPageDataRepos := GetWorkspaceListPageDataRepositories(repositories)
	getListPageDataServices := GetWorkspaceListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getItemPageDataRepos := GetWorkspaceItemPageDataRepositories(repositories)
	getItemPageDataServices := GetWorkspaceItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	switchRepos := SwitchWorkspaceRepositories(repositories)
	switchServices := SwitchWorkspaceServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listUserRepos := ListUserWorkspacesRepositories(repositories)
	listUserServices := ListUserWorkspacesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	validateSlugServices := ValidateSlugServices{
		Translator:    services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		ReservedSlugs: services.ReservedSlugs,
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
		ValidateSlug:             NewValidateSlugUseCase(validateSlugServices),
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
