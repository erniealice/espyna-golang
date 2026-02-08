package role

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
)

// ListRolesRepositories groups all repository dependencies
type ListRolesRepositories struct {
	Role rolepb.RoleDomainServiceServer // Primary entity repository
}

// ListRolesServices groups all business service dependencies
type ListRolesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListRolesUseCase handles the business logic for listing roles
type ListRolesUseCase struct {
	repositories ListRolesRepositories
	services     ListRolesServices
}

// NewListRolesUseCase creates use case with grouped dependencies
func NewListRolesUseCase(
	repositories ListRolesRepositories,
	services ListRolesServices,
) *ListRolesUseCase {
	return &ListRolesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListRolesUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListRolesUseCase with grouped parameters instead
func NewListRolesUseCaseUngrouped(roleRepo rolepb.RoleDomainServiceServer) *ListRolesUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListRolesRepositories{
		Role: roleRepo,
	}

	services := ListRolesServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewListRolesUseCase(repositories, services)
}

// Execute performs the list roles operation
func (uc *ListRolesUseCase) Execute(ctx context.Context, req *rolepb.ListRolesRequest) (*rolepb.ListRolesResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Role.ListRoles(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.errors.list_failed", "Failed to retrieve roles [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListRolesUseCase) validateInput(ctx context.Context, req *rolepb.ListRolesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "role.validation.request_required", "Request is required for roles [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for listing
func (uc *ListRolesUseCase) validateBusinessRules(ctx context.Context, req *rolepb.ListRolesRequest) error {
	// No specific business rules for listing roles
	return nil
}
