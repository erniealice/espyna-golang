package role_permission

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
	rolepermissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role_permission"
)

// RolePermissionRepositories groups all repository dependencies for role permission use cases
type RolePermissionRepositories struct {
	RolePermission rolepermissionpb.RolePermissionDomainServiceServer // Primary entity repository
	Role           rolepb.RoleDomainServiceServer                     // Entity reference validation
	Permission     permissionpb.PermissionDomainServiceServer         // Entity reference validation
}

// RolePermissionServices groups all business service dependencies for role permission use cases
type RolePermissionServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all role permission-related use cases
type UseCases struct {
	CreateRolePermission *CreateRolePermissionUseCase
	ReadRolePermission   *ReadRolePermissionUseCase
	UpdateRolePermission *UpdateRolePermissionUseCase
	DeleteRolePermission *DeleteRolePermissionUseCase
	ListRolePermissions  *ListRolePermissionsUseCase
}

// NewUseCases creates a new collection of role permission use cases
func NewUseCases(
	repositories RolePermissionRepositories,
	services RolePermissionServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateRolePermissionRepositories(repositories)
	createServices := CreateRolePermissionServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadRolePermissionRepositories(repositories)
	readServices := ReadRolePermissionServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateRolePermissionRepositories(repositories)
	updateServices := UpdateRolePermissionServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteRolePermissionRepositories(repositories)
	deleteServices := DeleteRolePermissionServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListRolePermissionsRepositories(repositories)
	listServices := ListRolePermissionsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		CreateRolePermission: NewCreateRolePermissionUseCase(createRepos, createServices),
		ReadRolePermission:   NewReadRolePermissionUseCase(readRepos, readServices),
		UpdateRolePermission: NewUpdateRolePermissionUseCase(updateRepos, updateServices),
		DeleteRolePermission: NewDeleteRolePermissionUseCase(deleteRepos, deleteServices),
		ListRolePermissions:  NewListRolePermissionsUseCase(listRepos, listServices),
	}
}

// NewUseCasesUngrouped creates a new collection of role permission use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(
	rolePermissionRepo rolepermissionpb.RolePermissionDomainServiceServer,
	roleRepo rolepb.RoleDomainServiceServer,
	permissionRepo permissionpb.PermissionDomainServiceServer,
	authorizationService ports.Authorizer,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := RolePermissionRepositories{
		RolePermission: rolePermissionRepo,
		Role:           roleRepo,
		Permission:     permissionRepo,
	}

	services := RolePermissionServices{
		Authorizer: authorizationService,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewUseCases(repositories, services)
}
