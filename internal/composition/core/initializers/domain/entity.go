package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"

	// Dashboard use cases
	// Note: admin dashboard relocated to service/dashboard/admin per Wave B P1.C.1
	// (docs/plan/20260520-service-domain-migration §P1.C.1). The admin dashboard
	// repository wiring lives in initializers/service.go now.
	// Note: location dashboard relocated to service/dashboard/location per Wave B P1.C.2
	// (docs/plan/20260520-service-domain-migration §P1.C.2). The location dashboard
	// repository wiring lives in initializers/service.go now.

	// Entity sub-domain use cases
	adminUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/admin"
	clientUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/client"
	clientAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/client_attribute"
	clientCategoryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/client_category"
	clientPortalGrantUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/client_portal_grant"
	delegateUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/delegate"
	delegateAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/delegate_attribute"
	delegateClientUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/delegate_client"
	delegateSupplierUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/delegate_supplier"
	groupUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/group"
	groupAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/group_attribute"
	locationUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/location"
	locationAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/location_attribute"
	permissionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/permission"
	roleUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/role"
	rolePermissionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/role_permission"
	staffUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/staff"
	staffAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/staff_attribute"
	supplierUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/supplier"
	supplierAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/supplier_attribute"
	supplierCategoryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/supplier_category"
	supplierPortalGrantUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/supplier_portal_grant"
	userUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/user"
	userPreferenceUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/user_preference"
	workspaceUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/workspace"
	workspaceUserUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/workspace_user"
	workspaceUserRoleUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/workspace_user_role"
)

// InitializeEntity creates all entity use cases from provider repositories.
// Use cases are only created when their primary repository is available (graceful degradation).
// This is composition logic - it wires infrastructure (providers) to application (use cases)
func InitializeEntity(
	repos *domain.EntityRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idSvc ports.IDGenerator,
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
				Authorizer:  s.Authorizer,
				Transactor:  s.Transactor,
				Translator:  s.Translator,
				IDGenerator: s.IDGenerator,
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
				Authorizer:  s.Authorizer,
				Transactor:  s.Transactor,
				Translator:  s.Translator,
				IDGenerator: s.IDGenerator,
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
				Authorizer:  s.Authorizer,
				Transactor:  s.Transactor,
				Translator:  s.Translator,
				IDGenerator: s.IDGenerator,
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
				Authorizer:  s.Authorizer,
				Transactor:  s.Transactor,
				Translator:  s.Translator,
				IDGenerator: s.IDGenerator,
			},
		)
	}

	if repos.Workspace != nil {
		// WorkspaceServices carries an additional ReservedSlugs port (Phase P-1
		// of 20260521-workspace-keyed-routing) that is not part of the shared
		// entityServices shape. Wire the shared fields explicitly and leave
		// ReservedSlugs nil — service-admin composition will pass a non-nil
		// provider via its own construction path when wiring the ValidateSlug
		// use case for HTTP routes. Nil here disables only the reserved-word
		// check; format/length checks still run.
		s := svc()
		result.Workspace = workspaceUseCases.NewUseCases(
			workspaceUseCases.WorkspaceRepositories{Workspace: repos.Workspace},
			workspaceUseCases.WorkspaceServices{
				Authorizer:  s.Authorizer,
				Transactor:  s.Transactor,
				Translator:  s.Translator,
				IDGenerator: s.IDGenerator,
			},
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

	// Admin dashboard wiring relocated to initializers/service.go
	// (Wave B P1.C.1 — Q-SDM-DASHBOARD-LAYOUT). The cross-entity repository
	// type assertions live on service.Dashboard.Admin's Deps now.
	//
	// Location dashboard wiring relocated to initializers/service.go
	// (Wave B P1.C.2 — Q-SDM-DASHBOARD-LAYOUT). The cross-entity repository
	// type assertions live on service.Dashboard.Location's Deps now.

	return result, nil
}

// entityServices is a type alias to reduce repetition in InitializeEntity.
// All entity *Services structs share the same layout, so we can convert between them.
type entityServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}
