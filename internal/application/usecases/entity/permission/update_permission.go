package permission

import (
	"context"
	"errors"
	"fmt"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	permissionpb "leapfor.xyz/esqyma/golang/v1/domain/entity/permission"
)

// UpdatePermissionRepositories groups all repository dependencies
type UpdatePermissionRepositories struct {
	Permission permissionpb.PermissionDomainServiceServer // Primary entity repository
}

// UpdatePermissionServices groups all business service dependencies
type UpdatePermissionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUpdatePermissionUseCase(repositories, services)
}

func (uc *UpdatePermissionUseCase) Execute(ctx context.Context, req *permissionpb.UpdatePermissionRequest) (*permissionpb.UpdatePermissionResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes permission update within a transaction
func (uc *UpdatePermissionUseCase) executeWithTransaction(ctx context.Context, req *permissionpb.UpdatePermissionRequest) (*permissionpb.UpdatePermissionResponse, error) {
	var result *permissionpb.UpdatePermissionResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "permission.errors.update_failed", "Permission update failed [DEFAULT]")
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
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.validation.request_required", "Request is required for permissions [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.validation.data_required", "Permission data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.validation.id_required", "Permission ID is required [DEFAULT]"))
	}
	if req.Data.WorkspaceId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.validation.workspace_id_required", "Workspace ID is required [DEFAULT]"))
	}
	if req.Data.UserId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.validation.user_id_required", "User ID is required [DEFAULT]"))
	}
	if req.Data.GrantedByUserId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.validation.granted_by_user_id_required", "Granted by user ID is required [DEFAULT]"))
	}
	if req.Data.PermissionCode == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.validation.permission_code_required", "Permission code is required [DEFAULT]"))
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
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.validation.permission_code_too_short", "Permission code must be at least 3 characters long [DEFAULT]"))
	}

	if len(permission.PermissionCode) > 50 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.validation.permission_code_too_long", "Permission code cannot exceed 50 characters [DEFAULT]"))
	}

	// Validate permission type
	if permission.PermissionType == permissionpb.PermissionType_PERMISSION_TYPE_UNSPECIFIED {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.validation.permission_type_unspecified", "Permission type must be specified [DEFAULT]"))
	}

	// Validate that user is not granting permission to themselves
	if permission.UserId == permission.GrantedByUserId {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.validation.self_grant_not_allowed", "Users cannot grant permissions to themselves [DEFAULT]"))
	}

	return nil
}
