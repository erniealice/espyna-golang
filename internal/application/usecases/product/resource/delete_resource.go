package resource

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	resourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/resource"
)

// DeleteResourceUseCase handles the business logic for deleting resources
// DeleteResourceRepositories groups all repository dependencies
type DeleteResourceRepositories struct {
	Resource resourcepb.ResourceDomainServiceServer // Primary entity repository
}

// DeleteResourceServices groups all business service dependencies
type DeleteResourceServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// DeleteResourceUseCase handles the business logic for deleting resources
type DeleteResourceUseCase struct {
	repositories DeleteResourceRepositories
	services     DeleteResourceServices
}

// NewDeleteResourceUseCase creates a new DeleteResourceUseCase
func NewDeleteResourceUseCase(
	repositories DeleteResourceRepositories,
	services DeleteResourceServices,
) *DeleteResourceUseCase {
	return &DeleteResourceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete resource operation
func (uc *DeleteResourceUseCase) Execute(ctx context.Context, req *resourcepb.DeleteResourceRequest) (*resourcepb.DeleteResourceResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityResource, ports.ActionDelete); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

// executeWithTransaction wraps the core logic in a transaction
func (uc *DeleteResourceUseCase) executeWithTransaction(ctx context.Context, req *resourcepb.DeleteResourceRequest) (*resourcepb.DeleteResourceResponse, error) {
	var result *resourcepb.DeleteResourceResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})
	return result, err
}

// executeCore contains the core deletion logic
func (uc *DeleteResourceUseCase) executeCore(ctx context.Context, req *resourcepb.DeleteResourceRequest) (*resourcepb.DeleteResourceResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.Resource.DeleteResource(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteResourceUseCase) validateInput(ctx context.Context, req *resourcepb.DeleteResourceRequest) error {

	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.data_required", "Resource data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.id_required", "Resource ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for resource deletion
func (uc *DeleteResourceUseCase) validateBusinessRules(ctx context.Context, req *resourcepb.DeleteResourceRequest) error {

	// Additional business rule validation can be added here
	// For example: check if resource is in use by active events or bookings
	if uc.isResourceInUse(ctx, req.Data.Id) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.errors.in_use", "Resource is currently in use and cannot be deleted [DEFAULT]"))
	}
	return nil
}

// isResourceInUse checks if the resource is referenced by other entities (e.g., active events or bookings)
func (uc *DeleteResourceUseCase) isResourceInUse(ctx context.Context, resourceID string) bool {
	// Placeholder for actual implementation
	// TODO: Implement actual check for resource usage
	return false
}
