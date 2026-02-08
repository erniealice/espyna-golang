package domain

import (
	"fmt"

	"leapfor.xyz/espyna/internal/composition/contracts"
	"leapfor.xyz/espyna/internal/infrastructure/registry"

	// Protobuf domain services - Common domain
	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"

	// Protobuf domain services - Entity domain
	adminpb "leapfor.xyz/esqyma/golang/v1/domain/entity/admin"
	clientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/client"
	clientattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/client_attribute"
	clientcategorypb "leapfor.xyz/esqyma/golang/v1/domain/entity/client_category"
	delegatepb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate"
	delegateattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate_attribute"
	delegateclientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate_client"
	grouppb "leapfor.xyz/esqyma/golang/v1/domain/entity/group"
	groupattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/group_attribute"
	locationpb "leapfor.xyz/esqyma/golang/v1/domain/entity/location"
	locationattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/location_attribute"
	permissionpb "leapfor.xyz/esqyma/golang/v1/domain/entity/permission"
	rolepb "leapfor.xyz/esqyma/golang/v1/domain/entity/role"
	rolepermissionpb "leapfor.xyz/esqyma/golang/v1/domain/entity/role_permission"
	staffpb "leapfor.xyz/esqyma/golang/v1/domain/entity/staff"
	staffattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/staff_attribute"
	userpb "leapfor.xyz/esqyma/golang/v1/domain/entity/user"
	workspacepb "leapfor.xyz/esqyma/golang/v1/domain/entity/workspace"
	workspaceuserpb "leapfor.xyz/esqyma/golang/v1/domain/entity/workspace_user"
	workspaceuserrolepb "leapfor.xyz/esqyma/golang/v1/domain/entity/workspace_user_role"
)

// EntityRepositories contains all 20 entity domain repositories plus common repositories
type EntityRepositories struct {
	Admin             adminpb.AdminDomainServiceServer
	Client            clientpb.ClientDomainServiceServer
	ClientAttribute   clientattributepb.ClientAttributeDomainServiceServer
	ClientCategory    clientcategorypb.ClientCategoryDomainServiceServer
	Delegate          delegatepb.DelegateDomainServiceServer
	DelegateAttribute delegateattributepb.DelegateAttributeDomainServiceServer
	DelegateClient    delegateclientpb.DelegateClientDomainServiceServer
	Group             grouppb.GroupDomainServiceServer
	GroupAttribute    groupattributepb.GroupAttributeDomainServiceServer
	Location          locationpb.LocationDomainServiceServer
	LocationAttribute locationattributepb.LocationAttributeDomainServiceServer
	Permission        permissionpb.PermissionDomainServiceServer
	Role              rolepb.RoleDomainServiceServer
	RolePermission    rolepermissionpb.RolePermissionDomainServiceServer
	Staff             staffpb.StaffDomainServiceServer
	StaffAttribute    staffattributepb.StaffAttributeDomainServiceServer
	User              userpb.UserDomainServiceServer
	Workspace         workspacepb.WorkspaceDomainServiceServer
	WorkspaceUser     workspaceuserpb.WorkspaceUserDomainServiceServer
	WorkspaceUserRole workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer
	// Cross-domain dependency from Common domain
	Attribute attributepb.AttributeDomainServiceServer
}

