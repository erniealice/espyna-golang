package workspace_user_role

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	rolepb "leapfor.xyz/esqyma/golang/v1/domain/entity/role"
	workspaceuserpb "leapfor.xyz/esqyma/golang/v1/domain/entity/workspace_user"
	workspaceuserrolepb "leapfor.xyz/esqyma/golang/v1/domain/entity/workspace_user_role"
)

// UpdateWorkspaceUserRoleRepositories groups all repository dependencies
type UpdateWorkspaceUserRoleRepositories struct {
	WorkspaceUserRole workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer // Primary entity repository
	WorkspaceUser     workspaceuserpb.WorkspaceUserDomainServiceServer         // Entity reference validation
	Role              rolepb.RoleDomainServiceServer                           // Entity reference validation
}

// UpdateWorkspaceUserRoleServices groups all business service dependencies
type UpdateWorkspaceUserRoleServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateWorkspaceUserRoleUseCase handles the business logic for updating a workspace user role
type UpdateWorkspaceUserRoleUseCase struct {
	repositories UpdateWorkspaceUserRoleRepositories
	services     UpdateWorkspaceUserRoleServices
}

// NewUpdateWorkspaceUserRoleUseCase creates use case with grouped dependencies
func NewUpdateWorkspaceUserRoleUseCase(
	repositories UpdateWorkspaceUserRoleRepositories,
	services UpdateWorkspaceUserRoleServices,
) *UpdateWorkspaceUserRoleUseCase {
	return &UpdateWorkspaceUserRoleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update workspace user role operation
func (uc *UpdateWorkspaceUserRoleUseCase) Execute(ctx context.Context, req *workspaceuserrolepb.UpdateWorkspaceUserRoleRequest) (*workspaceuserrolepb.UpdateWorkspaceUserRoleResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		// Extract user ID from context (should be set by authentication middleware)
		userID := contextutil.ExtractUserIDFromContext(ctx)
		if userID == "" {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.errors.user_not_authenticated", "User not authenticated "))
		}

		// Check permission to update workspace user roles
		permission := ports.EntityPermission(ports.EntityWorkspaceUserRole, ports.ActionUpdate)
		authorized, err := uc.services.AuthorizationService.HasGlobalPermission(ctx, userID, permission)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.errors.authorization_check_failed", "Authorization check failed ")
			return nil, fmt.Errorf("%s: %w", translatedError, err)
		}

		if !authorized {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.errors.access_denied", "Access denied ")
			return nil, errors.New(translatedError)
		}
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichWorkspaceUserRoleData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.errors.enrichment_failed", "Business logic enrichment failed ")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Determine if we should use transactions
	if uc.shouldUseTransaction(ctx) {
		return uc.executeWithTransaction(ctx, req)
	}

	// Execute without transaction (backward compatibility)
	return uc.executeWithoutTransaction(ctx, req)
}

// validateInput validates the input request
func (uc *UpdateWorkspaceUserRoleUseCase) validateInput(ctx context.Context, req *workspaceuserrolepb.UpdateWorkspaceUserRoleRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.validation.request_required", "Request is required for workspace user roles "))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.validation.data_required", "Workspace-User-Role data is required "))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.validation.id_required", "Workspace-User-Role ID is required "))
	}
	if req.Data.WorkspaceUserId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.validation.workspace_user_id_required", "Workspace-User ID is required "))
	}
	if req.Data.RoleId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.validation.role_id_required", "Role ID is required "))
	}
	return nil
}

// enrichWorkspaceUserRoleData adds generated fields and audit information
func (uc *UpdateWorkspaceUserRoleUseCase) enrichWorkspaceUserRoleData(workspaceUserRole *workspaceuserrolepb.WorkspaceUserRole) error {
	now := time.Now()

	// Update audit fields (preserve original creation date)
	workspaceUserRole.DateModified = &[]int64{now.UnixMilli()}[0]
	workspaceUserRole.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateWorkspaceUserRoleUseCase) validateBusinessRules(ctx context.Context, workspaceUserRole *workspaceuserrolepb.WorkspaceUserRole) error {
	// Validate workspace user and role relationship
	if workspaceUserRole.WorkspaceUserId == workspaceUserRole.RoleId {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.validation.same_id", "Workspace user ID and role ID cannot be the same "))
	}

	return nil
}

