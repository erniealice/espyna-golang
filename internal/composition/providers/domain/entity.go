package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	// Protobuf domain services - Common domain
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"

	// Protobuf domain services - Entity domain
	adminpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/admin"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	clientattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_attribute"
	clientcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_category"
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
	delegateattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_attribute"
	delegateclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_client"
	grouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/group"
	groupattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/group_attribute"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
	locationareapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_area"
	locationattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_attribute"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
	rolepermissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role_permission"
	staffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff"
	staffattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff_attribute"
	supplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier"
	supplierattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_attribute"
	suppliercategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_category"
	sessionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/session"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
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
	LocationArea      locationareapb.LocationAreaDomainServiceServer
	LocationAttribute locationattributepb.LocationAttributeDomainServiceServer
	Permission        permissionpb.PermissionDomainServiceServer
	Role              rolepb.RoleDomainServiceServer
	RolePermission    rolepermissionpb.RolePermissionDomainServiceServer
	Staff             staffpb.StaffDomainServiceServer
	StaffAttribute    staffattributepb.StaffAttributeDomainServiceServer
	Supplier          supplierpb.SupplierDomainServiceServer
	SupplierAttribute supplierattributepb.SupplierAttributeDomainServiceServer
	SupplierCategory  suppliercategorypb.SupplierCategoryDomainServiceServer
	Session           sessionpb.SessionDomainServiceServer
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
func NewEntityRepositories(dbProvider contracts.Provider, tableConfig *registry.TableConfig) (*EntityRepositories, error) {
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
	tryCreate := func(entity string) interface{} {
		repo, err := repoCreator.CreateRepository(entity, conn, tableConfig.TableName(entity))
		if err != nil {
			skipped = append(skipped, entity)
			return nil
		}
		return repo
	}

	// Create each repository individually — failures are non-fatal
	if r := tryCreate(entityid.Admin); r != nil {
		repos.Admin = r.(adminpb.AdminDomainServiceServer)
	}
	if r := tryCreate(entityid.Client); r != nil {
		repos.Client = r.(clientpb.ClientDomainServiceServer)
	}
	if r := tryCreate(entityid.ClientAttribute); r != nil {
		repos.ClientAttribute = r.(clientattributepb.ClientAttributeDomainServiceServer)
	}
	if r := tryCreate(entityid.ClientCategory); r != nil {
		repos.ClientCategory = r.(clientcategorypb.ClientCategoryDomainServiceServer)
	}
	if r := tryCreate(entityid.Delegate); r != nil {
		repos.Delegate = r.(delegatepb.DelegateDomainServiceServer)
	}
	if r := tryCreate(entityid.DelegateAttribute); r != nil {
		repos.DelegateAttribute = r.(delegateattributepb.DelegateAttributeDomainServiceServer)
	}
	if r := tryCreate(entityid.DelegateClient); r != nil {
		repos.DelegateClient = r.(delegateclientpb.DelegateClientDomainServiceServer)
	}
	if r := tryCreate(entityid.Group); r != nil {
		repos.Group = r.(grouppb.GroupDomainServiceServer)
	}
	if r := tryCreate(entityid.GroupAttribute); r != nil {
		repos.GroupAttribute = r.(groupattributepb.GroupAttributeDomainServiceServer)
	}
	if r := tryCreate(entityid.Location); r != nil {
		repos.Location = r.(locationpb.LocationDomainServiceServer)
	}
	if r := tryCreate(entityid.LocationArea); r != nil {
		repos.LocationArea = r.(locationareapb.LocationAreaDomainServiceServer)
	}
	if r := tryCreate(entityid.LocationAttribute); r != nil {
		repos.LocationAttribute = r.(locationattributepb.LocationAttributeDomainServiceServer)
	}
	if r := tryCreate(entityid.Permission); r != nil {
		repos.Permission = r.(permissionpb.PermissionDomainServiceServer)
	}
	if r := tryCreate(entityid.Role); r != nil {
		repos.Role = r.(rolepb.RoleDomainServiceServer)
	}
	if r := tryCreate(entityid.RolePermission); r != nil {
		repos.RolePermission = r.(rolepermissionpb.RolePermissionDomainServiceServer)
	}
	if r := tryCreate(entityid.Staff); r != nil {
		repos.Staff = r.(staffpb.StaffDomainServiceServer)
	}
	if r := tryCreate(entityid.StaffAttribute); r != nil {
		repos.StaffAttribute = r.(staffattributepb.StaffAttributeDomainServiceServer)
	}
	if r := tryCreate(entityid.Supplier); r != nil {
		repos.Supplier = r.(supplierpb.SupplierDomainServiceServer)
	}
	if r := tryCreate(entityid.SupplierAttribute); r != nil {
		repos.SupplierAttribute = r.(supplierattributepb.SupplierAttributeDomainServiceServer)
	}
	if r := tryCreate(entityid.SupplierCategory); r != nil {
		repos.SupplierCategory = r.(suppliercategorypb.SupplierCategoryDomainServiceServer)
	}
	if r := tryCreate(entityid.Session); r != nil {
		repos.Session = r.(sessionpb.SessionDomainServiceServer)
	}
	if r := tryCreate(entityid.User); r != nil {
		repos.User = r.(userpb.UserDomainServiceServer)
	}
	if r := tryCreate(entityid.Workspace); r != nil {
		repos.Workspace = r.(workspacepb.WorkspaceDomainServiceServer)
	}
	if r := tryCreate(entityid.WorkspaceUser); r != nil {
		repos.WorkspaceUser = r.(workspaceuserpb.WorkspaceUserDomainServiceServer)
	}
	if r := tryCreate(entityid.WorkspaceUserRole); r != nil {
		repos.WorkspaceUserRole = r.(workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer)
	}
	// Cross-domain dependency: Attribute repository from Common domain
	if r := tryCreate(entityid.Attribute); r != nil {
		repos.Attribute = r.(attributepb.AttributeDomainServiceServer)
	}

	if len(skipped) > 0 {
		fmt.Printf("⚠️  Entity repos skipped (no adapter registered): %v\n", skipped)
	}

	return repos, nil
}
