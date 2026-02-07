package permission

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	permissionpb "leapfor.xyz/esqyma/golang/v1/domain/entity/permission"
)

// ListPermissionsRepositories groups all repository dependencies
type ListPermissionsRepositories struct {
	Permission permissionpb.PermissionDomainServiceServer // Primary entity repository
}

// ListPermissionsServices groups all business service dependencies
type ListPermissionsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListPermissionsUseCase handles the business logic for listing permissions
type ListPermissionsUseCase struct {
	repositories ListPermissionsRepositories
	services     ListPermissionsServices
}

// NewListPermissionsUseCase creates use case with grouped dependencies
func NewListPermissionsUseCase(
	repositories ListPermissionsRepositories,
	services ListPermissionsServices,
) *ListPermissionsUseCase {
	return &ListPermissionsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListPermissionsUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListPermissionsUseCase with grouped parameters instead
func NewListPermissionsUseCaseUngrouped(permissionRepo permissionpb.PermissionDomainServiceServer) *ListPermissionsUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListPermissionsRepositories{
		Permission: permissionRepo,
	}

	services := ListPermissionsServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewListPermissionsUseCase(repositories, services)
}

func (uc *ListPermissionsUseCase) Execute(ctx context.Context, req *permissionpb.ListPermissionsRequest) (*permissionpb.ListPermissionsResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Permission.ListPermissions(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.errors.list_failed", "Failed to retrieve permissions [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListPermissionsUseCase) validateInput(ctx context.Context, req *permissionpb.ListPermissionsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.validation.request_required", "Request is required for permissions [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for listing
func (uc *ListPermissionsUseCase) validateBusinessRules(ctx context.Context, req *permissionpb.ListPermissionsRequest) error {
	// No additional business rules for listing permissions
	// Pagination is not supported in current protobuf definition
	return nil
}
