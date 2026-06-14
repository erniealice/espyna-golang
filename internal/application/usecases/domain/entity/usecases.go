package entity

import (
	// Entity use cases
	adminUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/admin"
	clientUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/client"
	clientAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/client_attribute"
	clientCategoryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/client_category"
	clientPortalGrantUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/client_portal_grant"
	clientWorkspaceUserUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/client_workspace_user"
	delegateUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/delegate"
	delegateAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/delegate_attribute"
	delegateClientUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/delegate_client"
	delegateSupplierUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/delegate_supplier"
	groupUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/group"
	groupAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/group_attribute"
	locationUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/location"
	locationAreaUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/location_area"
	locationAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/location_attribute"
	paymentTermUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity/payment_term"
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
	// Note: Protobuf imports removed as domain-level constructors are no longer used
	// Dashboard use cases
	// Note: AdminDashboard relocated to service/dashboard/admin per
	// docs/plan/20260520-service-domain-migration §P1.C.1 (Q-SDM-DASHBOARD-LAYOUT).
	// Note: LocationDashboard relocated to service/dashboard/location per
	// docs/plan/20260520-service-domain-migration §P1.C.2 (Q-SDM-DASHBOARD-LAYOUT).
)

// EntityUseCases contains all entity-related use cases
type EntityUseCases struct {
	Admin               *adminUseCases.UseCases
	Client              *clientUseCases.UseCases
	ClientAttribute     *clientAttributeUseCases.UseCases
	ClientCategory      *clientCategoryUseCases.UseCases
	ClientPortalGrant   *clientPortalGrantUseCases.UseCases
	Delegate            *delegateUseCases.UseCases
	DelegateAttribute   *delegateAttributeUseCases.UseCases
	DelegateClient      *delegateClientUseCases.UseCases
	DelegateSupplier    *delegateSupplierUseCases.UseCases
	Group               *groupUseCases.UseCases
	GroupAttribute      *groupAttributeUseCases.UseCases
	Location            *locationUseCases.UseCases
	LocationArea        *locationAreaUseCases.UseCases
	LocationAttribute   *locationAttributeUseCases.UseCases
	PaymentTerm         *paymentTermUseCases.UseCases
	Permission          *permissionUseCases.UseCases
	Role                *roleUseCases.UseCases
	RolePermission      *rolePermissionUseCases.UseCases
	Staff               *staffUseCases.UseCases
	StaffAttribute      *staffAttributeUseCases.UseCases
	Supplier            *supplierUseCases.UseCases
	SupplierAttribute   *supplierAttributeUseCases.UseCases
	SupplierCategory    *supplierCategoryUseCases.UseCases
	SupplierPortalGrant *supplierPortalGrantUseCases.UseCases
	User                *userUseCases.UseCases
	UserPreference      *userPreferenceUseCases.UseCases
	Workspace           *workspaceUseCases.UseCases
	WorkspaceUser       *workspaceUserUseCases.UseCases
	WorkspaceUserRole   *workspaceUserRoleUseCases.UseCases
	// Outsourcing-vertical client account-team membership
	ClientWorkspaceUser *clientWorkspaceUserUseCases.UseCases

	// Dashboard use cases retired to service-driven layer:
	//   - AdminDashboard → service.Dashboard.Admin (Wave B P1.C.1)
	//   - LocationDashboard → service.Dashboard.Location (Wave B P1.C.2)
	// per docs/plan/20260520-service-domain-migration §P1.C, Q-SDM-DASHBOARD-LAYOUT.
}

// Note: Domain-level constructors are no longer needed with the new architecture.
// Individual use cases are now constructed directly in the repository factory
// with explicit entity reference dependency injection.
