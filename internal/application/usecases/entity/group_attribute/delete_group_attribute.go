package group_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	groupattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/group_attribute"
)

// DeleteGroupAttributeUseCase handles the business logic for deleting group attributes
// DeleteGroupAttributeRepositories groups all repository dependencies
type DeleteGroupAttributeRepositories struct {
	GroupAttribute groupattributepb.GroupAttributeDomainServiceServer // Primary entity repository
}

// DeleteGroupAttributeServices groups all business service dependencies
type DeleteGroupAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// DeleteGroupAttributeUseCase handles the business logic for deleting group attributes
type DeleteGroupAttributeUseCase struct {
	repositories DeleteGroupAttributeRepositories
	services     DeleteGroupAttributeServices
}

// NewDeleteGroupAttributeUseCase creates use case with grouped dependencies
func NewDeleteGroupAttributeUseCase(
	repositories DeleteGroupAttributeRepositories,
	services DeleteGroupAttributeServices,
) *DeleteGroupAttributeUseCase {
	return &DeleteGroupAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteGroupAttributeUseCaseUngrouped creates a new DeleteGroupAttributeUseCase
// Deprecated: Use NewDeleteGroupAttributeUseCase with grouped parameters instead
func NewDeleteGroupAttributeUseCaseUngrouped(groupAttributeRepo groupattributepb.GroupAttributeDomainServiceServer) *DeleteGroupAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteGroupAttributeRepositories{
		GroupAttribute: groupAttributeRepo,
	}

	services := DeleteGroupAttributeServices{
		AuthorizationService: nil,
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewDeleteGroupAttributeUseCase(repositories, services)
}

// Execute performs the delete group attribute operation
func (uc *DeleteGroupAttributeUseCase) Execute(ctx context.Context, req *groupattributepb.DeleteGroupAttributeRequest) (*groupattributepb.DeleteGroupAttributeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityGroupAttribute, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.GroupAttribute.DeleteGroupAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.deletion_failed", "Group attribute deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteGroupAttributeUseCase) validateInput(ctx context.Context, req *groupattributepb.DeleteGroupAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.request_required", "Request is required for group attributes [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.data_required", "Group attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.id_required", "Group attribute ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeleteGroupAttributeUseCase) validateBusinessRules(ctx context.Context, req *groupattributepb.DeleteGroupAttributeRequest) error {
	// TODO: Additional business rules
	// Example: Check if attribute is required and cannot be deleted
	// Example: Check permissions for deleting this attribute
	// Example: Validate cascading effects
	// For now, allow all deletions

	return nil
}
