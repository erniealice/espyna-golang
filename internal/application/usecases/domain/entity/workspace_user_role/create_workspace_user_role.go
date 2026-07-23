package workspace_user_role

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
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

// CreateWorkspaceUserRoleRepositories groups all repository dependencies
type CreateWorkspaceUserRoleRepositories struct {
	WorkspaceUserRole workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer // Primary entity repository
	WorkspaceUser     workspaceuserpb.WorkspaceUserDomainServiceServer         // Entity reference validation
	Role              rolepb.RoleDomainServiceServer                           // Entity reference validation
}

// CreateWorkspaceUserRoleServices groups all business service dependencies
type CreateWorkspaceUserRoleServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator      ports.IDGenerator
	// PermissionCacheInvalidator evicts the assigned binding's cached RBAC codes
	// so granting a role to a user takes effect on the next request (P10/D2).
	// Nil-safe (no-op when unset).
	PermissionCacheInvalidator securityports.PermissionCacheInvalidator
}

// CreateWorkspaceUserRoleUseCase handles the business logic for creating workspace user roles
type CreateWorkspaceUserRoleUseCase struct {
	repositories CreateWorkspaceUserRoleRepositories
	services     CreateWorkspaceUserRoleServices
}

// NewCreateWorkspaceUserRoleUseCase creates use case with grouped dependencies
func NewCreateWorkspaceUserRoleUseCase(
	repositories CreateWorkspaceUserRoleRepositories,
	services CreateWorkspaceUserRoleServices,
) *CreateWorkspaceUserRoleUseCase {
	return &CreateWorkspaceUserRoleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create workspace user role operation
func (uc *CreateWorkspaceUserRoleUseCase) Execute(ctx context.Context, req *workspaceuserrolepb.CreateWorkspaceUserRoleRequest) (*workspaceuserrolepb.CreateWorkspaceUserRoleResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.WorkspaceUserRole,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	var resp *workspaceuserrolepb.CreateWorkspaceUserRoleResponse
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

	// Post-commit: assigning a role to a binding changes that binding's effective
	// permission set — evict its cached codes so the grant is live on the next
	// request (P10/D2). Precise: only this workspace_user's binding entries are
	// dropped (the cache-key bindingID == workspace_user_id), never other users.
	if req != nil && req.Data != nil {
		uc.invalidateBindingCache(req.Data.WorkspaceUserId)
	}
	return resp, nil
}

// invalidateBindingCache drops the workspace_user binding's cached permission
// codes. Nil-safe: no-op when the invalidator is unwired or the id is empty.
func (uc *CreateWorkspaceUserRoleUseCase) invalidateBindingCache(workspaceUserID string) {
	if uc.services.PermissionCacheInvalidator == nil || workspaceUserID == "" {
		return
	}
	uc.services.PermissionCacheInvalidator.InvalidateBinding(workspaceUserID)
}

// executeWithTransaction executes workspace user role creation within a transaction
func (uc *CreateWorkspaceUserRoleUseCase) executeWithTransaction(ctx context.Context, req *workspaceuserrolepb.CreateWorkspaceUserRoleRequest) (*workspaceuserrolepb.CreateWorkspaceUserRoleResponse, error) {
	var result *workspaceuserrolepb.CreateWorkspaceUserRoleResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "workspace_user_role.errors.creation_failed", "Workspace-User-Role creation failed ")
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
func (uc *CreateWorkspaceUserRoleUseCase) executeCore(ctx context.Context, req *workspaceuserrolepb.CreateWorkspaceUserRoleRequest) (*workspaceuserrolepb.CreateWorkspaceUserRoleResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichWorkspaceUserRoleData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace_user_role.errors.enrichment_failed", "Business logic enrichment failed ")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		// The validateEntityReferences function now returns clean translated errors
		// Return the error directly without additional wrapping
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace_user_role.errors.business_rule_validation_failed", "Business rule validation failed ")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.WorkspaceUserRole.CreateWorkspaceUserRole(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace_user_role.errors.creation_failed", "Workspace-User-Role creation failed ")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	log.Printf("AUTHZ_CHANGE | action=assign_role | workspace_user_role_id=%s", req.Data.Id)

	return resp, nil
}