// NewEntityRepositories creates and returns a new set of EntityRepositories.
// Individual repository failures are logged but do not prevent other repositories
// from being created (graceful degradation per-repository).
func NewEntityRepositories(dbProvider contracts.Provider, dbTableConfig *registry.DatabaseTableConfig) (*EntityRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()
	repos := &EntityRepositories{}
	var skipped []string

	// Helper: try to create a repository, log and skip on failure
	tryCreate := func(name string, tableName string) interface{} {
		repo, err := repoCreator.CreateRepository(name, conn, tableName)
		if err != nil {
			skipped = append(skipped, name)
			return nil
		}
		return repo
	}

	// Create each repository individually — failures are non-fatal
	if r := tryCreate("admin", dbTableConfig.Admin); r != nil {
		repos.Admin = r.(adminpb.AdminDomainServiceServer)
	}
	if r := tryCreate("client", dbTableConfig.Client); r != nil {
		repos.Client = r.(clientpb.ClientDomainServiceServer)
	}
	if r := tryCreate("client_attribute", dbTableConfig.ClientAttribute); r != nil {
		repos.ClientAttribute = r.(clientattributepb.ClientAttributeDomainServiceServer)
	}
	if r := tryCreate("client_category", dbTableConfig.ClientCategory); r != nil {
		repos.ClientCategory = r.(clientcategorypb.ClientCategoryDomainServiceServer)
	}
	if r := tryCreate("delegate", dbTableConfig.Delegate); r != nil {
		repos.Delegate = r.(delegatepb.DelegateDomainServiceServer)
	}
	if r := tryCreate("delegate_attribute", dbTableConfig.DelegateAttribute); r != nil {
		repos.DelegateAttribute = r.(delegateattributepb.DelegateAttributeDomainServiceServer)
	}
	if r := tryCreate("delegate_client", dbTableConfig.DelegateClient); r != nil {
		repos.DelegateClient = r.(delegateclientpb.DelegateClientDomainServiceServer)
	}
	if r := tryCreate("group", dbTableConfig.Group); r != nil {
		repos.Group = r.(grouppb.GroupDomainServiceServer)
	}
	if r := tryCreate("group_attribute", dbTableConfig.GroupAttribute); r != nil {
		repos.GroupAttribute = r.(groupattributepb.GroupAttributeDomainServiceServer)
	}
	if r := tryCreate("location", dbTableConfig.Location); r != nil {
		repos.Location = r.(locationpb.LocationDomainServiceServer)
	}
	if r := tryCreate("location_attribute", dbTableConfig.LocationAttribute); r != nil {
		repos.LocationAttribute = r.(locationattributepb.LocationAttributeDomainServiceServer)
	}
	if r := tryCreate("permission", dbTableConfig.Permission); r != nil {
		repos.Permission = r.(permissionpb.PermissionDomainServiceServer)
	}
	if r := tryCreate("role", dbTableConfig.Role); r != nil {
		repos.Role = r.(rolepb.RoleDomainServiceServer)
	}
	if r := tryCreate("role_permission", dbTableConfig.RolePermission); r != nil {
		repos.RolePermission = r.(rolepermissionpb.RolePermissionDomainServiceServer)
	}
	if r := tryCreate("staff", dbTableConfig.Staff); r != nil {
		repos.Staff = r.(staffpb.StaffDomainServiceServer)
	}
	if r := tryCreate("staff_attribute", dbTableConfig.StaffAttribute); r != nil {
		repos.StaffAttribute = r.(staffattributepb.StaffAttributeDomainServiceServer)
	}
	if r := tryCreate("user", dbTableConfig.User); r != nil {
		repos.User = r.(userpb.UserDomainServiceServer)
	}
	if r := tryCreate("workspace", dbTableConfig.Workspace); r != nil {
		repos.Workspace = r.(workspacepb.WorkspaceDomainServiceServer)
	}
	if r := tryCreate("workspace_user", dbTableConfig.WorkspaceUser); r != nil {
		repos.WorkspaceUser = r.(workspaceuserpb.WorkspaceUserDomainServiceServer)
	}
	if r := tryCreate("workspace_user_role", dbTableConfig.WorkspaceUserRole); r != nil {
		repos.WorkspaceUserRole = r.(workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer)
	}
	// Cross-domain dependency: Attribute repository from Common domain
	if r := tryCreate("attribute", dbTableConfig.Attribute); r != nil {
		repos.Attribute = r.(attributepb.AttributeDomainServiceServer)
	}

	if len(skipped) > 0 {
		fmt.Printf("⚠️  Entity repos skipped (no adapter registered): %v\n", skipped)
	}

	return repos, nil
}
