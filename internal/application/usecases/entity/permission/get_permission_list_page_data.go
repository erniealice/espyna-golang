package permission

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
)

// GetPermissionListPageDataRepositories groups all repository dependencies
type GetPermissionListPageDataRepositories struct {
	Permission permissionpb.PermissionDomainServiceServer // Primary entity repository
}

// GetPermissionListPageDataServices groups all business service dependencies
type GetPermissionListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetPermissionListPageDataUseCase handles the business logic for getting permission list page data
type GetPermissionListPageDataUseCase struct {
	repositories GetPermissionListPageDataRepositories
	services     GetPermissionListPageDataServices
}

// NewGetPermissionListPageDataUseCase creates use case with grouped dependencies
func NewGetPermissionListPageDataUseCase(
	repositories GetPermissionListPageDataRepositories,
	services GetPermissionListPageDataServices,
) *GetPermissionListPageDataUseCase {
	return &GetPermissionListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetPermissionListPageDataUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewGetPermissionListPageDataUseCase with grouped parameters instead
func NewGetPermissionListPageDataUseCaseUngrouped(permissionRepo permissionpb.PermissionDomainServiceServer) *GetPermissionListPageDataUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := GetPermissionListPageDataRepositories{
		Permission: permissionRepo,
	}

	services := GetPermissionListPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewGetPermissionListPageDataUseCase(repositories, services)
}

// Execute performs the get permission list page data operation
func (uc *GetPermissionListPageDataUseCase) Execute(ctx context.Context, req *permissionpb.GetPermissionListPageDataRequest) (*permissionpb.GetPermissionListPageDataResponse, error) {
	// Input validation
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	// Authorization check (if authorization service is available)
	if uc.services.AuthorizationService != nil {
		userID := contextutil.ExtractUserIDFromContext(ctx)
		if userID == "" {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "auth.errors.user_not_authenticated", "User not authenticated [DEFAULT]")
			return nil, fmt.Errorf("%s", translatedError)
		}

		// Check if user has permission to list permissions
		isAuthorized, err := uc.services.AuthorizationService.HasPermission(ctx, userID, "permission.list")
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "auth.errors.authorization_check_failed", "Authorization check failed [DEFAULT]")
			return nil, fmt.Errorf("%s: %w", translatedError, err)
		}

		if !isAuthorized {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "auth.errors.insufficient_permissions", "Insufficient permissions [DEFAULT]")
			return nil, fmt.Errorf("%s", translatedError)
		}
	}

	// Call repository
	resp, err := uc.repositories.Permission.GetPermissionListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.errors.list_page_data_failed", "Failed to retrieve permission list page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