// validateInput validates the input request
func (uc *CreateWorkspaceUserRoleUseCase) validateInput(ctx context.Context, req *workspaceuserrolepb.CreateWorkspaceUserRoleRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace_user_role.validation.request_required", "Request is required for workspace user roles "))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace_user_role.validation.data_required", "Workspace-User-Role data is required "))
	}
	if req.Data.WorkspaceUserId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace_user_role.validation.workspace_user_id_required", "Workspace-User ID is required "))
	}
	if req.Data.RoleId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace_user_role.validation.role_id_required", "Role ID is required "))
	}
	return nil
}

// enrichWorkspaceUserRoleData adds generated fields and audit information
func (uc *CreateWorkspaceUserRoleUseCase) enrichWorkspaceUserRoleData(workspaceUserRole *workspaceuserrolepb.WorkspaceUserRole) error {
	now := time.Now()

	// Generate WorkspaceUserRole ID if not provided
	if workspaceUserRole.Id == "" {
		workspaceUserRole.Id = uc.services.IDGenerator.GenerateID()
	}

	// Set audit fields
	workspaceUserRole.DateCreated = &[]int64{now.UnixMilli()}[0]
	workspaceUserRole.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	workspaceUserRole.DateModified = &[]int64{now.UnixMilli()}[0]
	workspaceUserRole.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	workspaceUserRole.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateWorkspaceUserRoleUseCase) validateBusinessRules(ctx context.Context, workspaceUserRole *workspaceuserrolepb.WorkspaceUserRole) error {
	// Validate workspace user and role relationship
	if workspaceUserRole.WorkspaceUserId == workspaceUserRole.RoleId {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace_user_role.validation.same_id", "Workspace user ID and role ID cannot be the same "))
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateWorkspaceUserRoleUseCase) validateEntityReferences(ctx context.Context, workspaceUserRole *workspaceuserrolepb.WorkspaceUserRole) error {
	// Validate WorkspaceUser entity reference
	if workspaceUserRole.WorkspaceUserId != "" {
		workspaceUser, err := uc.repositories.WorkspaceUser.ReadWorkspaceUser(ctx, &workspaceuserpb.ReadWorkspaceUserRequest{
			Data: &workspaceuserpb.WorkspaceUser{Id: workspaceUserRole.WorkspaceUserId},
		})
		if err != nil {
			// Check if this is a "not found" error - if so, create our own translated message
			// Otherwise, return the error as-is for the calling function to handle
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist") {
				translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace_user_role.errors.workspace_user_not_found", "Referenced workspace user with ID \"{workspaceUserId}\" not found ")
				translatedError = strings.ReplaceAll(translatedError, "{workspaceUserId}", workspaceUserRole.WorkspaceUserId)
				return errors.New(translatedError)
			}
			// For other errors, return as-is
			return err
		}
		if workspaceUser == nil || workspaceUser.Data == nil || len(workspaceUser.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace_user_role.errors.workspace_user_not_found", "Referenced workspace user with ID \"{workspaceUserId}\" not found ")
			translatedError = strings.ReplaceAll(translatedError, "{workspaceUserId}", workspaceUserRole.WorkspaceUserId)
			return errors.New(translatedError)
		}
		if !workspaceUser.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace_user_role.errors.workspace_user_not_active", "Referenced workspace user with ID \"{workspaceUserId}\" is not active ")
			translatedError = strings.ReplaceAll(translatedError, "{workspaceUserId}", workspaceUserRole.WorkspaceUserId)
			return errors.New(translatedError)
		}
	}

	// Validate Role entity reference
	if workspaceUserRole.RoleId != "" {
		role, err := uc.repositories.Role.ReadRole(ctx, &rolepb.ReadRoleRequest{
			Data: &rolepb.Role{Id: workspaceUserRole.RoleId},
		})
		if err != nil {
			// Return the underlying error directly - let the calling function handle wrapping
			return err
		}
		if role == nil || role.Data == nil || len(role.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace_user_role.errors.role_not_found", "Referenced role with ID \"{roleId}\" does not exist ")
			translatedError = strings.ReplaceAll(translatedError, "{roleId}", workspaceUserRole.RoleId)
			return errors.New(translatedError)
		}
		if !role.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace_user_role.errors.role_not_active", "Referenced role with ID \"{roleId}\" is not active ")
			translatedError = strings.ReplaceAll(translatedError, "{roleId}", workspaceUserRole.RoleId)
			return errors.New(translatedError)
		}
	}

	return nil
}

// Helper functions

// Additional validation methods can be added here as needed
