package role_permission

import (
	"leapfor.xyz/espyna/internal/application/ports"
	permissionpb "leapfor.xyz/esqyma/golang/v1/domain/entity/permission"
	rolepb "leapfor.xyz/esqyma/golang/v1/domain/entity/role"
	rolepermissionpb "leapfor.xyz/esqyma/golang/v1/domain/entity/role_permission"
)

// RolePermissionRepositories groups all repository dependencies for role permission use cases
type RolePermissionRepositories struct {
	RolePermission rolepermissionpb.RolePermissionDomainServiceServer // Primary entity repository
	Role           rolepb.RoleDomainServiceServer                     // Entity reference validation
	Permission     permissionpb.PermissionDomainServiceServer         // Entity reference validation
}

// RolePermissionServices groups all business service dependencies for role permission use cases
type RolePermissionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadRolePermissionRepositories(repositories)
	readServices := ReadRolePermissionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateRolePermissionRepositories(repositories)
	updateServices := UpdateRolePermissionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteRolePermissionRepositories(repositories)
	deleteServices := DeleteRolePermissionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListRolePermissionsRepositories(repositories)
	listServices := ListRolePermissionsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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
	authorizationService ports.AuthorizationService,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := RolePermissionRepositories{
		RolePermission: rolePermissionRepo,
		Role:           roleRepo,
		Permission:     permissionRepo,
	}

	services := RolePermissionServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
