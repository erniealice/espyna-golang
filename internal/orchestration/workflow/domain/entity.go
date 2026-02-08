package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases"
	"github.com/erniealice/espyna-golang/internal/orchestration/workflow/executor"
)

// RegisterEntityUseCases registers all entity domain use cases with the registry.
// Entity domain includes: Admin, Client, Delegate, DelegateClient, Group, Location,
// Permission, Role, RolePermission, Staff, User, Workspace, WorkspaceUser, WorkspaceUserRole
// and their associated attribute entities.
//
// Note: GetListPageData and GetItemPageData use cases are not registered because they
// have entity-specific response types that require explicit generic type parameters.
// These can be added individually if needed for specific workflow automation scenarios.
func RegisterEntityUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Entity == nil {
		return
	}

	// Admin use cases
	registerAdminUseCases(useCases, register)

	// Client use cases
	registerClientUseCases(useCases, register)

	// Delegate use cases
	registerDelegateUseCases(useCases, register)

	// DelegateClient use cases
	registerDelegateClientUseCases(useCases, register)

	// Group use cases
	registerGroupUseCases(useCases, register)

	// Location use cases
	registerLocationUseCases(useCases, register)

	// Permission use cases
	registerPermissionUseCases(useCases, register)

	// Role use cases
	registerRoleUseCases(useCases, register)

	// RolePermission use cases
	registerRolePermissionUseCases(useCases, register)

	// Staff use cases
	registerStaffUseCases(useCases, register)

	// User use cases
	registerUserUseCases(useCases, register)

	// Workspace use cases
	registerWorkspaceUseCases(useCases, register)

	// WorkspaceUser use cases
	registerWorkspaceUserUseCases(useCases, register)

	// WorkspaceUserRole use cases
	registerWorkspaceUserRoleUseCases(useCases, register)
}

// registerAdminUseCases registers admin CRUD use cases.
func registerAdminUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Entity.Admin == nil {
		return
	}

	if useCases.Entity.Admin.CreateAdmin != nil {
		register("entity.admin.create", executor.New(useCases.Entity.Admin.CreateAdmin.Execute))
	}
	if useCases.Entity.Admin.ReadAdmin != nil {
		register("entity.admin.read", executor.New(useCases.Entity.Admin.ReadAdmin.Execute))
	}
	if useCases.Entity.Admin.UpdateAdmin != nil {
		register("entity.admin.update", executor.New(useCases.Entity.Admin.UpdateAdmin.Execute))
	}
	if useCases.Entity.Admin.DeleteAdmin != nil {
		register("entity.admin.delete", executor.New(useCases.Entity.Admin.DeleteAdmin.Execute))
	}
	if useCases.Entity.Admin.ListAdmins != nil {
		register("entity.admin.list", executor.New(useCases.Entity.Admin.ListAdmins.Execute))
	}
}

// registerClientUseCases registers client CRUD and custom use cases.
func registerClientUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Entity.Client == nil {
		return
	}

	if useCases.Entity.Client.CreateClient != nil {
		register("entity.client.create", executor.New(useCases.Entity.Client.CreateClient.Execute))
	}
	if useCases.Entity.Client.ReadClient != nil {
		register("entity.client.read", executor.New(useCases.Entity.Client.ReadClient.Execute))
	}
	if useCases.Entity.Client.UpdateClient != nil {
		register("entity.client.update", executor.New(useCases.Entity.Client.UpdateClient.Execute))
	}
	if useCases.Entity.Client.DeleteClient != nil {
		register("entity.client.delete", executor.New(useCases.Entity.Client.DeleteClient.Execute))
	}
	if useCases.Entity.Client.ListClients != nil {
		register("entity.client.list", executor.New(useCases.Entity.Client.ListClients.Execute))
	}
	// Custom client use cases
	if useCases.Entity.Client.FindOrCreateClient != nil {
		register("entity.client.find_or_create", executor.New(useCases.Entity.Client.FindOrCreateClient.Execute))
	}
	if useCases.Entity.Client.GetClientByEmail != nil {
		register("entity.client.get_by_email", executor.New(useCases.Entity.Client.GetClientByEmail.Execute))
	}
}

