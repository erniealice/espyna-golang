package initializers

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/entity"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"

	// Dashboard use cases
	admindashboard "github.com/erniealice/espyna-golang/internal/application/usecases/entity/admin/dashboard"
	locationdashboard "github.com/erniealice/espyna-golang/internal/application/usecases/entity/location/dashboard"

	// Entity sub-domain use cases
	adminUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/admin"
	clientUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/client"
	clientAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/client_attribute"
	clientCategoryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/client_category"
	clientPortalGrantUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/client_portal_grant"
	delegateUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/delegate"
	delegateAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/delegate_attribute"
	delegateClientUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/delegate_client"
	delegateSupplierUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/delegate_supplier"
	groupUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/group"
	groupAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/group_attribute"
	locationUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/location"
	locationAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/location_attribute"
	permissionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/permission"
	roleUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/role"
	rolePermissionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/role_permission"
	staffUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/staff"
	staffAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/staff_attribute"
	supplierUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/supplier"
	supplierAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/supplier_attribute"
	supplierCategoryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/supplier_category"
	supplierPortalGrantUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/supplier_portal_grant"
	userUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/user"
	userPreferenceUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/user_preference"
	workspaceUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/workspace"
	workspaceUserUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/workspace_user"
	workspaceUserRoleUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/workspace_user_role"
)

