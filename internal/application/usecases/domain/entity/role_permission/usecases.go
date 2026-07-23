package role_permission

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	securityports "github.com/erniealice/espyna-golang/internal/application/ports/security"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	clientportalgrantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_portal_grant"
	delegateclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_client"
	delegatesupplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_supplier"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
	rolepermissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role_permission"
	supplierportalgrantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_portal_grant"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

// RolePermissionRepositories groups all repository dependencies for role permission use cases
type RolePermissionRepositories struct {
	RolePermission rolepermissionpb.RolePermissionDomainServiceServer // Primary entity repository
	Role           rolepb.RoleDomainServiceServer                     // Entity reference validation
	Permission     permissionpb.PermissionDomainServiceServer         // Entity reference validation
	// WorkspaceUserRole enumerates the bindings assigned a role so a
	// grant/revoke can evict exactly their cached permission codes (P10/D2).
	// Consumed only by the create/delete use cases. Nil-safe.
	WorkspaceUserRole workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer
	// The remaining grant-table seams extend that invalidation to EVERY binding
	// kind the loader caches — operator/staff via WorkspaceUser, CLIENT/SUPPLIER
	// portal + delegate via their grant tables (CF-1). Consumed only by
	// create/delete. All nil-safe.
	WorkspaceUser       workspaceuserpb.WorkspaceUserDomainServiceServer
	ClientPortalGrant   clientportalgrantpb.ClientPortalGrantDomainServiceServer
	SupplierPortalGrant supplierportalgrantpb.SupplierPortalGrantDomainServiceServer
	DelegateClient      delegateclientpb.DelegateClientDomainServiceServer
	DelegateSupplier    delegatesupplierpb.DelegateSupplierDomainServiceServer
}

// RolePermissionServices groups all business service dependencies for role permission use cases
type RolePermissionServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator      ports.IDGenerator
	// PermissionCacheInvalidator evicts affected bindings' cached RBAC codes on
	// grant/revoke (P10/D2); consumed only by create/delete. Nil-safe.
	PermissionCacheInvalidator securityports.PermissionCacheInvalidator
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
	// Build individual grouped parameters for each use case. Create/Delete take
	// the WorkspaceUserRole repo + invalidator (P10/D2); Read/Update/List keep
	// their narrower 3-repo shape (explicit literals — no struct conversion,
	// since the module struct now carries an extra field they do not need).
	createRepos := CreateRolePermissionRepositories{
		RolePermission:      repositories.RolePermission,
		Role:                repositories.Role,
		Permission:          repositories.Permission,
		WorkspaceUserRole:   repositories.WorkspaceUserRole,
		WorkspaceUser:       repositories.WorkspaceUser,
		ClientPortalGrant:   repositories.ClientPortalGrant,
		SupplierPortalGrant: repositories.SupplierPortalGrant,
		DelegateClient:      repositories.DelegateClient,
		DelegateSupplier:    repositories.DelegateSupplier,
	}
	createServices := CreateRolePermissionServices{
		Authorizer:                 services.Authorizer,
		Transactor:                 services.Transactor,
		Translator:                 services.Translator,
		ActionGatekeeper:           services.ActionGatekeeper,
		IDGenerator:                services.IDGenerator,
		PermissionCacheInvalidator: services.PermissionCacheInvalidator,
	}

	readRepos := ReadRolePermissionRepositories{
		RolePermission: repositories.RolePermission,
		Role:           repositories.Role,
		Permission:     repositories.Permission,
	}
	readServices := ReadRolePermissionServices{
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateRolePermissionRepositories{
		RolePermission: repositories.RolePermission,
		Role:           repositories.Role,
		Permission:     repositories.Permission,
	}
	updateServices := UpdateRolePermissionServices{
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteRolePermissionRepositories{
		RolePermission:      repositories.RolePermission,
		Role:                repositories.Role,
		Permission:          repositories.Permission,
		WorkspaceUserRole:   repositories.WorkspaceUserRole,
		WorkspaceUser:       repositories.WorkspaceUser,
		ClientPortalGrant:   repositories.ClientPortalGrant,
		SupplierPortalGrant: repositories.SupplierPortalGrant,
		DelegateClient:      repositories.DelegateClient,
		DelegateSupplier:    repositories.DelegateSupplier,
	}
	deleteServices := DeleteRolePermissionServices{
		Authorizer:                 services.Authorizer,
		Transactor:                 services.Transactor,
		Translator:                 services.Translator,
		ActionGatekeeper:           services.ActionGatekeeper,
		PermissionCacheInvalidator: services.PermissionCacheInvalidator,
	}

	listRepos := ListRolePermissionsRepositories{
		RolePermission: repositories.RolePermission,
		Role:           repositories.Role,
		Permission:     repositories.Permission,
	}
	listServices := ListRolePermissionsServices{
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
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
		Authorizer:       authorizationService,
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewUseCases(repositories, services)
}
