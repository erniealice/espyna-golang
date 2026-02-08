package permission

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
)

// CreatePermissionRepositories groups all repository dependencies
type CreatePermissionRepositories struct {
	Permission permissionpb.PermissionDomainServiceServer // Primary entity repository
}

// CreatePermissionServices groups all business service dependencies
type CreatePermissionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreatePermissionUseCase(repositories, services)
}

func (uc *CreatePermissionUseCase) Execute(ctx context.Context, req *permissionpb.CreatePermissionRequest) (*permissionpb.CreatePermissionResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes permission creation within a transaction
func (uc *CreatePermissionUseCase) executeWithTransaction(ctx context.Context, req *permissionpb.CreatePermissionRequest) (*permissionpb.CreatePermissionResponse, error) {
	var result *permissionpb.CreatePermissionResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "permission.errors.creation_failed", "Permission creation failed [DEFAULT]")
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
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.validation.request_required", "Request is required for permissions [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.validation.data_required", "Permission data is required [DEFAULT]"))
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

// enrichPermissionData adds generated fields and audit information
func (uc *CreatePermissionUseCase) enrichPermissionData(permission *permissionpb.Permission) error {
	now := time.Now()

	// Generate Permission ID if not provided
	if permission.Id == "" {
		permission.Id = uc.services.IDService.GenerateID()
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
