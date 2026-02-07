package resource

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	resourcepb "leapfor.xyz/esqyma/golang/v1/domain/product/resource"
)

// ReadResourceUseCase handles the business logic for reading a resource
// ReadResourceRepositories groups all repository dependencies
type ReadResourceRepositories struct {
	Resource resourcepb.ResourceDomainServiceServer // Primary entity repository
}

// ReadResourceServices groups all business service dependencies
type ReadResourceServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ReadResourceUseCase handles the business logic for reading a resource
type ReadResourceUseCase struct {
	repositories ReadResourceRepositories
	services     ReadResourceServices
}

// NewReadResourceUseCase creates a new ReadResourceUseCase
func NewReadResourceUseCase(
	repositories ReadResourceRepositories,
	services ReadResourceServices,
) *ReadResourceUseCase {
	return &ReadResourceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read resource operation
func (uc *ReadResourceUseCase) Execute(ctx context.Context, req *resourcepb.ReadResourceRequest) (*resourcepb.ReadResourceResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		userID := contextutil.ExtractUserIDFromContext(ctx)
		if userID == "" {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.errors.user_not_authenticated", "User not authenticated"))
		}

		permission := ports.EntityPermission(ports.EntityResource, ports.ActionRead)
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
	resp, err := uc.repositories.Resource.ReadResource(ctx, req)
	if err != nil {
		return nil, err
	}

	// Not found error
	if resp == nil || resp.Data == nil || len(resp.Data) == 0 {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "resource.errors.not_found", "Resource with ID \"{resourceId}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{resourceId}", req.Data.Id)
		return nil, errors.New(translatedError)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadResourceUseCase) validateInput(ctx context.Context, req *resourcepb.ReadResourceRequest) error {
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
