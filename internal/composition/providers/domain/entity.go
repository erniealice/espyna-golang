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

// NewEntityRepositories creates and returns a new set of EntityRepositories
func NewEntityRepositories(dbProvider contracts.Provider, dbTableConfig *registry.DatabaseTableConfig) (*EntityRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()

	// Create each repository individually using configured table names directly from dbTableConfig
	adminRepo, err := repoCreator.CreateRepository("admin", conn, dbTableConfig.Admin)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin repository: %w", err)
	}

	clientRepo, err := repoCreator.CreateRepository("client", conn, dbTableConfig.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to create client repository: %w", err)
	}

	clientAttributeRepo, err := repoCreator.CreateRepository("client_attribute", conn, dbTableConfig.ClientAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create client_attribute repository: %w", err)
	}

	clientCategoryRepo, err := repoCreator.CreateRepository("client_category", conn, dbTableConfig.ClientCategory)
	if err != nil {
		return nil, fmt.Errorf("failed to create client_category repository: %w", err)
	}

	delegateRepo, err := repoCreator.CreateRepository("delegate", conn, dbTableConfig.Delegate)
	if err != nil {
		return nil, fmt.Errorf("failed to create delegate repository: %w", err)
	}

	delegateAttributeRepo, err := repoCreator.CreateRepository("delegate_attribute", conn, dbTableConfig.DelegateAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create delegate_attribute repository: %w", err)
	}

	delegateClientRepo, err := repoCreator.CreateRepository("delegate_client", conn, dbTableConfig.DelegateClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create delegate_client repository: %w", err)
	}

	groupRepo, err := repoCreator.CreateRepository("group", conn, dbTableConfig.Group)
	if err != nil {
		return nil, fmt.Errorf("failed to create group repository: %w", err)
	}

	groupAttributeRepo, err := repoCreator.CreateRepository("group_attribute", conn, dbTableConfig.GroupAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create group_attribute repository: %w", err)
	}

	locationRepo, err := repoCreator.CreateRepository("location", conn, dbTableConfig.Location)
	if err != nil {
		return nil, fmt.Errorf("failed to create location repository: %w", err)
	}

	locationAttributeRepo, err := repoCreator.CreateRepository("location_attribute", conn, dbTableConfig.LocationAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create location_attribute repository: %w", err)
	}

	permissionRepo, err := repoCreator.CreateRepository("permission", conn, dbTableConfig.Permission)
	if err != nil {
		return nil, fmt.Errorf("failed to create permission repository: %w", err)
	}

	roleRepo, err := repoCreator.CreateRepository("role", conn, dbTableConfig.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to create role repository: %w", err)
	}

	rolePermissionRepo, err := repoCreator.CreateRepository("role_permission", conn, dbTableConfig.RolePermission)
	if err != nil {
		return nil, fmt.Errorf("failed to create role_permission repository: %w", err)
	}

	staffRepo, err := repoCreator.CreateRepository("staff", conn, dbTableConfig.Staff)
	if err != nil {
		return nil, fmt.Errorf("failed to create staff repository: %w", err)
	}

	staffAttributeRepo, err := repoCreator.CreateRepository("staff_attribute", conn, dbTableConfig.StaffAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create staff_attribute repository: %w", err)
	}

	userRepo, err := repoCreator.CreateRepository("user", conn, dbTableConfig.User)
	if err != nil {
		return nil, fmt.Errorf("failed to create user repository: %w", err)
	}

	workspaceRepo, err := repoCreator.CreateRepository("workspace", conn, dbTableConfig.Workspace)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace repository: %w", err)
	}

	workspaceUserRepo, err := repoCreator.CreateRepository("workspace_user", conn, dbTableConfig.WorkspaceUser)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace_user repository: %w", err)
	}

	workspaceUserRoleRepo, err := repoCreator.CreateRepository("workspace_user_role", conn, dbTableConfig.WorkspaceUserRole)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace_user_role repository: %w", err)
	}

	// Cross-domain dependency: Attribute repository from Common domain
	attributeRepo, err := repoCreator.CreateRepository("attribute", conn, dbTableConfig.Attribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create attribute repository: %w", err)
	}

	return &EntityRepositories{
		Admin:             adminRepo.(adminpb.AdminDomainServiceServer),
		Client:            clientRepo.(clientpb.ClientDomainServiceServer),
		ClientAttribute:   clientAttributeRepo.(clientattributepb.ClientAttributeDomainServiceServer),
		ClientCategory:    clientCategoryRepo.(clientcategorypb.ClientCategoryDomainServiceServer),
		Delegate:          delegateRepo.(delegatepb.DelegateDomainServiceServer),
		DelegateAttribute: delegateAttributeRepo.(delegateattributepb.DelegateAttributeDomainServiceServer),
		DelegateClient:    delegateClientRepo.(delegateclientpb.DelegateClientDomainServiceServer),
		Group:             groupRepo.(grouppb.GroupDomainServiceServer),
		GroupAttribute:    groupAttributeRepo.(groupattributepb.GroupAttributeDomainServiceServer),
		Location:          locationRepo.(locationpb.LocationDomainServiceServer),
		LocationAttribute: locationAttributeRepo.(locationattributepb.LocationAttributeDomainServiceServer),
		Permission:        permissionRepo.(permissionpb.PermissionDomainServiceServer),
		Role:              roleRepo.(rolepb.RoleDomainServiceServer),
		RolePermission:    rolePermissionRepo.(rolepermissionpb.RolePermissionDomainServiceServer),
		Staff:             staffRepo.(staffpb.StaffDomainServiceServer),
		StaffAttribute:    staffAttributeRepo.(staffattributepb.StaffAttributeDomainServiceServer),
		User:              userRepo.(userpb.UserDomainServiceServer),
		Workspace:         workspaceRepo.(workspacepb.WorkspaceDomainServiceServer),
		WorkspaceUser:     workspaceUserRepo.(workspaceuserpb.WorkspaceUserDomainServiceServer),
		WorkspaceUserRole: workspaceUserRoleRepo.(workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer),
		Attribute:         attributeRepo.(attributepb.AttributeDomainServiceServer),
	}, nil
}
