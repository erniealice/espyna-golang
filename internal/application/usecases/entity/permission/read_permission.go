package permission

import (
	"context"
	"errors"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
)

// ReadPermissionRepositories groups all repository dependencies
type ReadPermissionRepositories struct {
	Permission permissionpb.PermissionDomainServiceServer // Primary entity repository
}

// ReadPermissionServices groups all business service dependencies
type ReadPermissionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadPermissionUseCase handles the business logic for reading permissions
type ReadPermissionUseCase struct {
	repositories ReadPermissionRepositories
	services     ReadPermissionServices
}

// NewReadPermissionUseCase creates use case with grouped dependencies
func NewReadPermissionUseCase(
	repositories ReadPermissionRepositories,
	services ReadPermissionServices,
) *ReadPermissionUseCase {
	return &ReadPermissionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadPermissionUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadPermissionUseCase with grouped parameters instead
func NewReadPermissionUseCaseUngrouped(permissionRepo permissionpb.PermissionDomainServiceServer) *ReadPermissionUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadPermissionRepositories{
		Permission: permissionRepo,
	}

	services := ReadPermissionServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadPermissionUseCase(repositories, services)
}

func (uc *ReadPermissionUseCase) Execute(ctx context.Context, req *permissionpb.ReadPermissionRequest) (*permissionpb.ReadPermissionResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.Permission.ReadPermission(ctx, req)
	if err != nil {
		return nil, err
	}

	// Not found error
	if len(resp.Data) == 0 || resp.Data[0].Id == "" { // Assuming resp.Data will be nil or have empty ID if not found
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.errors.not_found", "Permission with ID \"{permissionId}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{permissionId}", req.Data.Id)
		return nil, errors.New(translatedError)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadPermissionUseCase) validateInput(ctx context.Context, req *permissionpb.ReadPermissionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.validation.request_required", "Request is required for permissions [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.validation.data_required", "Permission data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "permission.validation.id_required", "Permission ID is required [DEFAULT]"))
	}
	return nil
}
