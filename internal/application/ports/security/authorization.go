package security

import "context"

// AuthorizationService defines the interface for authorization operations
// This interface is framework-agnostic and resides in the application layer
type AuthorizationService interface {
	// HasPermission checks if a user has a specific permission
	HasPermission(ctx context.Context, userID, permission string) (bool, error)

	// HasGlobalPermission checks if a user has a global/system-wide permission
	// This is equivalent to HasPermission but with clearer naming for global scope
	HasGlobalPermission(ctx context.Context, userID, permission string) (bool, error)

	// HasPermissionInWorkspace checks if a user has a permission within a specific workspace
	HasPermissionInWorkspace(ctx context.Context, userID, workspaceID, permission string) (bool, error)

	// GetUserRoles returns all roles assigned to a user
	GetUserRoles(ctx context.Context, userID string) ([]string, error)

	// GetUserRolesInWorkspace returns user roles within a specific workspace
	GetUserRolesInWorkspace(ctx context.Context, userID, workspaceID string) ([]string, error)

	// GetUserWorkspaces returns all workspaces a user has access to
	GetUserWorkspaces(ctx context.Context, userID string) ([]string, error)

	// GetUserPermissionCodes returns all effective permission codes for a user (for UI adaptation).
	// Returns only ALLOW'd codes that are not overridden by a DENY.
	GetUserPermissionCodes(ctx context.Context, userID string) ([]string, error)

	// IsEnabled returns whether authorization is enabled
	IsEnabled() bool
}

// NoOpAuthorizationService provides a non-operational fallback that allows all actions.
type noOpAuthorizationService struct{}

func (s *noOpAuthorizationService) HasPermission(ctx context.Context, userID, permission string) (bool, error) {
	return true, nil
}
func (s *noOpAuthorizationService) HasGlobalPermission(ctx context.Context, userID, permission string) (bool, error) {
	return true, nil
}
func (s *noOpAuthorizationService) HasPermissionInWorkspace(ctx context.Context, userID, workspaceID, permission string) (bool, error) {
	return true, nil
}
func (s *noOpAuthorizationService) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	return []string{}, nil
}
func (s *noOpAuthorizationService) GetUserRolesInWorkspace(ctx context.Context, userID, workspaceID string) ([]string, error) {
	return []string{}, nil
}
func (s *noOpAuthorizationService) GetUserWorkspaces(ctx context.Context, userID string) ([]string, error) {
	return []string{}, nil
}
func (s *noOpAuthorizationService) GetUserPermissionCodes(ctx context.Context, userID string) ([]string, error) {
	return []string{}, nil
}
func (s *noOpAuthorizationService) IsEnabled() bool {
	return false
}

func NewNoOpAuthorizationService() AuthorizationService {
	return &noOpAuthorizationService{}
}

// AuthorizationProvider defines the interface for different authorization sources
type AuthorizationProvider interface {
	// Name returns the provider name (e.g., "jwt_claims", "database_rbac", "hybrid")
	Name() string

	// HasPermission checks permission using this provider's source
	HasPermission(ctx context.Context, userID, permission string) (bool, error)

	// HasGlobalPermission checks global permission using this provider's source
	HasGlobalPermission(ctx context.Context, userID, permission string) (bool, error)

	// HasPermissionInWorkspace checks workspace-specific permission
	HasPermissionInWorkspace(ctx context.Context, userID, workspaceID, permission string) (bool, error)

	// GetUserRoles retrieves user roles from this provider's source
	GetUserRoles(ctx context.Context, userID string) ([]string, error)

	// GetUserRolesInWorkspace retrieves workspace-specific user roles
	GetUserRolesInWorkspace(ctx context.Context, userID, workspaceID string) ([]string, error)

	// GetUserWorkspaces retrieves accessible workspaces
	GetUserWorkspaces(ctx context.Context, userID string) ([]string, error)

	// GetUserPermissionCodes returns all effective permission codes for a user
	GetUserPermissionCodes(ctx context.Context, userID string) ([]string, error)

	// IsEnabled returns whether this provider is enabled
	IsEnabled() bool

	// Initialize performs any required setup
	Initialize() error

	// Close performs cleanup
	Close() error
}

// Permission utility functions for dynamic permission generation
// These follow the patterns: "entity:action" and "workspace:entity:action"

// EntityPermission generates a permission string for an entity and action
func EntityPermission(entity, action string) string {
	return entity + ":" + action
}

// WorkspacePermission generates a workspace-specific permission string
func WorkspacePermission(workspace, entity, action string) string {
	return workspace + ":" + entity + ":" + action
}

// Common permission actions
const (
	ActionCreate = "create"
	ActionRead   = "read"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionList   = "list"
	ActionManage = "manage"
)

// Common entity types (matching the 40 entities in the system)
const (
	// Entity Domain (20 entities)
	EntityAdmin             = "admin"
	EntityClient            = "client"
	EntityClientAttribute   = "client_attribute"
	EntityDelegate          = "delegate"
	EntityDelegateAttribute = "delegate_attribute"
	EntityDelegateClient    = "delegate_client"
	EntityGroup             = "group"
	EntityGroupAttribute    = "group_attribute"
	EntityLocation          = "location"
	EntityLocationAttribute = "location_attribute"
	EntityManager           = "manager"
	EntityPermissions       = "permission"
	EntityRole              = "role"
	EntityRolePermission    = "role_permission"
	EntityStaff             = "staff"
	EntityStaffAttribute    = "staff_attribute"
	EntityUser              = "user"
	EntityWorkspace         = "workspace"
	EntityWorkspaceUser     = "workspace_user"
	EntityWorkspaceUserRole = "workspace_user_role"

	// Event Domain (4 entities)
	EntityEvent          = "event"
	EntityEventAttribute = "event_attribute"
	EntityEventClient    = "event_client"
	EntityEventProduct   = "event_product"

	// Framework Domain (3 entities)
	EntityFramework = "framework"
	EntityObjective = "objective"
	EntityTask      = "task"

	// Payment Domain (5 entities)
	EntityPayment                     = "payment"
	EntityPaymentAttribute            = "payment_attribute"
	EntityPaymentMethod               = "payment_method"
	EntityPaymentProfile              = "payment_profile"
	EntityPaymentProfilePaymentMethod = "payment_profile_payment_method"

	// Product Domain (10 entities)
	EntityCollection          = "collection"
	EntityCollectionAttribute = "collection_attribute"
	EntityCollectionPlan      = "collection_plan"
	EntityPriceList           = "price_list"
	EntityPriceProduct        = "price_product"
	EntityProduct             = "product"
	EntityProductAttribute    = "product_attribute"
	EntityProductCollection   = "product_collection"
	EntityProductPlan         = "product_plan"
	EntityResource            = "resource"

	// Record Domain (1 entity)
	EntityRecord = "record"

	// Subscription Domain (12 entities)
	EntityBalance               = "balance"
	EntityBalanceAttribute      = "balance_attribute"
	EntityInvoice               = "invoice"
	EntityInvoiceAttribute      = "invoice_attribute"
	EntityLicense               = "license"
	EntityLicenseHistory        = "license_history"
	EntityPlan                  = "plan"
	EntityPlanAttribute         = "plan_attribute"
	EntityPlanSettings          = "plan_settings"
	EntityPricePlan             = "price_plan"
	EntitySubscription          = "subscription"
	EntitySubscriptionAttribute = "subscription_attribute"
)