// registerDelegateUseCases registers delegate CRUD use cases.
func registerDelegateUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Entity.Delegate == nil {
		return
	}

	if useCases.Entity.Delegate.CreateDelegate != nil {
		register("entity.delegate.create", executor.New(useCases.Entity.Delegate.CreateDelegate.Execute))
	}
	if useCases.Entity.Delegate.ReadDelegate != nil {
		register("entity.delegate.read", executor.New(useCases.Entity.Delegate.ReadDelegate.Execute))
	}
	if useCases.Entity.Delegate.UpdateDelegate != nil {
		register("entity.delegate.update", executor.New(useCases.Entity.Delegate.UpdateDelegate.Execute))
	}
	if useCases.Entity.Delegate.DeleteDelegate != nil {
		register("entity.delegate.delete", executor.New(useCases.Entity.Delegate.DeleteDelegate.Execute))
	}
	if useCases.Entity.Delegate.ListDelegates != nil {
		register("entity.delegate.list", executor.New(useCases.Entity.Delegate.ListDelegates.Execute))
	}
}

// registerDelegateClientUseCases registers delegate client CRUD use cases.
func registerDelegateClientUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Entity.DelegateClient == nil {
		return
	}

	if useCases.Entity.DelegateClient.CreateDelegateClient != nil {
		register("entity.delegate_client.create", executor.New(useCases.Entity.DelegateClient.CreateDelegateClient.Execute))
	}
	if useCases.Entity.DelegateClient.ReadDelegateClient != nil {
		register("entity.delegate_client.read", executor.New(useCases.Entity.DelegateClient.ReadDelegateClient.Execute))
	}
	if useCases.Entity.DelegateClient.UpdateDelegateClient != nil {
		register("entity.delegate_client.update", executor.New(useCases.Entity.DelegateClient.UpdateDelegateClient.Execute))
	}
	if useCases.Entity.DelegateClient.DeleteDelegateClient != nil {
		register("entity.delegate_client.delete", executor.New(useCases.Entity.DelegateClient.DeleteDelegateClient.Execute))
	}
	if useCases.Entity.DelegateClient.ListDelegateClients != nil {
		register("entity.delegate_client.list", executor.New(useCases.Entity.DelegateClient.ListDelegateClients.Execute))
	}
}

// registerGroupUseCases registers group CRUD use cases.
func registerGroupUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Entity.Group == nil {
		return
	}

	if useCases.Entity.Group.CreateGroup != nil {
		register("entity.group.create", executor.New(useCases.Entity.Group.CreateGroup.Execute))
	}
	if useCases.Entity.Group.ReadGroup != nil {
		register("entity.group.read", executor.New(useCases.Entity.Group.ReadGroup.Execute))
	}
	if useCases.Entity.Group.UpdateGroup != nil {
		register("entity.group.update", executor.New(useCases.Entity.Group.UpdateGroup.Execute))
	}
	if useCases.Entity.Group.DeleteGroup != nil {
		register("entity.group.delete", executor.New(useCases.Entity.Group.DeleteGroup.Execute))
	}
	if useCases.Entity.Group.ListGroups != nil {
		register("entity.group.list", executor.New(useCases.Entity.Group.ListGroups.Execute))
	}
}

// registerLocationUseCases registers location CRUD use cases.
func registerLocationUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Entity.Location == nil {
		return
	}

	if useCases.Entity.Location.CreateLocation != nil {
		register("entity.location.create", executor.New(useCases.Entity.Location.CreateLocation.Execute))
	}
	if useCases.Entity.Location.ReadLocation != nil {
		register("entity.location.read", executor.New(useCases.Entity.Location.ReadLocation.Execute))
	}
	if useCases.Entity.Location.UpdateLocation != nil {
		register("entity.location.update", executor.New(useCases.Entity.Location.UpdateLocation.Execute))
	}
	if useCases.Entity.Location.DeleteLocation != nil {
		register("entity.location.delete", executor.New(useCases.Entity.Location.DeleteLocation.Execute))
	}
	if useCases.Entity.Location.ListLocations != nil {
		register("entity.location.list", executor.New(useCases.Entity.Location.ListLocations.Execute))
	}
}

