package role_permission

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	securityports "github.com/erniealice/espyna-golang/internal/application/ports/security"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
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

// CreateRolePermissionRepositories groups all repository dependencies
type CreateRolePermissionRepositories struct {
	RolePermission rolepermissionpb.RolePermissionDomainServiceServer // Primary entity repository
	Role           rolepb.RoleDomainServiceServer                     // Entity reference validation
	Permission     permissionpb.PermissionDomainServiceServer         // Entity reference validation
	// WorkspaceUserRole enumerates the bindings assigned this role so a grant can
	// evict exactly their cached permission codes (P10/D2). Nil-safe.
	WorkspaceUserRole workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer
	// The remaining grant-table seams let a grant's cache invalidation reach every
	// binding kind the permission loader caches, not just workspace_user_role
	// (CF-1): WorkspaceUser resolves the staff (kind 7) login user; the portal and
	// delegate seams enumerate the CLIENT/SUPPLIER (portal + delegate) principals
	// holding this role. All nil-safe (absent => that chain is skipped).
	WorkspaceUser       workspaceuserpb.WorkspaceUserDomainServiceServer
	ClientPortalGrant   clientportalgrantpb.ClientPortalGrantDomainServiceServer
	SupplierPortalGrant supplierportalgrantpb.SupplierPortalGrantDomainServiceServer
	DelegateClient      delegateclientpb.DelegateClientDomainServiceServer
	DelegateSupplier    delegatesupplierpb.DelegateSupplierDomainServiceServer
}

// CreateRolePermissionServices groups all business service dependencies
type CreateRolePermissionServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator      ports.IDGenerator
	// PermissionCacheInvalidator evicts the affected bindings' cached RBAC codes
	// after a grant so it applies immediately (P10/D2). Nil-safe.
	PermissionCacheInvalidator securityports.PermissionCacheInvalidator
}

// CreateRolePermissionUseCase handles the business logic for creating role permissions
type CreateRolePermissionUseCase struct {
	repositories CreateRolePermissionRepositories
	services     CreateRolePermissionServices
}

// NewCreateRolePermissionUseCase creates use case with grouped dependencies
func NewCreateRolePermissionUseCase(
	repositories CreateRolePermissionRepositories,
	services CreateRolePermissionServices,
) *CreateRolePermissionUseCase {
	return &CreateRolePermissionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateRolePermissionUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateRolePermissionUseCase with grouped parameters instead
func NewCreateRolePermissionUseCaseUngrouped(
	rolePermissionRepo rolepermissionpb.RolePermissionDomainServiceServer,
	roleRepo rolepb.RoleDomainServiceServer,
	permissionRepo permissionpb.PermissionDomainServiceServer,
	authorizationService ports.Authorizer,
) *CreateRolePermissionUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateRolePermissionRepositories{
		RolePermission: rolePermissionRepo,
		Role:           roleRepo,
		Permission:     permissionRepo,
	}

	services := CreateRolePermissionServices{
		Authorizer:       authorizationService,
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
		IDGenerator:      ports.NewNoOpIDGenerator(),
	}

	return NewCreateRolePermissionUseCase(repositories, services)
}

func (uc *CreateRolePermissionUseCase) Execute(ctx context.Context, req *rolepermissionpb.CreateRolePermissionRequest) (*rolepermissionpb.CreateRolePermissionResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.RolePermission,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.errors.input_validation_failed", "Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichRolePermissionData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.errors.enrichment_failed", "Business logic enrichment failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.errors.business_rule_validation_failed", "Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.errors.reference_validation_failed", "Entity reference validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// C14 reactivate-on-create: "Remove" soft-deletes the junction (active=false),
	// but uq_role_permission_1 (role_id, permission_id) has NO active filter — so a
	// plain re-INSERT of a previously-removed pair collides on the still-present
	// inactive row and 422s (a removed grant becomes one-way via the UI). When the
	// SAME (role_id, permission_id) already exists soft-deleted, REACTIVATE it
	// (active=true + the requested permission_type) instead of inserting a
	// duplicate. Same class/fix as the proven sgpps assign upsert (C10): no schema
	// change, soft-delete/audit honored (the Update adapter preserves date_created
	// and re-stamps date_modified).
	resp, err := uc.createOrReactivate(ctx, req)
	if err != nil {
		return nil, err
	}

	log.Printf("AUTHZ_CHANGE | action=grant | role_permission_id=%s", req.Data.Id)

	// Post-grant: the new permission changes the effective code set of every
	// binding assigned this role — evict exactly those bindings' cached codes so
	// the grant is live on their next request, no TTL wait, no restart (P10/D2).
	// Covers EVERY binding kind the loader caches (operator/staff/portal/delegate),
	// not just workspace_user_role (CF-1).
	invalidateRoleBindings(ctx, uc.invalidationRepos(), uc.services.PermissionCacheInvalidator, req.Data.RoleId)

	return resp, nil
}

