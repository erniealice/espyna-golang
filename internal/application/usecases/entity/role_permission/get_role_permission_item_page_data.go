//go:build mock_db && mock_auth

package role_permission

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	rolepermissionpb "leapfor.xyz/esqyma/golang/v1/domain/entity/role_permission"
)

// GetRolePermissionItemPageDataRepositories groups all repository dependencies
type GetRolePermissionItemPageDataRepositories struct {
	RolePermission rolepermissionpb.RolePermissionDomainServiceServer // Primary entity repository
}

// GetRolePermissionItemPageDataServices groups all business service dependencies
type GetRolePermissionItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetRolePermissionItemPageDataUseCase handles the business logic for retrieving role permission item page data
type GetRolePermissionItemPageDataUseCase struct {
	repositories GetRolePermissionItemPageDataRepositories
	services     GetRolePermissionItemPageDataServices
}

// NewGetRolePermissionItemPageDataUseCase creates use case with grouped dependencies
func NewGetRolePermissionItemPageDataUseCase(
	repositories GetRolePermissionItemPageDataRepositories,
	services GetRolePermissionItemPageDataServices,
) *GetRolePermissionItemPageDataUseCase {
	return &GetRolePermissionItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetRolePermissionItemPageDataUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewGetRolePermissionItemPageDataUseCase with grouped parameters instead
func NewGetRolePermissionItemPageDataUseCaseUngrouped(rolePermissionRepo rolepermissionpb.RolePermissionDomainServiceServer) *GetRolePermissionItemPageDataUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := GetRolePermissionItemPageDataRepositories{
		RolePermission: rolePermissionRepo,
	}

	services := GetRolePermissionItemPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewGetRolePermissionItemPageDataUseCase(repositories, services)
}

// Execute performs the get role permission item page data operation
func (uc *GetRolePermissionItemPageDataUseCase) Execute(ctx context.Context, req *rolepermissionpb.GetRolePermissionItemPageDataRequest) (*rolepermissionpb.GetRolePermissionItemPageDataResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes role permission item page data retrieval within a transaction
func (uc *GetRolePermissionItemPageDataUseCase) executeWithTransaction(ctx context.Context, req *rolepermissionpb.GetRolePermissionItemPageDataRequest) (*rolepermissionpb.GetRolePermissionItemPageDataResponse, error) {
	var result *rolepermissionpb.GetRolePermissionItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "role_permission.errors.item_page_data_retrieval_failed", "Role permission item page data retrieval failed [DEFAULT]")
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

// executeCore contains the core business logic
func (uc *GetRolePermissionItemPageDataUseCase) executeCore(ctx context.Context, req *rolepermissionpb.GetRolePermissionItemPageDataRequest) (*rolepermissionpb.GetRolePermissionItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Authorization check
	if err := uc.checkPermissions(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.RolePermission.GetRolePermissionItemPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetRolePermissionItemPageDataUseCase) validateInput(ctx context.Context, req *rolepermissionpb.GetRolePermissionItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.validation.request_required", "Request is required for role permission item page data [DEFAULT]"))
	}

	if req.RolePermissionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.validation.role_permission_id_required", "Role permission ID is required [DEFAULT]"))
	}

	return nil
}

// checkPermissions verifies user authorization for role permission item access
func (uc *GetRolePermissionItemPageDataUseCase) checkPermissions(ctx context.Context, req *rolepermissionpb.GetRolePermissionItemPageDataRequest) error {
	if uc.services.AuthorizationService == nil {
		// No authorization service configured, allow access
		return nil
	}

	// Check if user has permission to read role permissions
	userID := contextutil.ExtractUserIDFromContext(ctx)
	hasPermission, err := uc.services.AuthorizationService.HasPermission(ctx, userID, "role_permission.read")
	if err != nil {
		return fmt.Errorf("authorization check failed: %w", err)
	}

	if !hasPermission {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.errors.insufficient_permissions", "Insufficient permissions to access role permission details [DEFAULT]"))
	}

	return nil
}
