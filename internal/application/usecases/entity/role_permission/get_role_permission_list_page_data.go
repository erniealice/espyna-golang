//go:build mock_db && mock_auth

package role_permission

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	rolepermissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role_permission"
)

// GetRolePermissionListPageDataRepositories groups all repository dependencies
type GetRolePermissionListPageDataRepositories struct {
	RolePermission rolepermissionpb.RolePermissionDomainServiceServer // Primary entity repository
}

// GetRolePermissionListPageDataServices groups all business service dependencies
type GetRolePermissionListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetRolePermissionListPageDataUseCase handles the business logic for retrieving role permission list page data
type GetRolePermissionListPageDataUseCase struct {
	repositories GetRolePermissionListPageDataRepositories
	services     GetRolePermissionListPageDataServices
}

// NewGetRolePermissionListPageDataUseCase creates use case with grouped dependencies
func NewGetRolePermissionListPageDataUseCase(
	repositories GetRolePermissionListPageDataRepositories,
	services GetRolePermissionListPageDataServices,
) *GetRolePermissionListPageDataUseCase {
	return &GetRolePermissionListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetRolePermissionListPageDataUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewGetRolePermissionListPageDataUseCase with grouped parameters instead
func NewGetRolePermissionListPageDataUseCaseUngrouped(rolePermissionRepo rolepermissionpb.RolePermissionDomainServiceServer) *GetRolePermissionListPageDataUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := GetRolePermissionListPageDataRepositories{
		RolePermission: rolePermissionRepo,
	}

	services := GetRolePermissionListPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewGetRolePermissionListPageDataUseCase(repositories, services)
}

// Execute performs the get role permission list page data operation
func (uc *GetRolePermissionListPageDataUseCase) Execute(ctx context.Context, req *rolepermissionpb.GetRolePermissionListPageDataRequest) (*rolepermissionpb.GetRolePermissionListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityRolePermission, ports.ActionList); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes role permission list page data retrieval within a transaction
func (uc *GetRolePermissionListPageDataUseCase) executeWithTransaction(ctx context.Context, req *rolepermissionpb.GetRolePermissionListPageDataRequest) (*rolepermissionpb.GetRolePermissionListPageDataResponse, error) {
	var result *rolepermissionpb.GetRolePermissionListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "role_permission.errors.list_page_data_retrieval_failed", "Role permission list page data retrieval failed [DEFAULT]")
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
func (uc *GetRolePermissionListPageDataUseCase) executeCore(ctx context.Context, req *rolepermissionpb.GetRolePermissionListPageDataRequest) (*rolepermissionpb.GetRolePermissionListPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.RolePermission.GetRolePermissionListPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetRolePermissionListPageDataUseCase) validateInput(ctx context.Context, req *rolepermissionpb.GetRolePermissionListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.validation.request_required", "Request is required for role permission list page data [DEFAULT]"))
	}

	// Validate pagination parameters if provided
	if req.Pagination != nil {
		if req.Pagination.Limit < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.validation.invalid_limit", "Pagination limit cannot be negative [DEFAULT]"))
		}
		if req.Pagination.Limit > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role_permission.validation.limit_too_large", "Pagination limit cannot exceed 100 [DEFAULT]"))
		}
	}

	return nil
}

