package entity

import (
	// Entity use cases
	adminUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/admin"
	clientUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/client"
	clientAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/client_attribute"
	clientCategoryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/client_category"
	delegateUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/delegate"
	delegateAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/delegate_attribute"
	delegateClientUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/delegate_client"
	groupUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/group"
	groupAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/group_attribute"
	locationUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/location"
	locationAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/location_attribute"
	permissionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/permission"
	roleUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/role"
	rolePermissionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/role_permission"
	staffUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/staff"
	staffAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/staff_attribute"
	userUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/user"
	workspaceUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/workspace"
	workspaceUserUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/workspace_user"
	workspaceUserRoleUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/workspace_user_role"
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
