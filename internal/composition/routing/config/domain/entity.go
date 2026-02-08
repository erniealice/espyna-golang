package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/usecases/entity"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"

	// Protobuf imports with module naming pattern
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
	locationattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_attribute"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
	rolepermissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role_permission"
	staffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff"
	staffattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff_attribute"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

// ConfigureEntityDomain configures routes for the Entity domain with use cases injected directly
func ConfigureEntityDomain(entityUseCases *entity.EntityUseCases) contracts.DomainRouteConfiguration {
	if entityUseCases == nil {
		fmt.Printf("⚠️  Entity use cases is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "entity",
			Prefix:  "/entity",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}

	// Validate all essential use cases are properly initialized
	if entityUseCases.Admin == nil {
		fmt.Printf("⚠️  Entity.Admin use case is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "entity",
			Prefix:  "/entity",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}
	if entityUseCases.Client == nil {
		fmt.Printf("⚠️  Entity.Client use case is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "entity",
			Prefix:  "/entity",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}
	if entityUseCases.User == nil {
		fmt.Printf("⚠️  Entity.User use case is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "entity",
			Prefix:  "/entity",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}
	if entityUseCases.Workspace == nil {
		fmt.Printf("⚠️  Entity.Workspace use case is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "entity",
			Prefix:  "/entity",
			Enabled: false,
			Routes:  []contracts.RouteConfiguration{},
		}
	}
	fmt.Printf("✅ Entity use cases are properly initialized!\n")

	routes := []contracts.RouteConfiguration{}

	// Admin routes (Primary Entity)
	if entityUseCases.Admin != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/admin/create",
				Handler: contracts.NewGenericHandler(entityUseCases.Admin.CreateAdmin, &adminpb.CreateAdminRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/admin/read",
				Handler: contracts.NewGenericHandler(entityUseCases.Admin.ReadAdmin, &adminpb.ReadAdminRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/admin/update",
				Handler: contracts.NewGenericHandler(entityUseCases.Admin.UpdateAdmin, &adminpb.UpdateAdminRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/admin/delete",
				Handler: contracts.NewGenericHandler(entityUseCases.Admin.DeleteAdmin, &adminpb.DeleteAdminRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/admin/list",
				Handler: contracts.NewGenericHandler(entityUseCases.Admin.ListAdmins, &adminpb.ListAdminsRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/admin/get-list-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.Admin.GetAdminListPageData, &adminpb.GetAdminListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/admin/get-item-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.Admin.GetAdminItemPageData, &adminpb.GetAdminItemPageDataRequest{}),
			},
		)
	}

	// Client routes (Primary Entity)
	if entityUseCases.Client != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/client/create",
				Handler: contracts.NewGenericHandler(entityUseCases.Client.CreateClient, &clientpb.CreateClientRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/client/read",
				Handler: contracts.NewGenericHandler(entityUseCases.Client.ReadClient, &clientpb.ReadClientRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/client/update",
				Handler: contracts.NewGenericHandler(entityUseCases.Client.UpdateClient, &clientpb.UpdateClientRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/client/delete",
				Handler: contracts.NewGenericHandler(entityUseCases.Client.DeleteClient, &clientpb.DeleteClientRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/client/list",
				Handler: contracts.NewGenericHandler(entityUseCases.Client.ListClients, &clientpb.ListClientsRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/client/get-list-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.Client.GetClientListPageData, &clientpb.GetClientListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/client/get-item-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.Client.GetClientItemPageData, &clientpb.GetClientItemPageDataRequest{}),
			},
		)
	}

	// ClientAttribute routes
	if entityUseCases.ClientAttribute != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/client-attribute/create",
				Handler: contracts.NewGenericHandler(entityUseCases.ClientAttribute.CreateClientAttribute, &clientattributepb.CreateClientAttributeRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/client-attribute/read",
				Handler: contracts.NewGenericHandler(entityUseCases.ClientAttribute.ReadClientAttribute, &clientattributepb.ReadClientAttributeRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/client-attribute/update",
				Handler: contracts.NewGenericHandler(entityUseCases.ClientAttribute.UpdateClientAttribute, &clientattributepb.UpdateClientAttributeRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/client-attribute/delete",
				Handler: contracts.NewGenericHandler(entityUseCases.ClientAttribute.DeleteClientAttribute, &clientattributepb.DeleteClientAttributeRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/client-attribute/list",
				Handler: contracts.NewGenericHandler(entityUseCases.ClientAttribute.ListClientAttributes, &clientattributepb.ListClientAttributesRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/client-attribute/get-list-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.ClientAttribute.GetClientAttributeListPageData, &clientattributepb.GetClientAttributeListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/client-attribute/get-item-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.ClientAttribute.GetClientAttributeItemPageData, &clientattributepb.GetClientAttributeItemPageDataRequest{}),
			},
		)
	}

	// ClientCategory routes
	if entityUseCases.ClientCategory != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/client-category/create",
				Handler: contracts.NewGenericHandler(entityUseCases.ClientCategory.CreateClientCategory, &clientcategorypb.CreateClientCategoryRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/client-category/read",
				Handler: contracts.NewGenericHandler(entityUseCases.ClientCategory.ReadClientCategory, &clientcategorypb.ReadClientCategoryRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/client-category/update",
				Handler: contracts.NewGenericHandler(entityUseCases.ClientCategory.UpdateClientCategory, &clientcategorypb.UpdateClientCategoryRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/client-category/delete",
				Handler: contracts.NewGenericHandler(entityUseCases.ClientCategory.DeleteClientCategory, &clientcategorypb.DeleteClientCategoryRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/client-category/list",
				Handler: contracts.NewGenericHandler(entityUseCases.ClientCategory.ListClientCategories, &clientcategorypb.ListClientCategoriesRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/client-category/get-list-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.ClientCategory.GetClientCategoryListPageData, &clientcategorypb.GetClientCategoryListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/client-category/get-item-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.ClientCategory.GetClientCategoryItemPageData, &clientcategorypb.GetClientCategoryItemPageDataRequest{}),
			},
		)
	}

	// Delegate routes
	if entityUseCases.Delegate != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/delegate/create",
				Handler: contracts.NewGenericHandler(entityUseCases.Delegate.CreateDelegate, &delegatepb.CreateDelegateRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/delegate/read",
				Handler: contracts.NewGenericHandler(entityUseCases.Delegate.ReadDelegate, &delegatepb.ReadDelegateRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/delegate/update",
				Handler: contracts.NewGenericHandler(entityUseCases.Delegate.UpdateDelegate, &delegatepb.UpdateDelegateRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/delegate/delete",
				Handler: contracts.NewGenericHandler(entityUseCases.Delegate.DeleteDelegate, &delegatepb.DeleteDelegateRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/delegate/list",
				Handler: contracts.NewGenericHandler(entityUseCases.Delegate.ListDelegates, &delegatepb.ListDelegatesRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/delegate/get-list-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.Delegate.GetDelegateListPageData, &delegatepb.GetDelegateListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/delegate/get-item-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.Delegate.GetDelegateItemPageData, &delegatepb.GetDelegateItemPageDataRequest{}),
			},
		)
	}

	// DelegateAttribute routes
	if entityUseCases.DelegateAttribute != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/delegate-attribute/create",
				Handler: contracts.NewGenericHandler(entityUseCases.DelegateAttribute.CreateDelegateAttribute, &delegateattributepb.CreateDelegateAttributeRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/delegate-attribute/read",
				Handler: contracts.NewGenericHandler(entityUseCases.DelegateAttribute.ReadDelegateAttribute, &delegateattributepb.ReadDelegateAttributeRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/delegate-attribute/update",
				Handler: contracts.NewGenericHandler(entityUseCases.DelegateAttribute.UpdateDelegateAttribute, &delegateattributepb.UpdateDelegateAttributeRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/delegate-attribute/delete",
				Handler: contracts.NewGenericHandler(entityUseCases.DelegateAttribute.DeleteDelegateAttribute, &delegateattributepb.DeleteDelegateAttributeRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/delegate-attribute/list",
				Handler: contracts.NewGenericHandler(entityUseCases.DelegateAttribute.ListDelegateAttributes, &delegateattributepb.ListDelegateAttributesRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/delegate-attribute/get-list-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.DelegateAttribute.GetDelegateAttributeListPageData, &delegateattributepb.GetDelegateAttributeListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/delegate-attribute/get-item-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.DelegateAttribute.GetDelegateAttributeItemPageData, &delegateattributepb.GetDelegateAttributeItemPageDataRequest{}),
			},
		)
	}

	// DelegateClient routes
	if entityUseCases.DelegateClient != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/delegate-client/create",
				Handler: contracts.NewGenericHandler(entityUseCases.DelegateClient.CreateDelegateClient, &delegateclientpb.CreateDelegateClientRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/delegate-client/read",
				Handler: contracts.NewGenericHandler(entityUseCases.DelegateClient.ReadDelegateClient, &delegateclientpb.ReadDelegateClientRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/delegate-client/update",
				Handler: contracts.NewGenericHandler(entityUseCases.DelegateClient.UpdateDelegateClient, &delegateclientpb.UpdateDelegateClientRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/delegate-client/delete",
				Handler: contracts.NewGenericHandler(entityUseCases.DelegateClient.DeleteDelegateClient, &delegateclientpb.DeleteDelegateClientRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/delegate-client/list",
				Handler: contracts.NewGenericHandler(entityUseCases.DelegateClient.ListDelegateClients, &delegateclientpb.ListDelegateClientsRequest{}),
			},
			// contracts.RouteConfiguration{
			// 	Method:  "POST",
			// 	Path:    "/api/entity/delegate-client/get-list-page-data",
			// 	Handler: contracts.NewGenericHandler(entityUseCases.DelegateClient.GetDelegateClientListPageData, &delegateclientpb.GetDelegateClientListPageDataRequest{}),
			// },
			// contracts.RouteConfiguration{
			// 	Method:  "POST",
			// 	Path:    "/api/entity/delegate-client/get-item-page-data",
			// 	Handler: contracts.NewGenericHandler(entityUseCases.DelegateClient.GetDelegateClientItemPageData, &delegateclientpb.GetDelegateClientItemPageDataRequest{}),
			// },
		)
	}

	// Group routes
	if entityUseCases.Group != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/group/create",
				Handler: contracts.NewGenericHandler(entityUseCases.Group.CreateGroup, &grouppb.CreateGroupRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/group/read",
				Handler: contracts.NewGenericHandler(entityUseCases.Group.ReadGroup, &grouppb.ReadGroupRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/group/update",
				Handler: contracts.NewGenericHandler(entityUseCases.Group.UpdateGroup, &grouppb.UpdateGroupRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/group/delete",
				Handler: contracts.NewGenericHandler(entityUseCases.Group.DeleteGroup, &grouppb.DeleteGroupRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/group/list",
				Handler: contracts.NewGenericHandler(entityUseCases.Group.ListGroups, &grouppb.ListGroupsRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/group/get-list-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.Group.GetGroupListPageData, &grouppb.GetGroupListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/group/get-item-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.Group.GetGroupItemPageData, &grouppb.GetGroupItemPageDataRequest{}),
			},
		)
	}

	// GroupAttribute routes
	if entityUseCases.GroupAttribute != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/group-attribute/create",
				Handler: contracts.NewGenericHandler(entityUseCases.GroupAttribute.CreateGroupAttribute, &groupattributepb.CreateGroupAttributeRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/group-attribute/read",
				Handler: contracts.NewGenericHandler(entityUseCases.GroupAttribute.ReadGroupAttribute, &groupattributepb.ReadGroupAttributeRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/group-attribute/update",
				Handler: contracts.NewGenericHandler(entityUseCases.GroupAttribute.UpdateGroupAttribute, &groupattributepb.UpdateGroupAttributeRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/group-attribute/delete",
				Handler: contracts.NewGenericHandler(entityUseCases.GroupAttribute.DeleteGroupAttribute, &groupattributepb.DeleteGroupAttributeRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/group-attribute/list",
				Handler: contracts.NewGenericHandler(entityUseCases.GroupAttribute.ListGroupAttributes, &groupattributepb.ListGroupAttributesRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/group-attribute/get-list-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.GroupAttribute.GetGroupAttributeListPageData, &groupattributepb.GetGroupAttributeListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/group-attribute/get-item-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.GroupAttribute.GetGroupAttributeItemPageData, &groupattributepb.GetGroupAttributeItemPageDataRequest{}),
			},
		)
	}

	// Location routes
	if entityUseCases.Location != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/location/create",
				Handler: contracts.NewGenericHandler(entityUseCases.Location.CreateLocation, &locationpb.CreateLocationRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/location/read",
				Handler: contracts.NewGenericHandler(entityUseCases.Location.ReadLocation, &locationpb.ReadLocationRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/location/update",
				Handler: contracts.NewGenericHandler(entityUseCases.Location.UpdateLocation, &locationpb.UpdateLocationRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/location/delete",
				Handler: contracts.NewGenericHandler(entityUseCases.Location.DeleteLocation, &locationpb.DeleteLocationRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/location/list",
				Handler: contracts.NewGenericHandler(entityUseCases.Location.ListLocations, &locationpb.ListLocationsRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/location/get-list-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.Location.GetLocationListPageData, &locationpb.GetLocationListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/location/get-item-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.Location.GetLocationItemPageData, &locationpb.GetLocationItemPageDataRequest{}),
			},
		)
	}

	// LocationAttribute routes
	if entityUseCases.LocationAttribute != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/location-attribute/create",
				Handler: contracts.NewGenericHandler(entityUseCases.LocationAttribute.CreateLocationAttribute, &locationattributepb.CreateLocationAttributeRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/location-attribute/read",
				Handler: contracts.NewGenericHandler(entityUseCases.LocationAttribute.ReadLocationAttribute, &locationattributepb.ReadLocationAttributeRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/location-attribute/update",
				Handler: contracts.NewGenericHandler(entityUseCases.LocationAttribute.UpdateLocationAttribute, &locationattributepb.UpdateLocationAttributeRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/location-attribute/delete",
				Handler: contracts.NewGenericHandler(entityUseCases.LocationAttribute.DeleteLocationAttribute, &locationattributepb.DeleteLocationAttributeRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/location-attribute/list",
				Handler: contracts.NewGenericHandler(entityUseCases.LocationAttribute.ListLocationAttributes, &locationattributepb.ListLocationAttributesRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/location-attribute/get-list-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.LocationAttribute.GetLocationAttributeListPageData, &locationattributepb.GetLocationAttributeListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/location-attribute/get-item-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.LocationAttribute.GetLocationAttributeItemPageData, &locationattributepb.GetLocationAttributeItemPageDataRequest{}),
			},
		)
	}

	// Permission routes
	if entityUseCases.Permission != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/permission/create",
				Handler: contracts.NewGenericHandler(entityUseCases.Permission.CreatePermission, &permissionpb.CreatePermissionRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/permission/read",
				Handler: contracts.NewGenericHandler(entityUseCases.Permission.ReadPermission, &permissionpb.ReadPermissionRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/permission/update",
				Handler: contracts.NewGenericHandler(entityUseCases.Permission.UpdatePermission, &permissionpb.UpdatePermissionRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/permission/delete",
				Handler: contracts.NewGenericHandler(entityUseCases.Permission.DeletePermission, &permissionpb.DeletePermissionRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/permission/list",
				Handler: contracts.NewGenericHandler(entityUseCases.Permission.ListPermissions, &permissionpb.ListPermissionsRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/permission/get-list-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.Permission.GetPermissionListPageData, &permissionpb.GetPermissionListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/permission/get-item-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.Permission.GetPermissionItemPageData, &permissionpb.GetPermissionItemPageDataRequest{}),
			},
		)
	}

	// Role routes
	if entityUseCases.Role != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/role/create",
				Handler: contracts.NewGenericHandler(entityUseCases.Role.CreateRole, &rolepb.CreateRoleRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/role/read",
				Handler: contracts.NewGenericHandler(entityUseCases.Role.ReadRole, &rolepb.ReadRoleRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/role/update",
				Handler: contracts.NewGenericHandler(entityUseCases.Role.UpdateRole, &rolepb.UpdateRoleRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/role/delete",
				Handler: contracts.NewGenericHandler(entityUseCases.Role.DeleteRole, &rolepb.DeleteRoleRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/role/list",
				Handler: contracts.NewGenericHandler(entityUseCases.Role.ListRoles, &rolepb.ListRolesRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/role/get-list-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.Role.GetRoleListPageData, &rolepb.GetRoleListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/role/get-item-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.Role.GetRoleItemPageData, &rolepb.GetRoleItemPageDataRequest{}),
			},
		)
	}

	// RolePermission routes
	if entityUseCases.RolePermission != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/role-permission/create",
				Handler: contracts.NewGenericHandler(entityUseCases.RolePermission.CreateRolePermission, &rolepermissionpb.CreateRolePermissionRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/role-permission/read",
				Handler: contracts.NewGenericHandler(entityUseCases.RolePermission.ReadRolePermission, &rolepermissionpb.ReadRolePermissionRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/role-permission/update",
				Handler: contracts.NewGenericHandler(entityUseCases.RolePermission.UpdateRolePermission, &rolepermissionpb.UpdateRolePermissionRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/role-permission/delete",
				Handler: contracts.NewGenericHandler(entityUseCases.RolePermission.DeleteRolePermission, &rolepermissionpb.DeleteRolePermissionRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/role-permission/list",
				Handler: contracts.NewGenericHandler(entityUseCases.RolePermission.ListRolePermissions, &rolepermissionpb.ListRolePermissionsRequest{}),
			},
			// contracts.RouteConfiguration{
			// 	Method:  "POST",
			// 	Path:    "/api/entity/role-permission/get-list-page-data",
			// 	Handler: contracts.NewGenericHandler(entityUseCases.RolePermission.GetRolePermissionListPageData, &rolepermissionpb.GetRolePermissionListPageDataRequest{}),
			// },
			// contracts.RouteConfiguration{
			// 	Method:  "POST",
			// 	Path:    "/api/entity/role-permission/get-item-page-data",
			// 	Handler: contracts.NewGenericHandler(entityUseCases.RolePermission.GetRolePermissionItemPageData, &rolepermissionpb.GetRolePermissionItemPageDataRequest{}),
			// },
		)
	}

	// Staff routes
	if entityUseCases.Staff != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/staff/create",
				Handler: contracts.NewGenericHandler(entityUseCases.Staff.CreateStaff, &staffpb.CreateStaffRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/staff/read",
				Handler: contracts.NewGenericHandler(entityUseCases.Staff.ReadStaff, &staffpb.ReadStaffRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/staff/update",
				Handler: contracts.NewGenericHandler(entityUseCases.Staff.UpdateStaff, &staffpb.UpdateStaffRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/staff/delete",
				Handler: contracts.NewGenericHandler(entityUseCases.Staff.DeleteStaff, &staffpb.DeleteStaffRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/staff/list",
				Handler: contracts.NewGenericHandler(entityUseCases.Staff.ListStaffs, &staffpb.ListStaffsRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/staff/get-list-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.Staff.GetStaffListPageData, &staffpb.GetStaffListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/staff/get-item-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.Staff.GetStaffItemPageData, &staffpb.GetStaffItemPageDataRequest{}),
			},
		)
	}

	// StaffAttribute routes
	if entityUseCases.StaffAttribute != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/staff-attribute/create",
				Handler: contracts.NewGenericHandler(entityUseCases.StaffAttribute.CreateStaffAttribute, &staffattributepb.CreateStaffAttributeRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/staff-attribute/read",
				Handler: contracts.NewGenericHandler(entityUseCases.StaffAttribute.ReadStaffAttribute, &staffattributepb.ReadStaffAttributeRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/staff-attribute/update",
				Handler: contracts.NewGenericHandler(entityUseCases.StaffAttribute.UpdateStaffAttribute, &staffattributepb.UpdateStaffAttributeRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/staff-attribute/delete",
				Handler: contracts.NewGenericHandler(entityUseCases.StaffAttribute.DeleteStaffAttribute, &staffattributepb.DeleteStaffAttributeRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/staff-attribute/list",
				Handler: contracts.NewGenericHandler(entityUseCases.StaffAttribute.ListStaffAttributes, &staffattributepb.ListStaffAttributesRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/staff-attribute/get-list-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.StaffAttribute.GetStaffAttributeListPageData, &staffattributepb.GetStaffAttributeListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/staff-attribute/get-item-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.StaffAttribute.GetStaffAttributeItemPageData, &staffattributepb.GetStaffAttributeItemPageDataRequest{}),
			},
		)
	}

	// User routes (Primary Entity)
	if entityUseCases.User != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/user/create",
				Handler: contracts.NewGenericHandler(entityUseCases.User.CreateUser, &userpb.CreateUserRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/user/read",
				Handler: contracts.NewGenericHandler(entityUseCases.User.ReadUser, &userpb.ReadUserRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/user/update",
				Handler: contracts.NewGenericHandler(entityUseCases.User.UpdateUser, &userpb.UpdateUserRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/user/delete",
				Handler: contracts.NewGenericHandler(entityUseCases.User.DeleteUser, &userpb.DeleteUserRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/user/list",
				Handler: contracts.NewGenericHandler(entityUseCases.User.ListUsers, &userpb.ListUsersRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/user/get-list-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.User.GetUserListPageData, &userpb.GetUserListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/user/get-item-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.User.GetUserItemPageData, &userpb.GetUserItemPageDataRequest{}),
			},
		)
	}

	// Workspace routes (Primary Entity)
	if entityUseCases.Workspace != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/workspace/create",
				Handler: contracts.NewGenericHandler(entityUseCases.Workspace.CreateWorkspace, &workspacepb.CreateWorkspaceRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/workspace/read",
				Handler: contracts.NewGenericHandler(entityUseCases.Workspace.ReadWorkspace, &workspacepb.ReadWorkspaceRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/workspace/update",
				Handler: contracts.NewGenericHandler(entityUseCases.Workspace.UpdateWorkspace, &workspacepb.UpdateWorkspaceRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/workspace/delete",
				Handler: contracts.NewGenericHandler(entityUseCases.Workspace.DeleteWorkspace, &workspacepb.DeleteWorkspaceRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/workspace/list",
				Handler: contracts.NewGenericHandler(entityUseCases.Workspace.ListWorkspaces, &workspacepb.ListWorkspacesRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/workspace/get-list-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.Workspace.GetWorkspaceListPageData, &workspacepb.GetWorkspaceListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/workspace/get-item-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.Workspace.GetWorkspaceItemPageData, &workspacepb.GetWorkspaceItemPageDataRequest{}),
			},
		)
	}

	// WorkspaceUser routes
	if entityUseCases.WorkspaceUser != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/workspace-user/create",
				Handler: contracts.NewGenericHandler(entityUseCases.WorkspaceUser.CreateWorkspaceUser, &workspaceuserpb.CreateWorkspaceUserRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/workspace-user/read",
				Handler: contracts.NewGenericHandler(entityUseCases.WorkspaceUser.ReadWorkspaceUser, &workspaceuserpb.ReadWorkspaceUserRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/workspace-user/update",
				Handler: contracts.NewGenericHandler(entityUseCases.WorkspaceUser.UpdateWorkspaceUser, &workspaceuserpb.UpdateWorkspaceUserRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/workspace-user/delete",
				Handler: contracts.NewGenericHandler(entityUseCases.WorkspaceUser.DeleteWorkspaceUser, &workspaceuserpb.DeleteWorkspaceUserRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/workspace-user/list",
				Handler: contracts.NewGenericHandler(entityUseCases.WorkspaceUser.ListWorkspaceUsers, &workspaceuserpb.ListWorkspaceUsersRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/workspace-user/get-list-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.WorkspaceUser.GetWorkspaceUserListPageData, &workspaceuserpb.GetWorkspaceUserListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/workspace-user/get-item-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.WorkspaceUser.GetWorkspaceUserItemPageData, &workspaceuserpb.GetWorkspaceUserItemPageDataRequest{}),
			},
		)
	}

	// WorkspaceUserRole routes
	if entityUseCases.WorkspaceUserRole != nil {
		routes = append(routes,
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/workspace-user-role/create",
				Handler: contracts.NewGenericHandler(entityUseCases.WorkspaceUserRole.CreateWorkspaceUserRole, &workspaceuserrolepb.CreateWorkspaceUserRoleRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/workspace-user-role/read",
				Handler: contracts.NewGenericHandler(entityUseCases.WorkspaceUserRole.ReadWorkspaceUserRole, &workspaceuserrolepb.ReadWorkspaceUserRoleRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/workspace-user-role/update",
				Handler: contracts.NewGenericHandler(entityUseCases.WorkspaceUserRole.UpdateWorkspaceUserRole, &workspaceuserrolepb.UpdateWorkspaceUserRoleRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/workspace-user-role/delete",
				Handler: contracts.NewGenericHandler(entityUseCases.WorkspaceUserRole.DeleteWorkspaceUserRole, &workspaceuserrolepb.DeleteWorkspaceUserRoleRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/workspace-user-role/list",
				Handler: contracts.NewGenericHandler(entityUseCases.WorkspaceUserRole.ListWorkspaceUserRoles, &workspaceuserrolepb.ListWorkspaceUserRolesRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/workspace-user-role/get-list-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.WorkspaceUserRole.GetWorkspaceUserRoleListPageData, &workspaceuserrolepb.GetWorkspaceUserRoleListPageDataRequest{}),
			},
			contracts.RouteConfiguration{
				Method:  "POST",
				Path:    "/api/entity/workspace-user-role/get-item-page-data",
				Handler: contracts.NewGenericHandler(entityUseCases.WorkspaceUserRole.GetWorkspaceUserRoleItemPageData, &workspaceuserrolepb.GetWorkspaceUserRoleItemPageDataRequest{}),
			},
		)
	}

	return contracts.DomainRouteConfiguration{
		Domain:  "entity",
		Prefix:  "/entity",
		Enabled: true,
		Routes:  routes,
	}
}
