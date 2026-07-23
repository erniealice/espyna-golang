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

// CreatePermissionRepositories groups all repository dependencies
type CreatePermissionRepositories struct {
	Permission permissionpb.PermissionDomainServiceServer // Primary entity repository
}

// CreatePermissionServices groups all business service dependencies
type CreatePermissionServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator      ports.IDGenerator
	// PermissionCacheInvalidator evicts the acting-provenance user's cached RBAC
	// codes after a definition write (P10/D2). Nil-safe (no-op when unset).
	PermissionCacheInvalidator securityports.PermissionCacheInvalidator
}

// CreatePermissionUseCase handles the business logic for creating permissions
type CreatePermissionUseCase struct {
	repositories CreatePermissionRepositories
	services     CreatePermissionServices
}

// NewCreatePermissionUseCase creates use case with grouped dependencies
func NewCreatePermissionUseCase(
	repositories CreatePermissionRepositories,
	services CreatePermissionServices,
) *CreatePermissionUseCase {
	return &CreatePermissionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreatePermissionUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreatePermissionUseCase with grouped parameters instead
func NewCreatePermissionUseCaseUngrouped(permissionRepo permissionpb.PermissionDomainServiceServer) *CreatePermissionUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreatePermissionRepositories{
		Permission: permissionRepo,
	}

	services := CreatePermissionServices{
		Authorizer:       nil,
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
		IDGenerator:      ports.NewNoOpIDGenerator(),
	}

	return NewCreatePermissionUseCase(repositories, services)
}

func (uc *CreatePermissionUseCase) Execute(ctx context.Context, req *permissionpb.CreatePermissionRequest) (*permissionpb.CreatePermissionResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Permission,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	var resp *permissionpb.CreatePermissionResponse
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

	// Post-commit: evict the acting-provenance user's cached RBAC codes so a
	// permission definition change is not masked by the loader's TTL (P10/D2).
	// A permission definition is not itself a grant edge, so this does NOT
	// retroactively alter other users' effective codes; the eviction is a
	// conservative, single-user, cheap safeguard (no enumeration). Any effect a
	// deactivated permission has on role holders remains bounded by the loader
	// TTL until the next role_permission/binding change — documented residual.
	if req != nil && req.Data != nil {
		uc.invalidateProvenanceCache(req.Data.UserId)
	}
	return resp, nil
}

// invalidateProvenanceCache drops the provenance user's cached permission codes.
// Nil-safe: no-op when the invalidator is unwired or the id is empty.
func (uc *CreatePermissionUseCase) invalidateProvenanceCache(userID string) {
	if uc.services.PermissionCacheInvalidator == nil || userID == "" {
		return
	}
	uc.services.PermissionCacheInvalidator.InvalidateUser(userID)
}

// executeWithTransaction executes permission creation within a transaction
func (uc *CreatePermissionUseCase) executeWithTransaction(ctx context.Context, req *permissionpb.CreatePermissionRequest) (*permissionpb.CreatePermissionResponse, error) {
	var result *permissionpb.CreatePermissionResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "permission.errors.creation_failed", "Permission creation failed [DEFAULT]")
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
func (uc *CreatePermissionUseCase) executeCore(ctx context.Context, req *permissionpb.CreatePermissionRequest) (*permissionpb.CreatePermissionResponse, error) {
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
	return uc.repositories.Permission.CreatePermission(ctx, req)
}

// validateInput validates the input request
func (uc *CreatePermissionUseCase) validateInput(ctx context.Context, req *permissionpb.CreatePermissionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "permission.validation.request_required", "Request is required for permissions [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "permission.validation.data_required", "Permission data is required [DEFAULT]"))
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

// enrichPermissionData adds generated fields and audit information
func (uc *CreatePermissionUseCase) enrichPermissionData(permission *permissionpb.Permission) error {
	now := time.Now()

	// Generate Permission ID if not provided
	if permission.Id == "" {
		permission.Id = uc.services.IDGenerator.GenerateID()
	}

	// Set default permission type if not specified
	if permission.PermissionType == permissionpb.PermissionType_PERMISSION_TYPE_UNSPECIFIED {
		permission.PermissionType = permissionpb.PermissionType_PERMISSION_TYPE_ALLOW
	}

	// Set permission audit fields
	permission.DateCreated = &[]int64{now.Unix()}[0]
	permission.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	permission.DateModified = &[]int64{now.Unix()}[0]
	permission.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	permission.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreatePermissionUseCase) validateBusinessRules(ctx context.Context, permission *permissionpb.Permission) error {
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
	// copya seed row is self-provenanced to superadmin-001, and an admin creating a
	// definition through a single session principal ALWAYS has UserId == GrantedByUserId).
	// The actual grant vehicle is role_permission, and route authorization is the
	// Layer-2 permission:create gate — so a self-provenanced definition is valid.

	return nil
}
