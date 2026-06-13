package permission

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
)

// DeletePermissionRepositories groups all repository dependencies
type DeletePermissionRepositories struct {
	Permission permissionpb.PermissionDomainServiceServer // Primary entity repository
}

// DeletePermissionServices groups all business service dependencies
type DeletePermissionServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeletePermissionUseCase handles the business logic for deleting permissions
type DeletePermissionUseCase struct {
	repositories DeletePermissionRepositories
	services     DeletePermissionServices
}

// NewDeletePermissionUseCase creates use case with grouped dependencies
func NewDeletePermissionUseCase(
	repositories DeletePermissionRepositories,
	services DeletePermissionServices,
) *DeletePermissionUseCase {
	return &DeletePermissionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeletePermissionUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeletePermissionUseCase with grouped parameters instead
func NewDeletePermissionUseCaseUngrouped(permissionRepo permissionpb.PermissionDomainServiceServer) *DeletePermissionUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeletePermissionRepositories{
		Permission: permissionRepo,
	}

	services := DeletePermissionServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewDeletePermissionUseCase(repositories, services)
}

func (uc *DeletePermissionUseCase) Execute(ctx context.Context, req *permissionpb.DeletePermissionRequest) (*permissionpb.DeletePermissionResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Permission,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes permission deletion within a transaction
func (uc *DeletePermissionUseCase) executeWithTransaction(ctx context.Context, req *permissionpb.DeletePermissionRequest) (*permissionpb.DeletePermissionResponse, error) {
	var result *permissionpb.DeletePermissionResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "permission.errors.deletion_failed", "Permission deletion failed [DEFAULT]")
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
func (uc *DeletePermissionUseCase) executeCore(ctx context.Context, req *permissionpb.DeletePermissionRequest) (*permissionpb.DeletePermissionResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.Permission.DeletePermission(ctx, req)
}

// validateInput validates the input request
func (uc *DeletePermissionUseCase) validateInput(ctx context.Context, req *permissionpb.DeletePermissionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "permission.validation.request_required", "Request is required for permissions [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "permission.validation.data_required", "Permission data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "permission.validation.id_required", "Permission ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeletePermissionUseCase) validateBusinessRules(ctx context.Context, req *permissionpb.DeletePermissionRequest) error {
	// TODO: Add business rules for permission deletion
	// Example: Check if permission is critical for system operation
	// Example: Verify the requester has authority to revoke this permission
	// For now, allow all deletions

	return nil
}
