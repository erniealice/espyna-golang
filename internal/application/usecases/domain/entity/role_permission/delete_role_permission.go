package role_permission

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	securityports "github.com/erniealice/espyna-golang/internal/application/ports/security"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	clientportalgrantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_portal_grant"
	delegateclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_client"
	delegatesupplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_supplier"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
	rolepermissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role_permission"
	supplierportalgrantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_portal_grant"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

// DeleteRolePermissionRepositories groups all repository dependencies
type DeleteRolePermissionRepositories struct {
	RolePermission rolepermissionpb.RolePermissionDomainServiceServer // Primary entity repository
	Role           rolepb.RoleDomainServiceServer                     // Entity reference validation
	Permission     permissionpb.PermissionDomainServiceServer         // Entity reference validation
	// WorkspaceUserRole enumerates the bindings assigned this role so a revoke
	// can evict exactly their cached permission codes (P10/D2). Nil-safe.
	WorkspaceUserRole workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer
	// The remaining grant-table seams let a REVOKE's cache invalidation reach every
	// binding kind the loader caches, not just workspace_user_role (CF-1) — the
	// security-relevant direction (a revoked code must not linger for portal /
	// delegate / staff principals until the loader TTL). All nil-safe.
	WorkspaceUser       workspaceuserpb.WorkspaceUserDomainServiceServer
	ClientPortalGrant   clientportalgrantpb.ClientPortalGrantDomainServiceServer
	SupplierPortalGrant supplierportalgrantpb.SupplierPortalGrantDomainServiceServer
	DelegateClient      delegateclientpb.DelegateClientDomainServiceServer
	DelegateSupplier    delegatesupplierpb.DelegateSupplierDomainServiceServer
}

// DeleteRolePermissionServices groups all business service dependencies
type DeleteRolePermissionServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	// PermissionCacheInvalidator evicts the affected bindings' cached RBAC codes
	// after a revoke so it applies immediately (P10/D2). Nil-safe.
	PermissionCacheInvalidator securityports.PermissionCacheInvalidator
}

// DeleteRolePermissionUseCase handles the business logic for deleting role permissions
type DeleteRolePermissionUseCase struct {
	repositories DeleteRolePermissionRepositories
	services     DeleteRolePermissionServices
}

// NewDeleteRolePermissionUseCase creates use case with grouped dependencies
func NewDeleteRolePermissionUseCase(
	repositories DeleteRolePermissionRepositories,
	services DeleteRolePermissionServices,
) *DeleteRolePermissionUseCase {
	return &DeleteRolePermissionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteRolePermissionUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteRolePermissionUseCase with grouped parameters instead
func NewDeleteRolePermissionUseCaseUngrouped(
	rolePermissionRepo rolepermissionpb.RolePermissionDomainServiceServer,
	authorizationService ports.Authorizer,
) *DeleteRolePermissionUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteRolePermissionRepositories{
		RolePermission: rolePermissionRepo,
		Role:           nil, // Not needed for delete operations
		Permission:     nil, // Not needed for delete operations
	}

	services := DeleteRolePermissionServices{
		Authorizer:       authorizationService,
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewDeleteRolePermissionUseCase(repositories, services)
}

// Execute performs the delete role permission operation
func (uc *DeleteRolePermissionUseCase) Execute(ctx context.Context, req *rolepermissionpb.DeleteRolePermissionRequest) (*rolepermissionpb.DeleteRolePermissionResponse, error) {

	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.RolePermission,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.errors.input_validation_failed", "Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.errors.business_rule_validation_failed", "Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Resolve the grant's role_id BEFORE the delete so the cache eviction can
	// enumerate the affected bindings after the row is gone (P10/D2). Prefer a
	// role id already on the request; otherwise read the row by id.
	roleID := uc.resolveRoleID(ctx, req)

	// Call repository
	resp, err := uc.repositories.RolePermission.DeleteRolePermission(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.errors.deletion_failed", "Role-Permission deletion failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	log.Printf("AUTHZ_CHANGE | action=revoke | role_permission_id=%s", req.Data.Id)

	// Post-revoke: the removed permission changes the effective code set of every
	// binding assigned this role — evict exactly those bindings' cached codes so
	// the revoke is live on their next request (P10/D2). Covers EVERY binding kind
	// the loader caches (operator/staff/portal/delegate), not just
	// workspace_user_role (CF-1: a revoked code must not linger for those kinds).
	invalidateRoleBindings(ctx, roleBindingInvalidationRepos{
		WorkspaceUserRole:   uc.repositories.WorkspaceUserRole,
		WorkspaceUser:       uc.repositories.WorkspaceUser,
		ClientPortalGrant:   uc.repositories.ClientPortalGrant,
		SupplierPortalGrant: uc.repositories.SupplierPortalGrant,
		DelegateClient:      uc.repositories.DelegateClient,
		DelegateSupplier:    uc.repositories.DelegateSupplier,
	}, uc.services.PermissionCacheInvalidator, roleID)

	return resp, nil
}

// resolveRoleID returns the role_id of the grant being deleted, preferring a
// value already on the request and otherwise reading the row by id. Returns ""
// (=> no-op invalidation) when it cannot be resolved.
func (uc *DeleteRolePermissionUseCase) resolveRoleID(ctx context.Context, req *rolepermissionpb.DeleteRolePermissionRequest) string {
	if req == nil || req.Data == nil {
		return ""
	}
	if req.Data.RoleId != "" {
		return req.Data.RoleId
	}
	if uc.repositories.RolePermission == nil || req.Data.Id == "" {
		return ""
	}
	read, err := uc.repositories.RolePermission.ReadRolePermission(ctx, &rolepermissionpb.ReadRolePermissionRequest{
		Data: &rolepermissionpb.RolePermission{Id: req.Data.Id},
	})
	if err != nil || read == nil {
		return ""
	}
	for _, row := range read.GetData() {
		if row != nil && row.GetRoleId() != "" {
			return row.GetRoleId()
		}
	}
	return ""
}

// validateInput validates the input request
func (uc *DeleteRolePermissionUseCase) validateInput(ctx context.Context, req *rolepermissionpb.DeleteRolePermissionRequest) error {

	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.validation.request_required", "Request is required for role-permissions"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.validation.data_required", "Role-Permission data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.validation.id_required", "Role-Permission ID is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeleteRolePermissionUseCase) validateBusinessRules(ctx context.Context, req *rolepermissionpb.DeleteRolePermissionRequest) error {
	// TODO: Add business rules for role permission deletion
	// Example: Check if removing this permission would leave role without critical permissions
	// For now, allow all deletions

	return nil
}

// Helper functions
