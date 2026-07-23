package permission

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	securityports "github.com/erniealice/espyna-golang/internal/application/ports/security"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
)

// UpdatePermissionRepositories groups all repository dependencies
type UpdatePermissionRepositories struct {
	Permission permissionpb.PermissionDomainServiceServer // Primary entity repository
}

// UpdatePermissionServices groups all business service dependencies
type UpdatePermissionServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	// PermissionCacheInvalidator evicts the acting-provenance user's cached RBAC
	// codes after a definition write (P10/D2). Nil-safe (no-op when unset).
	PermissionCacheInvalidator securityports.PermissionCacheInvalidator
}

// UpdatePermissionUseCase handles the business logic for updating permissions
type UpdatePermissionUseCase struct {
	repositories UpdatePermissionRepositories
	services     UpdatePermissionServices
}

// NewUpdatePermissionUseCase creates use case with grouped dependencies
func NewUpdatePermissionUseCase(
	repositories UpdatePermissionRepositories,
	services UpdatePermissionServices,
) *UpdatePermissionUseCase {
	return &UpdatePermissionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdatePermissionUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdatePermissionUseCase with grouped parameters instead
func NewUpdatePermissionUseCaseUngrouped(permissionRepo permissionpb.PermissionDomainServiceServer) *UpdatePermissionUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdatePermissionRepositories{
		Permission: permissionRepo,
	}

	services := UpdatePermissionServices{
		Authorizer:       nil,
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewUpdatePermissionUseCase(repositories, services)
}

func (uc *UpdatePermissionUseCase) Execute(ctx context.Context, req *permissionpb.UpdatePermissionRequest) (*permissionpb.UpdatePermissionResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Permission,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	var resp *permissionpb.UpdatePermissionResponse
	var err error
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		resp, err = uc.executeWithTransaction(ctx, req)
	} else {
		// Fallback to non-transactional execution
		resp, err = uc.executeCore(ctx, req)
	}
	if err != nil {
		return nil, err
	}

	// Post-commit: evict the acting-provenance user's cached RBAC codes (P10/D2).
	// See CreatePermissionUseCase.Execute for the rationale — a permission
	// definition change is not a grant edge, so this is a conservative,
	// single-user, cheap safeguard, not a role-wide invalidation.
	if req != nil && req.Data != nil {
		uc.invalidateProvenanceCache(req.Data.UserId)
	}
	return resp, nil
}

// invalidateProvenanceCache drops the provenance user's cached permission codes.
// Nil-safe: no-op when the invalidator is unwired or the id is empty.
func (uc *UpdatePermissionUseCase) invalidateProvenanceCache(userID string) {
	if uc.services.PermissionCacheInvalidator == nil || userID == "" {
		return
	}
	uc.services.PermissionCacheInvalidator.InvalidateUser(userID)
}

// executeWithTransaction executes permission update within a transaction
func (uc *UpdatePermissionUseCase) executeWithTransaction(ctx context.Context, req *permissionpb.UpdatePermissionRequest) (*permissionpb.UpdatePermissionResponse, error) {
	var result *permissionpb.UpdatePermissionResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "permission.errors.update_failed", "Permission update failed [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic (moved from original Execute method)
func (uc *UpdatePermissionUseCase) executeCore(ctx context.Context, req *permissionpb.UpdatePermissionRequest) (*permissionpb.UpdatePermissionResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Defense-in-depth (CF-5): a body-supplied provenance id that disagrees with
	// the session principal is rejected. No-op when ctx has no resolved principal.
	if err := validatePrincipalProvenance(ctx, uc.services.Translator, req.Data); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichPermissionData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.Permission.UpdatePermission(ctx, req)
}

// validateInput validates the input request
func (uc *UpdatePermissionUseCase) validateInput(ctx context.Context, req *permissionpb.UpdatePermissionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "permission.validation.request_required", "Request is required for permissions [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "permission.validation.data_required", "Permission data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "permission.validation.id_required", "Permission ID is required [DEFAULT]"))
	}
	if req.Data.WorkspaceId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "permission.validation.workspace_id_required", "Workspace ID is required [DEFAULT]"))
	}
	if req.Data.UserId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "permission.validation.user_id_required", "User ID is required [DEFAULT]"))
	}
	if req.Data.GrantedByUserId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "permission.validation.granted_by_user_id_required", "Granted by user ID is required [DEFAULT]"))
	}
	if req.Data.PermissionCode == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "permission.validation.permission_code_required", "Permission code is required [DEFAULT]"))
	}
	return nil
}

// enrichPermissionData adds audit information for updates
func (uc *UpdatePermissionUseCase) enrichPermissionData(permission *permissionpb.Permission) error {
	now := time.Now()

	// Set permission audit fields for modification
	permission.DateModified = &[]int64{now.Unix()}[0]
	permission.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdatePermissionUseCase) validateBusinessRules(ctx context.Context, permission *permissionpb.Permission) error {
	// Validate permission code format
	if len(permission.PermissionCode) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "permission.validation.permission_code_too_short", "Permission code must be at least 3 characters long [DEFAULT]"))
	}

	if len(permission.PermissionCode) > 50 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "permission.validation.permission_code_too_long", "Permission code cannot exceed 50 characters [DEFAULT]"))
	}

	// Validate permission type
	if permission.PermissionType == permissionpb.PermissionType_PERMISSION_TYPE_UNSPECIFIED {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "permission.validation.permission_type_unspecified", "Permission type must be specified [DEFAULT]"))
	}

	// NOTE: No self-grant check. This entity is the permission-code DEFINITION row,
	// not a grant edge. UserId/GrantedByUserId are provenance bookkeeping only (every
	// copya seed row is self-provenanced to superadmin-001, and an admin editing a
	// definition through a single session principal ALWAYS has UserId == GrantedByUserId).
	// The actual grant vehicle is role_permission, and route authorization is the
	// Layer-2 permission:update gate — so a self-provenanced definition is valid.

	return nil
}
