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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
}

// NewUseCases creates a new collection of workspace use cases
func NewUseCases(
	repositories WorkspaceRepositories,
	services WorkspaceServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateWorkspaceRepositories(repositories)
	createServices := CreateWorkspaceServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadWorkspaceRepositories(repositories)
	readServices := ReadWorkspaceServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateWorkspaceRepositories(repositories)
	updateServices := UpdateWorkspaceServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteWorkspaceRepositories(repositories)
	deleteServices := DeleteWorkspaceServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListWorkspacesRepositories(repositories)
	listServices := ListWorkspacesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetWorkspaceListPageDataRepositories(repositories)
	getListPageDataServices := GetWorkspaceListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getItemPageDataRepos := GetWorkspaceItemPageDataRepositories(repositories)
	getItemPageDataServices := GetWorkspaceItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateWorkspace:          NewCreateWorkspaceUseCase(createRepos, createServices),
		ReadWorkspace:            NewReadWorkspaceUseCase(readRepos, readServices),
		UpdateWorkspace:          NewUpdateWorkspaceUseCase(updateRepos, updateServices),
		DeleteWorkspace:          NewDeleteWorkspaceUseCase(deleteRepos, deleteServices),
		ListWorkspaces:           NewListWorkspacesUseCase(listRepos, listServices),
		GetWorkspaceListPageData: NewGetWorkspaceListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetWorkspaceItemPageData: NewGetWorkspaceItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
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
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
