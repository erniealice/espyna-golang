package resource

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	resourcepb "leapfor.xyz/esqyma/golang/v1/domain/product/resource"
)

// ListResourcesUseCase handles the business logic for listing resources
// ListResourcesRepositories groups all repository dependencies
type ListResourcesRepositories struct {
	Resource resourcepb.ResourceDomainServiceServer // Primary entity repository
}

// ListResourcesServices groups all business service dependencies
type ListResourcesServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ListResourcesUseCase handles the business logic for listing resources
type ListResourcesUseCase struct {
	repositories ListResourcesRepositories
	services     ListResourcesServices
}

// NewListResourcesUseCase creates a new ListResourcesUseCase
func NewListResourcesUseCase(
	repositories ListResourcesRepositories,
	services ListResourcesServices,
) *ListResourcesUseCase {
	return &ListResourcesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list resources operation
func (uc *ListResourcesUseCase) Execute(ctx context.Context, req *resourcepb.ListResourcesRequest) (*resourcepb.ListResourcesResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		userID := contextutil.ExtractUserIDFromContext(ctx)
		if userID == "" {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.errors.user_not_authenticated", "User not authenticated"))
		}

		permission := ports.EntityPermission(ports.EntityResource, ports.ActionList)
		authorized, err := uc.services.AuthorizationService.HasGlobalPermission(ctx, userID, permission)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.errors.authorization_check_failed", "Authorization check failed")
			return nil, fmt.Errorf("%s: %w", translatedError, err)
		}

		if !authorized {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.errors.authorization_failed", "Access denied")
			return nil, errors.New(translatedError)
		}
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.Resource.ListResources(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListResourcesUseCase) validateInput(ctx context.Context, req *resourcepb.ListResourcesRequest) error {

	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.validation.request_required", "Request is required [DEFAULT]"))
	}
	// Additional validation can be added here if needed
	return nil
}
