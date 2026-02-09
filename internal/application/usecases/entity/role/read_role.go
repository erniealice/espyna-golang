package role

import (
	"context"
	"errors"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
)

// ReadRoleRepositories groups all repository dependencies
type ReadRoleRepositories struct {
	Role rolepb.RoleDomainServiceServer // Primary entity repository
}

// ReadRoleServices groups all business service dependencies
type ReadRoleServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadRoleUseCase handles the business logic for reading roles
type ReadRoleUseCase struct {
	repositories ReadRoleRepositories
	services     ReadRoleServices
}

// NewReadRoleUseCase creates use case with grouped dependencies
func NewReadRoleUseCase(
	repositories ReadRoleRepositories,
	services ReadRoleServices,
) *ReadRoleUseCase {
	return &ReadRoleUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadRoleUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadRoleUseCase with grouped parameters instead
func NewReadRoleUseCaseUngrouped(roleRepo rolepb.RoleDomainServiceServer) *ReadRoleUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadRoleRepositories{
		Role: roleRepo,
	}

	services := ReadRoleServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadRoleUseCase(repositories, services)
}

func (uc *ReadRoleUseCase) Execute(ctx context.Context, req *rolepb.ReadRoleRequest) (*rolepb.ReadRoleResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityRole, ports.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.Role.ReadRole(ctx, req)
	if err != nil {
		return nil, err
	}

	// Not found error
	if len(resp.Data) == 0 || resp.Data[0].Id == "" { // Assuming resp.Data will be nil or have empty ID if not found
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.errors.not_found", "Role with ID \"{roleId}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{roleId}", req.Data.Id)
		return nil, errors.New(translatedError)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadRoleUseCase) validateInput(ctx context.Context, req *rolepb.ReadRoleRequest) error {
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
