package permission

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
)

// GetPermissionItemPageDataRepositories groups all repository dependencies
type GetPermissionItemPageDataRepositories struct {
	Permission permissionpb.PermissionDomainServiceServer // Primary entity repository
}

// GetPermissionItemPageDataServices groups all business service dependencies
type GetPermissionItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetPermissionItemPageDataUseCase handles the business logic for getting permission item page data
type GetPermissionItemPageDataUseCase struct {
	repositories GetPermissionItemPageDataRepositories
	services     GetPermissionItemPageDataServices
}

// NewGetPermissionItemPageDataUseCase creates use case with grouped dependencies
func NewGetPermissionItemPageDataUseCase(
	repositories GetPermissionItemPageDataRepositories,
	services GetPermissionItemPageDataServices,
) *GetPermissionItemPageDataUseCase {
	return &GetPermissionItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetPermissionItemPageDataUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewGetPermissionItemPageDataUseCase with grouped parameters instead
func NewGetPermissionItemPageDataUseCaseUngrouped(permissionRepo permissionpb.PermissionDomainServiceServer) *GetPermissionItemPageDataUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := GetPermissionItemPageDataRepositories{
		Permission: permissionRepo,
	}

	services := GetPermissionItemPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewGetPermissionItemPageDataUseCase(repositories, services)
}

// Execute performs the get permission item page data operation
func (uc *GetPermissionItemPageDataUseCase) Execute(ctx context.Context, req *permissionpb.GetPermissionItemPageDataRequest) (*permissionpb.GetPermissionItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPermissions, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	if req.PermissionId == "" {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.errors.id_required", "Permission ID is required [DEFAULT]")
		return nil, fmt.Errorf("%s", translatedError)
	}

	// Call repository
	resp, err := uc.repositories.Permission.GetPermissionItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.errors.item_page_data_failed", "Failed to retrieve permission item page data [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