// InitializeEntity creates all entity use cases from provider repositories.
// Use cases are only created when their primary repository is available (graceful degradation).
// This is composition logic - it wires infrastructure (providers) to application (use cases)
func InitializeEntity(
	repos *domain.EntityRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*entity.EntityUseCases, error) {
	svc := func() entityServices {
		return entityServices{authSvc, txSvc, i18nSvc, idSvc}
	}

	result := &entity.EntityUseCases{}

	if repos.Admin != nil {
		result.Admin = adminUseCases.NewUseCases(
			adminUseCases.AdminRepositories{Admin: repos.Admin},
			adminUseCases.AdminServices(svc()),
		)
	}

	if repos.Client != nil {
		result.Client = clientUseCases.NewUseCases(
			clientUseCases.ClientRepositories{
				Client: repos.Client,
				User:   repos.User,
			},
			clientUseCases.ClientServices(svc()),
		)
	}

	if repos.ClientAttribute != nil {
		result.ClientAttribute = clientAttributeUseCases.NewUseCases(
			clientAttributeUseCases.ClientAttributeRepositories{
				ClientAttribute: repos.ClientAttribute,
				Client:          repos.Client,
				Attribute:       repos.Attribute,
			},
			clientAttributeUseCases.ClientAttributeServices(svc()),
		)
	}

	if repos.ClientCategory != nil {
		result.ClientCategory = clientCategoryUseCases.NewUseCases(
			clientCategoryUseCases.ClientCategoryRepositories{
				ClientCategory: repos.ClientCategory,
			},
			clientCategoryUseCases.ClientCategoryServices(svc()),
		)
	}

	if repos.ClientPortalGrant != nil {
		s := svc()
		result.ClientPortalGrant = clientPortalGrantUseCases.NewUseCases(
			clientPortalGrantUseCases.ClientPortalGrantRepositories{ClientPortalGrant: repos.ClientPortalGrant},
			clientPortalGrantUseCases.ClientPortalGrantServices{
				AuthorizationService: s.AuthorizationService,
				TransactionService:   s.TransactionService,
				TranslationService:   s.TranslationService,
				IDService:            s.IDService,
			},
		)
	}

	if repos.Delegate != nil {
		result.Delegate = delegateUseCases.NewUseCases(
			delegateUseCases.DelegateRepositories{Delegate: repos.Delegate},
			delegateUseCases.DelegateServices(svc()),
		)
	}

	if repos.DelegateAttribute != nil {
		result.DelegateAttribute = delegateAttributeUseCases.NewUseCases(
			delegateAttributeUseCases.DelegateAttributeRepositories{
				DelegateAttribute: repos.DelegateAttribute,
				Delegate:          repos.Delegate,
				Attribute:         repos.Attribute,
			},
			delegateAttributeUseCases.DelegateAttributeServices(svc()),
		)
	}

	if repos.DelegateClient != nil {
		result.DelegateClient = delegateClientUseCases.NewUseCases(
			delegateClientUseCases.DelegateClientRepositories{
				DelegateClient: repos.DelegateClient,
				Delegate:       repos.Delegate,
				Client:         repos.Client,
			},
			delegateClientUseCases.DelegateClientServices(svc()),
		)
	}

	if repos.DelegateSupplier != nil {
		s := svc()
		result.DelegateSupplier = delegateSupplierUseCases.NewUseCases(
			delegateSupplierUseCases.DelegateSupplierRepositories{DelegateSupplier: repos.DelegateSupplier},
			delegateSupplierUseCases.DelegateSupplierServices{
				AuthorizationService: s.AuthorizationService,
				TransactionService:   s.TransactionService,
				TranslationService:   s.TranslationService,
				IDService:            s.IDService,
			},
		)
	}

	if repos.Group != nil {
		result.Group = groupUseCases.NewUseCases(
			groupUseCases.GroupRepositories{Group: repos.Group},
			groupUseCases.GroupServices(svc()),
		)
	}

	if repos.GroupAttribute != nil {
		result.GroupAttribute = groupAttributeUseCases.NewUseCases(
			groupAttributeUseCases.GroupAttributeRepositories{
				GroupAttribute: repos.GroupAttribute,
				Group:          repos.Group,
				Attribute:      repos.Attribute,
			},
			groupAttributeUseCases.GroupAttributeServices(svc()),
		)
	}

	if repos.Location != nil {
		result.Location = locationUseCases.NewUseCases(
			locationUseCases.LocationRepositories{Location: repos.Location},
			locationUseCases.LocationServices(svc()),
		)
	}

	if repos.LocationAttribute != nil {
		result.LocationAttribute = locationAttributeUseCases.NewUseCases(
			locationAttributeUseCases.LocationAttributeRepositories{
				LocationAttribute: repos.LocationAttribute,
				Location:          repos.Location,
				Attribute:         repos.Attribute,
			},
			locationAttributeUseCases.LocationAttributeServices(svc()),
		)
	}

	if repos.Permission != nil {
		result.Permission = permissionUseCases.NewUseCases(
			permissionUseCases.PermissionRepositories{Permission: repos.Permission},
			permissionUseCases.PermissionServices(svc()),
		)
	}

	if repos.Role != nil {
		result.Role = roleUseCases.NewUseCases(
			roleUseCases.RoleRepositories{Role: repos.Role},
			roleUseCases.RoleServices(svc()),
		)
	}

	if repos.RolePermission != nil {
		result.RolePermission = rolePermissionUseCases.NewUseCases(
			rolePermissionUseCases.RolePermissionRepositories{
				RolePermission: repos.RolePermission,
				Role:           repos.Role,
				Permission:     repos.Permission,
			},
			rolePermissionUseCases.RolePermissionServices(svc()),
		)
	}

	if repos.Staff != nil {
		result.Staff = staffUseCases.NewUseCases(
			staffUseCases.StaffRepositories{Staff: repos.Staff},
			staffUseCases.StaffServices(svc()),
		)
	}

	if repos.StaffAttribute != nil {
		result.StaffAttribute = staffAttributeUseCases.NewUseCases(
			staffAttributeUseCases.StaffAttributeRepositories{
				StaffAttribute: repos.StaffAttribute,
				Staff:          repos.Staff,
				Attribute:      repos.Attribute,
			},
			staffAttributeUseCases.StaffAttributeServices(svc()),
		)
	}

	if repos.Supplier != nil {
		result.Supplier = supplierUseCases.NewUseCases(
			supplierUseCases.SupplierRepositories{
				Supplier: repos.Supplier,
				User:     repos.User,
			},
			supplierUseCases.SupplierServices(svc()),
		)
	}

	if repos.SupplierAttribute != nil {
		result.SupplierAttribute = supplierAttributeUseCases.NewUseCases(
			supplierAttributeUseCases.SupplierAttributeRepositories{
				SupplierAttribute: repos.SupplierAttribute,
				Supplier:          repos.Supplier,
				Attribute:         repos.Attribute,
			},
			supplierAttributeUseCases.SupplierAttributeServices(svc()),
		)
	}

	if repos.SupplierCategory != nil {
		result.SupplierCategory = supplierCategoryUseCases.NewUseCases(
			supplierCategoryUseCases.SupplierCategoryRepositories{
				SupplierCategory: repos.SupplierCategory,
			},
			supplierCategoryUseCases.SupplierCategoryServices(svc()),
		)
	}

	if repos.SupplierPortalGrant != nil {
		s := svc()
		result.SupplierPortalGrant = supplierPortalGrantUseCases.NewUseCases(
			supplierPortalGrantUseCases.SupplierPortalGrantRepositories{SupplierPortalGrant: repos.SupplierPortalGrant},
			supplierPortalGrantUseCases.SupplierPortalGrantServices{
				AuthorizationService: s.AuthorizationService,
				TransactionService:   s.TransactionService,
				TranslationService:   s.TranslationService,
				IDService:            s.IDService,
			},
		)
	}

	if repos.User != nil {
		result.User = userUseCases.NewUseCases(
			userUseCases.UserRepositories{User: repos.User},
			userUseCases.UserServices(svc()),
		)
	}

	if repos.UserPreference != nil {
		s := svc()
		result.UserPreference = userPreferenceUseCases.NewUseCases(
			userPreferenceUseCases.UserPreferenceRepositories{UserPreference: repos.UserPreference},
			userPreferenceUseCases.UserPreferenceServices{
				AuthorizationService: s.AuthorizationService,
				TransactionService:   s.TransactionService,
				TranslationService:   s.TranslationService,
				IDService:            s.IDService,
			},
		)
	}

	if repos.Workspace != nil {
		result.Workspace = workspaceUseCases.NewUseCases(
			workspaceUseCases.WorkspaceRepositories{Workspace: repos.Workspace},
			workspaceUseCases.WorkspaceServices(svc()),
		)
	}

	if repos.WorkspaceUser != nil {
		result.WorkspaceUser = workspaceUserUseCases.NewUseCases(
			workspaceUserUseCases.WorkspaceUserRepositories{
				WorkspaceUser: repos.WorkspaceUser,
				Workspace:     repos.Workspace,
				User:          repos.User,
			},
			workspaceUserUseCases.WorkspaceUserServices(svc()),
		)
	}

	if repos.WorkspaceUserRole != nil {
		result.WorkspaceUserRole = workspaceUserRoleUseCases.NewUseCases(
			workspaceUserRoleUseCases.WorkspaceUserRoleRepositories{
				WorkspaceUserRole: repos.WorkspaceUserRole,
				WorkspaceUser:     repos.WorkspaceUser,
				Role:              repos.Role,
			},
			workspaceUserRoleUseCases.WorkspaceUserRoleServices(svc()),
		)
	}

	// Wire location dashboard via type assertions on location + location_area repos.
	if repos.Location != nil {
		locQ, lOK := repos.Location.(locationdashboard.LocationDashboardRepository)
		if lOK {
			var areaQ locationdashboard.LocationAreaDashboardRepository
			if repos.LocationArea != nil {
				if aq, ok := repos.LocationArea.(locationdashboard.LocationAreaDashboardRepository); ok {
					areaQ = aq
				}
			}
			result.LocationDashboard = locationdashboard.NewGetLocationDashboardPageDataUseCase(locQ, areaQ)
		}
	}

	// Wire admin dashboard via type assertions on permission/role/workspace_user/workspace_user_role repos.
	if repos.Permission != nil && repos.Role != nil && repos.WorkspaceUser != nil && repos.WorkspaceUserRole != nil {
		permQ, p1 := repos.Permission.(admindashboard.PermissionDashboardRepository)
		roleQ, p2 := repos.Role.(admindashboard.RoleDashboardRepository)
		wuQ, p3 := repos.WorkspaceUser.(admindashboard.WorkspaceUserDashboardRepository)
		wurQ, p4 := repos.WorkspaceUserRole.(admindashboard.WorkspaceUserRoleDashboardRepository)
		if p1 && p2 && p3 && p4 {
			result.AdminDashboard = admindashboard.NewGetAdminDashboardPageDataUseCase(permQ, roleQ, wuQ, wurQ)
		}
	}

	return result, nil
}

// entityServices is a type alias to reduce repetition in InitializeEntity.
// All entity *Services structs share the same layout, so we can convert between them.
type entityServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}
