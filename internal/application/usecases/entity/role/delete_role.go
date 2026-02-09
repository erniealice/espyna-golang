package role

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
)

// DeleteRoleRepositories groups all repository dependencies
type DeleteRoleRepositories struct {
	Role rolepb.RoleDomainServiceServer // Primary entity repository
}

// DeleteRoleServices groups all business service dependencies
type DeleteRoleServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteRoleUseCase handles the business logic for deleting roles
type DeleteRoleUseCase struct {
	repositories DeleteRoleRepositories
	services     DeleteRoleServices
}

// NewDeleteRoleUseCase creates use case with grouped dependencies
func NewDeleteRoleUseCase(
	repositories DeleteRoleRepositories,
	services DeleteRoleServices,
) *DeleteRoleUseCase {
	return &DeleteRoleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteRoleUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteRoleUseCase with grouped parameters instead
func NewDeleteRoleUseCaseUngrouped(roleRepo rolepb.RoleDomainServiceServer) *DeleteRoleUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteRoleRepositories{
		Role: roleRepo,
	}

	services := DeleteRoleServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewDeleteRoleUseCase(repositories, services)
}

// Execute performs the delete role operation
func (uc *DeleteRoleUseCase) Execute(ctx context.Context, req *rolepb.DeleteRoleRequest) (*rolepb.DeleteRoleResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityRole, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes role deletion within a transaction
func (uc *DeleteRoleUseCase) executeWithTransaction(ctx context.Context, req *rolepb.DeleteRoleRequest) (*rolepb.DeleteRoleResponse, error) {
	var result *rolepb.DeleteRoleResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "role.errors.deletion_failed", "Role deletion failed [DEFAULT]")
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
func (uc *DeleteRoleUseCase) executeCore(ctx context.Context, req *rolepb.DeleteRoleRequest) (*rolepb.DeleteRoleResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.Role.DeleteRole(ctx, req)
}

// validateInput validates the input request
func (uc *DeleteRoleUseCase) validateInput(ctx context.Context, req *rolepb.DeleteRoleRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.validation.request_required", "Request is required for roles [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.validation.data_required", "Role data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.validation.id_required", "Role ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeleteRoleUseCase) validateBusinessRules(ctx context.Context, req *rolepb.DeleteRoleRequest) error {
	// TODO: Add business rules for role deletion
	// Example: Check if role is assigned to any users
	// Example: Prevent deletion of system-required roles
	// For now, allow all deletions

	return nil
}