// registerPermissionUseCases registers permission CRUD use cases.
func registerPermissionUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Entity.Permission == nil {
		return
	}

	if useCases.Entity.Permission.CreatePermission != nil {
		register("entity.permission.create", executor.New(useCases.Entity.Permission.CreatePermission.Execute))
	}
	if useCases.Entity.Permission.ReadPermission != nil {
		register("entity.permission.read", executor.New(useCases.Entity.Permission.ReadPermission.Execute))
	}
	if useCases.Entity.Permission.UpdatePermission != nil {
		register("entity.permission.update", executor.New(useCases.Entity.Permission.UpdatePermission.Execute))
	}
	if useCases.Entity.Permission.DeletePermission != nil {
		register("entity.permission.delete", executor.New(useCases.Entity.Permission.DeletePermission.Execute))
	}
	if useCases.Entity.Permission.ListPermissions != nil {
		register("entity.permission.list", executor.New(useCases.Entity.Permission.ListPermissions.Execute))
	}
}

// registerRoleUseCases registers role CRUD use cases.
func registerRoleUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Entity.Role == nil {
		return
	}

	if useCases.Entity.Role.CreateRole != nil {
		register("entity.role.create", executor.New(useCases.Entity.Role.CreateRole.Execute))
	}
	if useCases.Entity.Role.ReadRole != nil {
		register("entity.role.read", executor.New(useCases.Entity.Role.ReadRole.Execute))
	}
	if useCases.Entity.Role.UpdateRole != nil {
		register("entity.role.update", executor.New(useCases.Entity.Role.UpdateRole.Execute))
	}
	if useCases.Entity.Role.DeleteRole != nil {
		register("entity.role.delete", executor.New(useCases.Entity.Role.DeleteRole.Execute))
	}
	if useCases.Entity.Role.ListRoles != nil {
		register("entity.role.list", executor.New(useCases.Entity.Role.ListRoles.Execute))
	}
}

// registerRolePermissionUseCases registers role_permission CRUD use cases.
func registerRolePermissionUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Entity.RolePermission == nil {
		return
	}

	if useCases.Entity.RolePermission.CreateRolePermission != nil {
		register("entity.role_permission.create", executor.New(useCases.Entity.RolePermission.CreateRolePermission.Execute))
	}
	if useCases.Entity.RolePermission.ReadRolePermission != nil {
		register("entity.role_permission.read", executor.New(useCases.Entity.RolePermission.ReadRolePermission.Execute))
	}
	if useCases.Entity.RolePermission.DeleteRolePermission != nil {
		register("entity.role_permission.delete", executor.New(useCases.Entity.RolePermission.DeleteRolePermission.Execute))
	}
	if useCases.Entity.RolePermission.ListRolePermissions != nil {
		register("entity.role_permission.list", executor.New(useCases.Entity.RolePermission.ListRolePermissions.Execute))
	}
}

// registerStaffUseCases registers staff CRUD use cases.
func registerStaffUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Entity.Staff == nil {
		return
	}

	if useCases.Entity.Staff.CreateStaff != nil {
		register("entity.staff.create", executor.New(useCases.Entity.Staff.CreateStaff.Execute))
	}
	if useCases.Entity.Staff.ReadStaff != nil {
		register("entity.staff.read", executor.New(useCases.Entity.Staff.ReadStaff.Execute))
	}
	if useCases.Entity.Staff.UpdateStaff != nil {
		register("entity.staff.update", executor.New(useCases.Entity.Staff.UpdateStaff.Execute))
	}
	if useCases.Entity.Staff.DeleteStaff != nil {
		register("entity.staff.delete", executor.New(useCases.Entity.Staff.DeleteStaff.Execute))
	}
	if useCases.Entity.Staff.ListStaffs != nil {
		register("entity.staff.list", executor.New(useCases.Entity.Staff.ListStaffs.Execute))
	}
}

// registerUserUseCases registers user CRUD use cases.
func registerUserUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Entity.User == nil {
		return
	}

	if useCases.Entity.User.CreateUser != nil {
		register("entity.user.create", executor.New(useCases.Entity.User.CreateUser.Execute))
	}
	if useCases.Entity.User.ReadUser != nil {
		register("entity.user.read", executor.New(useCases.Entity.User.ReadUser.Execute))
	}
	if useCases.Entity.User.UpdateUser != nil {
		register("entity.user.update", executor.New(useCases.Entity.User.UpdateUser.Execute))
	}
	if useCases.Entity.User.DeleteUser != nil {
		register("entity.user.delete", executor.New(useCases.Entity.User.DeleteUser.Execute))
	}
	if useCases.Entity.User.ListUsers != nil {
		register("entity.user.list", executor.New(useCases.Entity.User.ListUsers.Execute))
	}
}

