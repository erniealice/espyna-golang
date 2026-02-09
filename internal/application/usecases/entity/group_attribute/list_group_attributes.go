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

// ListGroupAttributesUseCase handles the business logic for listing group attributes
// ListGroupAttributesRepositories groups all repository dependencies
type ListGroupAttributesRepositories struct {
	GroupAttribute groupattributepb.GroupAttributeDomainServiceServer // Primary entity repository
}

// ListGroupAttributesServices groups all business service dependencies
type ListGroupAttributesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// ListGroupAttributesUseCase handles the business logic for listing group attributes
type ListGroupAttributesUseCase struct {
	repositories ListGroupAttributesRepositories
	services     ListGroupAttributesServices
}

// NewListGroupAttributesUseCase creates use case with grouped dependencies
func NewListGroupAttributesUseCase(
	repositories ListGroupAttributesRepositories,
	services ListGroupAttributesServices,
) *ListGroupAttributesUseCase {
	return &ListGroupAttributesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListGroupAttributesUseCaseUngrouped creates a new ListGroupAttributesUseCase
// Deprecated: Use NewListGroupAttributesUseCase with grouped parameters instead
func NewListGroupAttributesUseCaseUngrouped(groupAttributeRepo groupattributepb.GroupAttributeDomainServiceServer) *ListGroupAttributesUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListGroupAttributesRepositories{
		GroupAttribute: groupAttributeRepo,
	}

	services := ListGroupAttributesServices{
		AuthorizationService: nil,
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewListGroupAttributesUseCase(repositories, services)
}

// Execute performs the list group attributes operation
func (uc *ListGroupAttributesUseCase) Execute(ctx context.Context, req *groupattributepb.ListGroupAttributesRequest) (*groupattributepb.ListGroupAttributesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityGroupAttribute, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Apply default values for pagination
	if err := uc.applyDefaults(req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.apply_defaults_failed", "Failed to apply default values [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.GroupAttribute.ListGroupAttributes(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.errors.list_failed", "Failed to retrieve group attributes [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListGroupAttributesUseCase) validateInput(ctx context.Context, req *groupattributepb.ListGroupAttributesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "group_attribute.validation.request_required", "Request is required for group attributes [DEFAULT]"))
	}

	// No additional business rules for listing group attributes
	// Pagination is not supported in current protobuf definition

	return nil
}

// applyDefaults sets default values for optional parameters
func (uc *ListGroupAttributesUseCase) applyDefaults(req *groupattributepb.ListGroupAttributesRequest) error {
	// No defaults to apply
	// Pagination is not supported in current protobuf definition
	return nil
}