// shouldUseTransaction determines if this operation should use a transaction
func (uc *UpdateWorkspaceUserRoleUseCase) shouldUseTransaction(ctx context.Context) bool {
	// Use transaction if:
	// 1. TransactionService is available, AND
	// 2. We're not already in a transaction context
	if uc.services.TransactionService == nil || !uc.services.TransactionService.SupportsTransactions() {
		return false
	}

	// Don't start a nested transaction if we're already in one
	if uc.services.TransactionService.IsTransactionActive(ctx) {
		return false
	}

	return true
}

// executeWithTransaction performs the operation within a transaction
func (uc *UpdateWorkspaceUserRoleUseCase) executeWithTransaction(ctx context.Context, req *workspaceuserrolepb.UpdateWorkspaceUserRoleRequest) (*workspaceuserrolepb.UpdateWorkspaceUserRoleResponse, error) {
	var response *workspaceuserrolepb.UpdateWorkspaceUserRoleResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// All validations and operations within transaction

		// Entity reference validation (reads happen in transaction context)
		if err := uc.validateEntityReferences(txCtx, req.Data); err != nil {
			// The validateEntityReferences function now returns clean translated errors
			// Return the error directly without additional wrapping
			return err
		}

		// Business rule validation
		if err := uc.validateBusinessRules(txCtx, req.Data); err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "workspace_user_role.errors.business_rule_validation_failed", "Business rule validation failed ")
			return fmt.Errorf("%s: %w", translatedError, err)
		}

		// Update WorkspaceUserRole (will participate in transaction)
		updateResponse, err := uc.repositories.WorkspaceUserRole.UpdateWorkspaceUserRole(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "workspace_user_role.errors.update_failed", "Workspace-User-Role update failed ")
			return fmt.Errorf("%s: %w", translatedError, err)
		}

		response = updateResponse
		return nil
	})

	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.errors.transaction_failed", "Transaction execution failed ")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return response, nil
}

// executeWithoutTransaction performs the operation without transaction (backward compatibility)
func (uc *UpdateWorkspaceUserRoleUseCase) executeWithoutTransaction(ctx context.Context, req *workspaceuserrolepb.UpdateWorkspaceUserRoleRequest) (*workspaceuserrolepb.UpdateWorkspaceUserRoleResponse, error) {
	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		// The validateEntityReferences function now returns clean translated errors
		// Return the error directly without additional wrapping
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.errors.business_rule_validation_failed", "Business rule validation failed ")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository (no transaction)
	resp, err := uc.repositories.WorkspaceUserRole.UpdateWorkspaceUserRole(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.errors.update_failed", "Workspace-User-Role update failed ")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateWorkspaceUserRoleUseCase) validateEntityReferences(ctx context.Context, workspaceUserRole *workspaceuserrolepb.WorkspaceUserRole) error {
	// Validate WorkspaceUser entity reference
	if workspaceUserRole.WorkspaceUserId != "" {
		workspaceUser, err := uc.repositories.WorkspaceUser.ReadWorkspaceUser(ctx, &workspaceuserpb.ReadWorkspaceUserRequest{
			Data: &workspaceuserpb.WorkspaceUser{Id: workspaceUserRole.WorkspaceUserId},
		})
		if err != nil {
			// Check if this is a "not found" error - if so, create our own translated message
			// Otherwise, return the error as-is for the calling function to handle
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist") {
				translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.errors.workspace_user_not_found", "Referenced workspace user with ID \"{workspaceUserId}\" not found ")
				translatedError = strings.ReplaceAll(translatedError, "{workspaceUserId}", workspaceUserRole.WorkspaceUserId)
				return errors.New(translatedError)
			}
			// For other errors, return as-is
			return err
		}
		if workspaceUser == nil || workspaceUser.Data == nil || len(workspaceUser.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.errors.workspace_user_not_found", "Referenced workspace user with ID \"{workspaceUserId}\" not found ")
			translatedError = strings.ReplaceAll(translatedError, "{workspaceUserId}", workspaceUserRole.WorkspaceUserId)
			return errors.New(translatedError)
		}
		if !workspaceUser.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.errors.workspace_user_not_active", "Referenced workspace user with ID \"{workspaceUserId}\" is not active ")
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
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.errors.role_not_found", "Referenced role with ID \"{roleId}\" does not exist ")
			translatedError = strings.ReplaceAll(translatedError, "{roleId}", workspaceUserRole.RoleId)
			return errors.New(translatedError)
		}
		if !role.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "workspace_user_role.errors.role_not_active", "Referenced role with ID \"{roleId}\" is not active ")
			translatedError = strings.ReplaceAll(translatedError, "{roleId}", workspaceUserRole.RoleId)
			return errors.New(translatedError)
		}
	}

	return nil
}

// Helper functions
