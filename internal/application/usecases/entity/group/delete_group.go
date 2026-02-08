package group

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	grouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/group"
)

// DeleteGroupRepositories groups all repository dependencies
type DeleteGroupRepositories struct {
	Group grouppb.GroupDomainServiceServer // Primary entity repository
}

// DeleteGroupServices groups all business service dependencies
type DeleteGroupServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteGroupUseCase handles the business logic for deleting groups
type DeleteGroupUseCase struct {
	repositories DeleteGroupRepositories
	services     DeleteGroupServices
}

// NewDeleteGroupUseCase creates use case with grouped dependencies
func NewDeleteGroupUseCase(
	repositories DeleteGroupRepositories,
	services DeleteGroupServices,
) *DeleteGroupUseCase {
	return &DeleteGroupUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteGroupUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteGroupUseCase with grouped parameters instead
func NewDeleteGroupUseCaseUngrouped(groupRepo grouppb.GroupDomainServiceServer) *DeleteGroupUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteGroupRepositories{
		Group: groupRepo,
	}

	services := DeleteGroupServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewDeleteGroupUseCase(repositories, services)
}

func (uc *DeleteGroupUseCase) Execute(ctx context.Context, req *grouppb.DeleteGroupRequest) (*grouppb.DeleteGroupResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Group.DeleteGroup(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.errors.deletion_failed", "Group deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteGroupUseCase) validateInput(ctx context.Context, req *grouppb.DeleteGroupRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.request_required", "Request is required for groups [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.data_required", "Group data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group.validation.id_required", "Group ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeleteGroupUseCase) validateBusinessRules(ctx context.Context, req *grouppb.DeleteGroupRequest) error {
	// TODO: Add business rules for group deletion
	// Example: Check if group has associated users or resources
	// For now, allow all deletions

	return nil
}