// registerWorkspaceUseCases registers workspace CRUD use cases.
func registerWorkspaceUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Entity.Workspace == nil {
		return
	}

	if useCases.Entity.Workspace.CreateWorkspace != nil {
		register("entity.workspace.create", executor.New(useCases.Entity.Workspace.CreateWorkspace.Execute))
	}
	if useCases.Entity.Workspace.ReadWorkspace != nil {
		register("entity.workspace.read", executor.New(useCases.Entity.Workspace.ReadWorkspace.Execute))
	}
	if useCases.Entity.Workspace.UpdateWorkspace != nil {
		register("entity.workspace.update", executor.New(useCases.Entity.Workspace.UpdateWorkspace.Execute))
	}
	if useCases.Entity.Workspace.DeleteWorkspace != nil {
		register("entity.workspace.delete", executor.New(useCases.Entity.Workspace.DeleteWorkspace.Execute))
	}
	if useCases.Entity.Workspace.ListWorkspaces != nil {
		register("entity.workspace.list", executor.New(useCases.Entity.Workspace.ListWorkspaces.Execute))
	}
}

// registerWorkspaceUserUseCases registers workspace user CRUD use cases.
func registerWorkspaceUserUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Entity.WorkspaceUser == nil {
		return
	}

	if useCases.Entity.WorkspaceUser.CreateWorkspaceUser != nil {
		register("entity.workspace_user.create", executor.New(useCases.Entity.WorkspaceUser.CreateWorkspaceUser.Execute))
	}
	if useCases.Entity.WorkspaceUser.ReadWorkspaceUser != nil {
		register("entity.workspace_user.read", executor.New(useCases.Entity.WorkspaceUser.ReadWorkspaceUser.Execute))
	}
	if useCases.Entity.WorkspaceUser.UpdateWorkspaceUser != nil {
		register("entity.workspace_user.update", executor.New(useCases.Entity.WorkspaceUser.UpdateWorkspaceUser.Execute))
	}
	if useCases.Entity.WorkspaceUser.DeleteWorkspaceUser != nil {
		register("entity.workspace_user.delete", executor.New(useCases.Entity.WorkspaceUser.DeleteWorkspaceUser.Execute))
	}
	if useCases.Entity.WorkspaceUser.ListWorkspaceUsers != nil {
		register("entity.workspace_user.list", executor.New(useCases.Entity.WorkspaceUser.ListWorkspaceUsers.Execute))
	}
}

// registerWorkspaceUserRoleUseCases registers workspace user role CRUD use cases.
func registerWorkspaceUserRoleUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Entity.WorkspaceUserRole == nil {
		return
	}

	if useCases.Entity.WorkspaceUserRole.CreateWorkspaceUserRole != nil {
		register("entity.workspace_user_role.create", executor.New(useCases.Entity.WorkspaceUserRole.CreateWorkspaceUserRole.Execute))
	}
	if useCases.Entity.WorkspaceUserRole.ReadWorkspaceUserRole != nil {
		register("entity.workspace_user_role.read", executor.New(useCases.Entity.WorkspaceUserRole.ReadWorkspaceUserRole.Execute))
	}
	if useCases.Entity.WorkspaceUserRole.UpdateWorkspaceUserRole != nil {
		register("entity.workspace_user_role.update", executor.New(useCases.Entity.WorkspaceUserRole.UpdateWorkspaceUserRole.Execute))
	}
	if useCases.Entity.WorkspaceUserRole.DeleteWorkspaceUserRole != nil {
		register("entity.workspace_user_role.delete", executor.New(useCases.Entity.WorkspaceUserRole.DeleteWorkspaceUserRole.Execute))
	}
	if useCases.Entity.WorkspaceUserRole.ListWorkspaceUserRoles != nil {
		register("entity.workspace_user_role.list", executor.New(useCases.Entity.WorkspaceUserRole.ListWorkspaceUserRoles.Execute))
	}
}
