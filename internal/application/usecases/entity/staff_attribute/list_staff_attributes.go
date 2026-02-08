package staff_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	staffattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff_attribute"
)

// ListStaffAttributesUseCase handles the business logic for listing staff attributes
// ListStaffAttributesRepositories groups all repository dependencies
type ListStaffAttributesRepositories struct {
	StaffAttribute staffattributepb.StaffAttributeDomainServiceServer // Primary entity repository
}

// ListStaffAttributesServices groups all business service dependencies
type ListStaffAttributesServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// ListStaffAttributesUseCase handles the business logic for listing staff attributes
type ListStaffAttributesUseCase struct {
	repositories ListStaffAttributesRepositories
	services     ListStaffAttributesServices
}

// NewListStaffAttributesUseCase creates use case with grouped dependencies
func NewListStaffAttributesUseCase(
	repositories ListStaffAttributesRepositories,
	services ListStaffAttributesServices,
) *ListStaffAttributesUseCase {
	return &ListStaffAttributesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListStaffAttributesUseCaseUngrouped creates a new ListStaffAttributesUseCase
// Deprecated: Use NewListStaffAttributesUseCase with grouped parameters instead
func NewListStaffAttributesUseCaseUngrouped(staffAttributeRepo staffattributepb.StaffAttributeDomainServiceServer) *ListStaffAttributesUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListStaffAttributesRepositories{
		StaffAttribute: staffAttributeRepo,
	}

	services := ListStaffAttributesServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewListStaffAttributesUseCase(repositories, services)
}

// Execute performs the list staff attributes operation
func (uc *ListStaffAttributesUseCase) Execute(ctx context.Context, req *staffattributepb.ListStaffAttributesRequest) (*staffattributepb.ListStaffAttributesResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Apply default values for pagination
	if err := uc.applyDefaults(req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.errors.apply_defaults_failed", "Failed to apply default values [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.StaffAttribute.ListStaffAttributes(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.errors.list_failed", "Failed to retrieve staff attributes [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListStaffAttributesUseCase) validateInput(ctx context.Context, req *staffattributepb.ListStaffAttributesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff_attribute.validation.request_required", "Request is required for staff attributes [DEFAULT]"))
	}

	// No additional business rules for listing staff attributes
	// Pagination is not supported in current protobuf definition

	return nil
}

// applyDefaults sets default values for optional parameters
func (uc *ListStaffAttributesUseCase) applyDefaults(req *staffattributepb.ListStaffAttributesRequest) error {
	// No defaults to apply
	// Pagination is not supported in current protobuf definition
	return nil
}
