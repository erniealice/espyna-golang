package entity

import (
	// Entity use cases
	adminUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/admin"
	clientUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/client"
	clientAttributeUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/client_attribute"
	clientCategoryUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/client_category"
	delegateUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/delegate"
	delegateAttributeUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/delegate_attribute"
	delegateClientUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/delegate_client"
	groupUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/group"
	groupAttributeUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/group_attribute"
	locationUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/location"
	locationAttributeUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/location_attribute"
	permissionUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/permission"
	roleUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/role"
	rolePermissionUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/role_permission"
	staffUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/staff"
	staffAttributeUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/staff_attribute"
	userUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/user"
	workspaceUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/workspace"
	workspaceUserUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/workspace_user"
	workspaceUserRoleUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/workspace_user_role"
	// Note: Protobuf imports removed as domain-level constructors are no longer used
)

// EntityUseCases contains all entity-related use cases
type EntityUseCases struct {
	Admin             *adminUseCases.UseCases
	Client            *clientUseCases.UseCases
	ClientAttribute   *clientAttributeUseCases.UseCases
	ClientCategory    *clientCategoryUseCases.UseCases
	Delegate          *delegateUseCases.UseCases
	DelegateAttribute *delegateAttributeUseCases.UseCases
	DelegateClient    *delegateClientUseCases.UseCases
	Group             *groupUseCases.UseCases
	GroupAttribute    *groupAttributeUseCases.UseCases
	Location          *locationUseCases.UseCases
	LocationAttribute *locationAttributeUseCases.UseCases
	Permission        *permissionUseCases.UseCases
	Role              *roleUseCases.UseCases
	RolePermission    *rolePermissionUseCases.UseCases
	Staff             *staffUseCases.UseCases
	StaffAttribute    *staffAttributeUseCases.UseCases
	User              *userUseCases.UseCases
	Workspace         *workspaceUseCases.UseCases
	WorkspaceUser     *workspaceUserUseCases.UseCases
	WorkspaceUserRole *workspaceUserRoleUseCases.UseCases
}

// Note: Domain-level constructors are no longer needed with the new architecture.
// Individual use cases are now constructed directly in the repository factory
// with explicit entity reference dependency injection.
