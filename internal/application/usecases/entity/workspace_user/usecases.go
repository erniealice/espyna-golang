package workspace_user

import (
	"leapfor.xyz/espyna/internal/application/ports"
	userpb "leapfor.xyz/esqyma/golang/v1/domain/entity/user"
	workspacepb "leapfor.xyz/esqyma/golang/v1/domain/entity/workspace"
	workspaceuserpb "leapfor.xyz/esqyma/golang/v1/domain/entity/workspace_user"
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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// NewUseCases creates a new collection of workspace user use cases
func NewUseCases(
	repositories WorkspaceUserRepositories,
	services WorkspaceUserServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateWorkspaceUserRepositories(repositories)
	createServices := CreateWorkspaceUserServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadWorkspaceUserRepositories(repositories)
	readServices := ReadWorkspaceUserServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateWorkspaceUserRepositories(repositories)
	updateServices := UpdateWorkspaceUserServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteWorkspaceUserRepositories(repositories)
	deleteServices := DeleteWorkspaceUserServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListWorkspaceUsersRepositories(repositories)
	listServices := ListWorkspaceUsersServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetWorkspaceUserListPageDataRepositories{
		WorkspaceUser: repositories.WorkspaceUser,
	}
	listPageDataServices := GetWorkspaceUserListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	itemPageDataRepos := GetWorkspaceUserItemPageDataRepositories{
		WorkspaceUser: repositories.WorkspaceUser,
	}
	itemPageDataServices := GetWorkspaceUserItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
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
	authorizationService ports.AuthorizationService,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := WorkspaceUserRepositories{
		WorkspaceUser: workspaceUserRepo,
		Workspace:     workspaceRepo,
		User:          userRepo,
	}

	services := WorkspaceUserServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
