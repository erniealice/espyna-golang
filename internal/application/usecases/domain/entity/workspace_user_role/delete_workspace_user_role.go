package workspace_user_role

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
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

// DeleteWorkspaceUserRoleRepositories groups all repository dependencies
type DeleteWorkspaceUserRoleRepositories struct {
	WorkspaceUserRole workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer // Primary entity repository
	WorkspaceUser     workspaceuserpb.WorkspaceUserDomainServiceServer         // Entity reference validation
	Role              rolepb.RoleDomainServiceServer                           // Entity reference validation
}

// DeleteWorkspaceUserRoleServices groups all business service dependencies
type DeleteWorkspaceUserRoleServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	// PermissionCacheInvalidator evicts the unassigned binding's cached RBAC
	// codes so revoking a role from a user takes effect on the next request
	// (P10/D2). Nil-safe (no-op when unset).
	PermissionCacheInvalidator securityports.PermissionCacheInvalidator
}

// DeleteWorkspaceUserRoleUseCase handles the business logic for deleting a workspace user role
type DeleteWorkspaceUserRoleUseCase struct {
	repositories DeleteWorkspaceUserRoleRepositories
	services     DeleteWorkspaceUserRoleServices
}

// NewDeleteWorkspaceUserRoleUseCase creates use case with grouped dependencies
func NewDeleteWorkspaceUserRoleUseCase(
	repositories DeleteWorkspaceUserRoleRepositories,
	services DeleteWorkspaceUserRoleServices,
) *DeleteWorkspaceUserRoleUseCase {
	return &DeleteWorkspaceUserRoleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete workspace user role operation
func (uc *DeleteWorkspaceUserRoleUseCase) Execute(ctx context.Context, req *workspaceuserrolepb.DeleteWorkspaceUserRoleRequest) (*workspaceuserrolepb.DeleteWorkspaceUserRoleResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.WorkspaceUserRole,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	// Resolve the binding id (workspace_user_id) BEFORE the delete so the cache
	// eviction can run post-success even though the row is gone (P10/D2). Prefer
	// the value already on the request; otherwise read the row by id.
	bindingID := uc.resolveBindingID(ctx, req)

	// Check if transaction service is available and supports transactions
	var resp *workspaceuserrolepb.DeleteWorkspaceUserRoleResponse
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

	// Post-commit: revoking a role from a binding changes its effective
	// permission set — evict its cached codes so the revoke is live on the next
	// request (P10/D2). Precise: only this workspace_user's binding entries drop.
	uc.invalidateBindingCache(bindingID)
	return resp, nil
}

// resolveBindingID returns the workspace_user_id for the WUR row being deleted,
// preferring a value already on the request and otherwise reading the row by id.
// Returns "" (=> no-op invalidation) when it cannot be resolved.
func (uc *DeleteWorkspaceUserRoleUseCase) resolveBindingID(ctx context.Context, req *workspaceuserrolepb.DeleteWorkspaceUserRoleRequest) string {
	if req == nil || req.Data == nil {
		return ""
	}
	if id := req.Data.WorkspaceUserId; id != "" {
		return id
	}
	if uc.repositories.WorkspaceUserRole == nil || req.Data.Id == "" {
		return ""
	}
	read, err := uc.repositories.WorkspaceUserRole.ReadWorkspaceUserRole(ctx, &workspaceuserrolepb.ReadWorkspaceUserRoleRequest{
		Data: &workspaceuserrolepb.WorkspaceUserRole{Id: req.Data.Id},
	})
	if err != nil || read == nil {
		return ""
	}
	for _, row := range read.GetData() {
		if row != nil && row.GetWorkspaceUserId() != "" {
			return row.GetWorkspaceUserId()
		}
	}
	return ""
}

// invalidateBindingCache drops the workspace_user binding's cached permission
// codes. Nil-safe: no-op when the invalidator is unwired or the id is empty.
func (uc *DeleteWorkspaceUserRoleUseCase) invalidateBindingCache(workspaceUserID string) {
	if uc.services.PermissionCacheInvalidator == nil || workspaceUserID == "" {
		return
	}
	uc.services.PermissionCacheInvalidator.InvalidateBinding(workspaceUserID)
}

// executeWithTransaction executes workspace user role deletion within a transaction
func (uc *DeleteWorkspaceUserRoleUseCase) executeWithTransaction(ctx context.Context, req *workspaceuserrolepb.DeleteWorkspaceUserRoleRequest) (*workspaceuserrolepb.DeleteWorkspaceUserRoleResponse, error) {
	var result *workspaceuserrolepb.DeleteWorkspaceUserRoleResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "workspace_user_role.errors.deletion_failed", "Workspace-User-Role deletion failed ")
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
func (uc *DeleteWorkspaceUserRoleUseCase) executeCore(ctx context.Context, req *workspaceuserrolepb.DeleteWorkspaceUserRoleRequest) (*workspaceuserrolepb.DeleteWorkspaceUserRoleResponse, error) {
	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace_user_role.validation.request_required", "Request is required for workspace user roles "))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace_user_role.validation.id_required", "Workspace-User-Role ID is required "))
	}

	// Call repository
	resp, err := uc.repositories.WorkspaceUserRole.DeleteWorkspaceUserRole(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace_user_role.errors.deletion_failed", "Workspace-User-Role deletion failed ")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	log.Printf("AUTHZ_CHANGE | action=unassign_role | workspace_user_role_id=%s", req.Data.Id)

	return resp, nil
}

// Helper functions
