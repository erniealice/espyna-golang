package permission

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
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
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPermissions, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	// Call repository
	resp, err := uc.repositories.Permission.GetPermissionListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.errors.list_page_data_failed", "Failed to retrieve permission list page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