// invalidationRepos gathers the grant-table seams the cache invalidation
// enumerates. All fields are nil-safe downstream.
func (uc *CreateRolePermissionUseCase) invalidationRepos() roleBindingInvalidationRepos {
	return roleBindingInvalidationRepos{
		WorkspaceUserRole:   uc.repositories.WorkspaceUserRole,
		WorkspaceUser:       uc.repositories.WorkspaceUser,
		ClientPortalGrant:   uc.repositories.ClientPortalGrant,
		SupplierPortalGrant: uc.repositories.SupplierPortalGrant,
		DelegateClient:      uc.repositories.DelegateClient,
		DelegateSupplier:    uc.repositories.DelegateSupplier,
	}
}

// createOrReactivate writes the grant: it reactivates a soft-deleted
// (role_id, permission_id) row when one exists (C14), else plain-creates.
func (uc *CreateRolePermissionUseCase) createOrReactivate(ctx context.Context, req *rolepermissionpb.CreateRolePermissionRequest) (*rolepermissionpb.CreateRolePermissionResponse, error) {
	inactive, err := uc.findInactiveRolePermission(ctx, req.Data.RoleId, req.Data.PermissionId)
	if err != nil {
		return nil, err
	}
	if inactive != nil {
		updResp, uerr := uc.repositories.RolePermission.UpdateRolePermission(ctx, &rolepermissionpb.UpdateRolePermissionRequest{
			Data: &rolepermissionpb.RolePermission{
				Id:             inactive.GetId(),
				RoleId:         req.Data.RoleId,
				PermissionId:   req.Data.PermissionId,
				PermissionType: req.Data.PermissionType,
				Active:         true,
			},
		})
		if uerr != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.errors.creation_failed", "Role-Permission creation failed")
			return nil, fmt.Errorf("%s: %w", translatedError, uerr)
		}
		return reactivatedCreateResponse(updResp, inactive.GetId(), req.Data), nil
	}

	resp, cerr := uc.repositories.RolePermission.CreateRolePermission(ctx, req)
	if cerr != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.errors.creation_failed", "Role-Permission creation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, cerr)
	}
	return resp, nil
}

// findInactiveRolePermission returns the single SOFT-DELETED (active=false)
// role_permission for the EXACT (roleID, permissionID) pair, or nil when none.
// uq_role_permission_1 (role_id, permission_id) has no active filter, so at most
// one row exists per pair regardless of active state.
//
// The List MUST carry an explicit active=false BooleanFilter: PostgresOperations.
// List DEFAULTS to `active = true` unless the caller supplies an explicit
// "active" filter (contrib/postgres adapter/core/operations.go List), so a
// role_id-only query would silently exclude the soft-deleted row and this lookup
// would fall through to the plain INSERT that collides (the exact C14 422). With
// active=false the soft-deleted row is returned; the result is re-filtered in
// memory (permission match, active==false) as defense-in-depth against an adapter
// that ignores the server-side filter. Mirrors the proven sgpps findInactiveEdge.
func (uc *CreateRolePermissionUseCase) findInactiveRolePermission(ctx context.Context, roleID, permissionID string) (*rolepermissionpb.RolePermission, error) {
	if uc.repositories.RolePermission == nil || roleID == "" || permissionID == "" {
		return nil, nil
	}
	resp, err := uc.repositories.RolePermission.ListRolePermissions(ctx, &rolepermissionpb.ListRolePermissionsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field:      "role_id",
					FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{Value: roleID, Operator: commonpb.StringOperator_STRING_EQUALS}},
				},
				{
					Field:      "active",
					FilterType: &commonpb.TypedFilter_BooleanFilter{BooleanFilter: &commonpb.BooleanFilter{Value: false}},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	for _, row := range resp.GetData() {
		if row == nil || row.GetActive() {
			continue
		}
		if row.GetRoleId() != roleID || row.GetPermissionId() != permissionID {
			continue
		}
		return row, nil
	}
	return nil, nil
}

