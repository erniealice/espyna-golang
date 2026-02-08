package role

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
)

// GetRoleListPageDataRepositories groups all repository dependencies
type GetRoleListPageDataRepositories struct {
	Role rolepb.RoleDomainServiceServer // Primary entity repository
}

// GetRoleListPageDataServices groups all business service dependencies
type GetRoleListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetRoleListPageDataUseCase handles the business logic for retrieving role list page data
type GetRoleListPageDataUseCase struct {
	repositories GetRoleListPageDataRepositories
	services     GetRoleListPageDataServices
}

// NewGetRoleListPageDataUseCase creates use case with grouped dependencies
func NewGetRoleListPageDataUseCase(
	repositories GetRoleListPageDataRepositories,
	services GetRoleListPageDataServices,
) *GetRoleListPageDataUseCase {
	return &GetRoleListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetRoleListPageDataUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewGetRoleListPageDataUseCase with grouped parameters instead
func NewGetRoleListPageDataUseCaseUngrouped(roleRepo rolepb.RoleDomainServiceServer) *GetRoleListPageDataUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := GetRoleListPageDataRepositories{
		Role: roleRepo,
	}

	services := GetRoleListPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewGetRoleListPageDataUseCase(repositories, services)
}

// Execute performs the get role list page data operation
func (uc *GetRoleListPageDataUseCase) Execute(ctx context.Context, req *rolepb.GetRoleListPageDataRequest) (*rolepb.GetRoleListPageDataResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes role list page data retrieval within a transaction
func (uc *GetRoleListPageDataUseCase) executeWithTransaction(ctx context.Context, req *rolepb.GetRoleListPageDataRequest) (*rolepb.GetRoleListPageDataResponse, error) {
	var result *rolepb.GetRoleListPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "role.errors.list_page_data_retrieval_failed", "Role list page data retrieval failed [DEFAULT]")
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
func (uc *GetRoleListPageDataUseCase) executeCore(ctx context.Context, req *rolepb.GetRoleListPageDataRequest) (*rolepb.GetRoleListPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Authorization check
	if err := uc.checkPermissions(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.Role.GetRoleListPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetRoleListPageDataUseCase) validateInput(ctx context.Context, req *rolepb.GetRoleListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.validation.request_required", "Request is required for role list page data [DEFAULT]"))
	}

	// Validate pagination parameters if provided
	if req.Pagination != nil {
		if req.Pagination.Limit < 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.validation.invalid_limit", "Pagination limit cannot be negative [DEFAULT]"))
		}
		if req.Pagination.Limit > 100 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.validation.limit_too_large", "Pagination limit cannot exceed 100 [DEFAULT]"))
		}
	}

	return nil
}

// checkPermissions verifies user authorization for role list access
func (uc *GetRoleListPageDataUseCase) checkPermissions(ctx context.Context, req *rolepb.GetRoleListPageDataRequest) error {
	if uc.services.AuthorizationService == nil {
		// No authorization service configured, allow access
		return nil
	}

	// Check if user has permission to list roles
	hasPermission, err := uc.services.AuthorizationService.HasPermission(ctx, "", "role.list")
	if err != nil {
		return fmt.Errorf("authorization check failed: %w", err)
	}

	if !hasPermission {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.errors.insufficient_permissions", "Insufficient permissions to access role list [DEFAULT]"))
	}

	return nil
}
