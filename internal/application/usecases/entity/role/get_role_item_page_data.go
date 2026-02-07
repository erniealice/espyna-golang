package role

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	rolepb "leapfor.xyz/esqyma/golang/v1/domain/entity/role"
)

// GetRoleItemPageDataRepositories groups all repository dependencies
type GetRoleItemPageDataRepositories struct {
	Role rolepb.RoleDomainServiceServer // Primary entity repository
}

// GetRoleItemPageDataServices groups all business service dependencies
type GetRoleItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetRoleItemPageDataUseCase handles the business logic for retrieving role item page data
type GetRoleItemPageDataUseCase struct {
	repositories GetRoleItemPageDataRepositories
	services     GetRoleItemPageDataServices
}

// NewGetRoleItemPageDataUseCase creates use case with grouped dependencies
func NewGetRoleItemPageDataUseCase(
	repositories GetRoleItemPageDataRepositories,
	services GetRoleItemPageDataServices,
) *GetRoleItemPageDataUseCase {
	return &GetRoleItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetRoleItemPageDataUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewGetRoleItemPageDataUseCase with grouped parameters instead
func NewGetRoleItemPageDataUseCaseUngrouped(roleRepo rolepb.RoleDomainServiceServer) *GetRoleItemPageDataUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := GetRoleItemPageDataRepositories{
		Role: roleRepo,
	}

	services := GetRoleItemPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewGetRoleItemPageDataUseCase(repositories, services)
}

// Execute performs the get role item page data operation
func (uc *GetRoleItemPageDataUseCase) Execute(ctx context.Context, req *rolepb.GetRoleItemPageDataRequest) (*rolepb.GetRoleItemPageDataResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes role item page data retrieval within a transaction
func (uc *GetRoleItemPageDataUseCase) executeWithTransaction(ctx context.Context, req *rolepb.GetRoleItemPageDataRequest) (*rolepb.GetRoleItemPageDataResponse, error) {
	var result *rolepb.GetRoleItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "role.errors.item_page_data_retrieval_failed", "Role item page data retrieval failed [DEFAULT]")
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
func (uc *GetRoleItemPageDataUseCase) executeCore(ctx context.Context, req *rolepb.GetRoleItemPageDataRequest) (*rolepb.GetRoleItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Authorization check
	if err := uc.checkPermissions(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.Role.GetRoleItemPageData(ctx, req)
}

// validateInput validates the input request
func (uc *GetRoleItemPageDataUseCase) validateInput(ctx context.Context, req *rolepb.GetRoleItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.validation.request_required", "Request is required for role item page data [DEFAULT]"))
	}

	if req.RoleId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.validation.role_id_required", "Role ID is required [DEFAULT]"))
	}

	return nil
}

// checkPermissions verifies user authorization for role item access
func (uc *GetRoleItemPageDataUseCase) checkPermissions(ctx context.Context, req *rolepb.GetRoleItemPageDataRequest) error {
	if uc.services.AuthorizationService == nil {
		// No authorization service configured, allow access
		return nil
	}

	// Check if user has permission to read roles
	hasPermission, err := uc.services.AuthorizationService.HasPermission(ctx, "", "role.read")
	if err != nil {
		return fmt.Errorf("authorization check failed: %w", err)
	}

	if !hasPermission {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.errors.insufficient_permissions", "Insufficient permissions to access role details [DEFAULT]"))
	}

	return nil
}