// reactivatedCreateResponse shapes a Create response from the reactivating
// Update. Falls back to a synthesized row when the adapter echoes no data.
func reactivatedCreateResponse(upd *rolepermissionpb.UpdateRolePermissionResponse, id string, src *rolepermissionpb.RolePermission) *rolepermissionpb.CreateRolePermissionResponse {
	data := upd.GetData()
	if len(data) == 0 {
		data = []*rolepermissionpb.RolePermission{{
			Id:             id,
			RoleId:         src.GetRoleId(),
			PermissionId:   src.GetPermissionId(),
			PermissionType: src.GetPermissionType(),
			Active:         true,
		}}
	}
	return &rolepermissionpb.CreateRolePermissionResponse{Success: true, Data: data}
}

// validateInput validates the input request
func (uc *CreateRolePermissionUseCase) validateInput(ctx context.Context, req *rolepermissionpb.CreateRolePermissionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.validation.request_required", "Request is required for role-permissions"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.validation.data_required", "Role-Permission data is required"))
	}
	if req.Data.RoleId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.validation.role_id_required", "Role ID is required"))
	}
	if req.Data.PermissionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.validation.permission_id_required", "Permission ID is required"))
	}
	return nil
}

// enrichRolePermissionData adds generated fields and audit information
func (uc *CreateRolePermissionUseCase) enrichRolePermissionData(rolePermission *rolepermissionpb.RolePermission) error {
	now := time.Now()

	// Generate RolePermission ID if not provided
	if rolePermission.Id == "" {
		rolePermission.Id = uc.services.IDGenerator.GenerateID()
	}

	// Set default permission type if not specified
	if rolePermission.PermissionType == permissionpb.PermissionType_PERMISSION_TYPE_UNSPECIFIED {
		rolePermission.PermissionType = permissionpb.PermissionType_PERMISSION_TYPE_ALLOW
	}

	// Set audit fields
	rolePermission.DateCreated = &[]int64{now.UnixMilli()}[0] // Milliseconds for consistency
	rolePermission.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	rolePermission.DateModified = &[]int64{now.UnixMilli()}[0] // Milliseconds for consistency
	rolePermission.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	rolePermission.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateRolePermissionUseCase) validateBusinessRules(ctx context.Context, rolePermission *rolepermissionpb.RolePermission) error {
	// Validate permission type
	if rolePermission.PermissionType == permissionpb.PermissionType_PERMISSION_TYPE_UNSPECIFIED {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.validation.permission_type_unspecified", "Permission type must be specified"))
	}

	// Validate role and permission relationship
	if rolePermission.RoleId == rolePermission.PermissionId {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.validation.same_id", "Role ID and permission ID cannot be the same"))
	}

	// Business rule: Prevent duplicate role-permission relationships
	// This validation should be checked at the repository level to ensure uniqueness
	// The repository implementation should check if a relationship already exists
	// between the role and permission before creating a new one

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateRolePermissionUseCase) validateEntityReferences(ctx context.Context, rolePermission *rolepermissionpb.RolePermission) error {
	// Validate Role entity reference
	if rolePermission.RoleId != "" {
		role, err := uc.repositories.Role.ReadRole(ctx, &rolepb.ReadRoleRequest{
			Data: &rolepb.Role{Id: rolePermission.RoleId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.errors.role_reference_validation_failed", "Failed to validate role entity reference")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if role == nil || !role.Success || role.Data == nil || len(role.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.errors.role_not_found", "Referenced role with ID '{roleId}' does not exist")
			translatedError = strings.ReplaceAll(translatedError, "{roleId}", rolePermission.RoleId)
			return errors.New(translatedError)
		}
		if !role.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.errors.role_not_active", "Referenced role with ID '{roleId}' is not active")
			translatedError = strings.ReplaceAll(translatedError, "{roleId}", rolePermission.RoleId)
			return errors.New(translatedError)
		}
	}

	// Validate Permission entity reference
	if rolePermission.PermissionId != "" {
		permission, err := uc.repositories.Permission.ReadPermission(ctx, &permissionpb.ReadPermissionRequest{
			Data: &permissionpb.Permission{Id: rolePermission.PermissionId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.errors.permission_reference_validation_failed", "Failed to validate permission entity reference")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if permission == nil || !permission.Success || permission.Data == nil || len(permission.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.errors.permission_not_found", "Referenced permission with ID '{permissionId}' does not exist")
			translatedError = strings.ReplaceAll(translatedError, "{permissionId}", rolePermission.PermissionId)
			return errors.New(translatedError)
		}
		if !permission.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "role_permission.errors.permission_not_active", "Referenced permission with ID '{permissionId}' is not active")
			translatedError = strings.ReplaceAll(translatedError, "{permissionId}", rolePermission.PermissionId)
			return errors.New(translatedError)
		}
	}

	return nil
}

// Helper functions
