//go:build mock_auth
package e2e

import (
	"testing"

	"github.com/erniealice/espyna-golang/tests/e2e/helper"
)

// TestEntityDomainCRUDOperations validates create, read, and list operations for Entity domain
func TestEntityDomainCRUDOperations(t *testing.T) {
	env := helper.SetupTestEnvironment(t)

	// Test Admin create/read/list operations
	t.Run("AdminOperations", func(t *testing.T) {
		entityPath := "/api/entity/admin"

		t.Run("CreateAdmin", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("admin")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("admin")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("admin")
			updateData := helper.GetUpdateDataForEntity("admin")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListAdmins", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test Client create/read/list operations
	t.Run("ClientOperations", func(t *testing.T) {
		entityPath := "/api/entity/client"

		t.Run("CreateClient", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("client")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("client")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("client")
			updateData := helper.GetUpdateDataForEntity("client")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListClients", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test ClientAttribute create/read/list operations
	t.Run("ClientAttributeOperations", func(t *testing.T) {
		entityPath := "/api/entity/client-attribute"

		t.Run("CreateClientAttribute", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("client-attribute")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("client-attribute")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("client-attribute")
			updateData := helper.GetUpdateDataForEntity("client-attribute")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListClientAttributes", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test Delegate create/read/list operations
	t.Run("DelegateOperations", func(t *testing.T) {
		entityPath := "/api/entity/delegate"

		t.Run("CreateDelegate", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("delegate")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("delegate")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("delegate")
			updateData := helper.GetUpdateDataForEntity("delegate")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListDelegates", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test DelegateClient create/read/list operations
	t.Run("DelegateClientOperations", func(t *testing.T) {
		entityPath := "/api/entity/delegate-client"

		t.Run("CreateDelegateClient", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("delegate-client")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("delegate-client")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("delegate-client")
			updateData := helper.GetUpdateDataForEntity("delegate-client")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListDelegateClients", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test Group create/read/list operations
	t.Run("GroupOperations", func(t *testing.T) {
		entityPath := "/api/entity/group"

		t.Run("CreateGroup", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("group")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("group")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("group")
			updateData := helper.GetUpdateDataForEntity("group")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListGroups", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test Location create/read/list operations
	t.Run("LocationOperations", func(t *testing.T) {
		entityPath := "/api/entity/location"

		t.Run("CreateLocation", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("location")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("location")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("location")
			updateData := helper.GetUpdateDataForEntity("location")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListLocations", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test LocationAttribute create/read/list operations
	t.Run("LocationAttributeOperations", func(t *testing.T) {
		entityPath := "/api/entity/location-attribute"

		t.Run("CreateLocationAttribute", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("location-attribute")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("location-attribute")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("location-attribute")
			updateData := helper.GetUpdateDataForEntity("location-attribute")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListLocationAttributes", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test Manager create/read/list operations
	t.Run("ManagerOperations", func(t *testing.T) {
		entityPath := "/api/entity/manager"

		t.Run("CreateManager", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("manager")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("manager")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("manager")
			updateData := helper.GetUpdateDataForEntity("manager")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListManagers", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test Permission create/read/list operations
	t.Run("PermissionOperations", func(t *testing.T) {
		entityPath := "/api/entity/permission"

		t.Run("CreatePermission", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("permission")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("permission")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("permission")
			updateData := helper.GetUpdateDataForEntity("permission")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListPermissions", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test Role create/read/list operations
	t.Run("RoleOperations", func(t *testing.T) {
		entityPath := "/api/entity/role"

		t.Run("CreateRole", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("role")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("role")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("role")
			updateData := helper.GetUpdateDataForEntity("role")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListRoles", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test RolePermission create/read/list operations
	t.Run("RolePermissionOperations", func(t *testing.T) {
		entityPath := "/api/entity/role-permission"

		t.Run("CreateRolePermission", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("role-permission")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("role-permission")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("role-permission")
			updateData := helper.GetUpdateDataForEntity("role-permission")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListRolePermissions", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test Staff create/read/list operations
	t.Run("StaffOperations", func(t *testing.T) {
		entityPath := "/api/entity/staff"

		t.Run("CreateStaff", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("staff")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("staff")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("staff")
			updateData := helper.GetUpdateDataForEntity("staff")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListStaffs", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test User create/read/list operations
	t.Run("UserOperations", func(t *testing.T) {
		entityPath := "/api/entity/user"

		t.Run("CreateUser", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("user")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("user")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("user")
			updateData := helper.GetUpdateDataForEntity("user")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListUsers", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test Workspace create/read/list operations
	t.Run("WorkspaceOperations", func(t *testing.T) {
		entityPath := "/api/entity/workspace"

		t.Run("CreateWorkspace", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("workspace")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("workspace")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("workspace")
			updateData := helper.GetUpdateDataForEntity("workspace")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListWorkspaces", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test WorkspaceUser create/read/list operations
	t.Run("WorkspaceUserOperations", func(t *testing.T) {
		entityPath := "/api/entity/workspace-user"

		t.Run("CreateWorkspaceUser", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("workspace-user")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("workspace-user")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("workspace-user")
			updateData := helper.GetUpdateDataForEntity("workspace-user")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListWorkspaceUsers", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test WorkspaceUserRole create/read/list operations
	t.Run("WorkspaceUserRoleOperations", func(t *testing.T) {
		entityPath := "/api/entity/workspace-user-role"

		t.Run("CreateWorkspaceUserRole", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("workspace-user-role")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("workspace-user-role")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("workspace-user-role")
			updateData := helper.GetUpdateDataForEntity("workspace-user-role")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListWorkspaceUserRoles", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})
}
