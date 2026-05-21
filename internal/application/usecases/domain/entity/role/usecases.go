package role

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
)

// RoleRepositories groups all repository dependencies for role use cases
type RoleRepositories struct {
	Role rolepb.RoleDomainServiceServer // Primary entity repository
}

// RoleServices groups all business service dependencies for role use cases
type RoleServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all role-related use cases
type UseCases struct {
	CreateRole          *CreateRoleUseCase
	ReadRole            *ReadRoleUseCase
	UpdateRole          *UpdateRoleUseCase
	DeleteRole          *DeleteRoleUseCase
	ListRoles           *ListRolesUseCase
	GetRoleListPageData *GetRoleListPageDataUseCase
	GetRoleItemPageData *GetRoleItemPageDataUseCase
}

// NewUseCases creates a new collection of role use cases
func NewUseCases(
	repositories RoleRepositories,
	services RoleServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateRoleRepositories(repositories)
	createServices := CreateRoleServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadRoleRepositories(repositories)
	readServices := ReadRoleServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateRoleRepositories(repositories)
	updateServices := UpdateRoleServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteRoleRepositories(repositories)
	deleteServices := DeleteRoleServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListRolesRepositories(repositories)
	listServices := ListRolesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetRoleListPageDataRepositories(repositories)
	getListPageDataServices := GetRoleListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getItemPageDataRepos := GetRoleItemPageDataRepositories(repositories)
	getItemPageDataServices := GetRoleItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateRole:          NewCreateRoleUseCase(createRepos, createServices),
		ReadRole:            NewReadRoleUseCase(readRepos, readServices),
		UpdateRole:          NewUpdateRoleUseCase(updateRepos, updateServices),
		DeleteRole:          NewDeleteRoleUseCase(deleteRepos, deleteServices),
		ListRoles:           NewListRolesUseCase(listRepos, listServices),
		GetRoleListPageData: NewGetRoleListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetRoleItemPageData: NewGetRoleItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of role use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(roleRepo rolepb.RoleDomainServiceServer) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := RoleRepositories{
		Role: roleRepo,
	}

	services := RoleServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewUseCases(repositories, services)
}
