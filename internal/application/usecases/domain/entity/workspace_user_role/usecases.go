package workspace_user_role

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

// WorkspaceUserRoleRepositories groups all repository dependencies for workspace user role use cases
type WorkspaceUserRoleRepositories struct {
	WorkspaceUserRole workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer // Primary entity repository
	WorkspaceUser     workspaceuserpb.WorkspaceUserDomainServiceServer         // Entity reference validation
	Role              rolepb.RoleDomainServiceServer                           // Entity reference validation
}

// WorkspaceUserRoleServices groups all business service dependencies for workspace user role use cases
type WorkspaceUserRoleServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all workspace user role-related use cases
type UseCases struct {
	CreateWorkspaceUserRole          *CreateWorkspaceUserRoleUseCase
	ReadWorkspaceUserRole            *ReadWorkspaceUserRoleUseCase
	UpdateWorkspaceUserRole          *UpdateWorkspaceUserRoleUseCase
	DeleteWorkspaceUserRole          *DeleteWorkspaceUserRoleUseCase
	ListWorkspaceUserRoles           *ListWorkspaceUserRolesUseCase
	GetWorkspaceUserRoleListPageData *GetWorkspaceUserRoleListPageDataUseCase
	GetWorkspaceUserRoleItemPageData *GetWorkspaceUserRoleItemPageDataUseCase
}

// NewUseCases creates a new collection of workspace user role use cases
func NewUseCases(
	repositories WorkspaceUserRoleRepositories,
	services WorkspaceUserRoleServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateWorkspaceUserRoleRepositories(repositories)
	createServices := CreateWorkspaceUserRoleServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadWorkspaceUserRoleRepositories(repositories)
	readServices := ReadWorkspaceUserRoleServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateWorkspaceUserRoleRepositories(repositories)
	updateServices := UpdateWorkspaceUserRoleServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteWorkspaceUserRoleRepositories(repositories)
	deleteServices := DeleteWorkspaceUserRoleServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListWorkspaceUserRolesRepositories(repositories)
	listServices := ListWorkspaceUserRolesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listPageDataRepos := GetWorkspaceUserRoleListPageDataRepositories{
		WorkspaceUserRole: repositories.WorkspaceUserRole,
	}
	listPageDataServices := GetWorkspaceUserRoleListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	itemPageDataRepos := GetWorkspaceUserRoleItemPageDataRepositories{
		WorkspaceUserRole: repositories.WorkspaceUserRole,
	}
	itemPageDataServices := GetWorkspaceUserRoleItemPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		CreateWorkspaceUserRole:          NewCreateWorkspaceUserRoleUseCase(createRepos, createServices),
		ReadWorkspaceUserRole:            NewReadWorkspaceUserRoleUseCase(readRepos, readServices),
		UpdateWorkspaceUserRole:          NewUpdateWorkspaceUserRoleUseCase(updateRepos, updateServices),
		DeleteWorkspaceUserRole:          NewDeleteWorkspaceUserRoleUseCase(deleteRepos, deleteServices),
		ListWorkspaceUserRoles:           NewListWorkspaceUserRolesUseCase(listRepos, listServices),
		GetWorkspaceUserRoleListPageData: NewGetWorkspaceUserRoleListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetWorkspaceUserRoleItemPageData: NewGetWorkspaceUserRoleItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}
