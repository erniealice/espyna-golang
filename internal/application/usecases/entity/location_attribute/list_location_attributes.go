package location_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	locationattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_attribute"
)

// ListLocationAttributesRepositories groups all repository dependencies
type ListLocationAttributesRepositories struct {
	LocationAttribute locationattributepb.LocationAttributeDomainServiceServer // Primary entity repository
}

// ListLocationAttributesServices groups all business service dependencies
type ListLocationAttributesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService // Current: Database transactions
	TranslationService ports.TranslationService
}

// ListLocationAttributesUseCase handles the business logic for listing location attributes
type ListLocationAttributesUseCase struct {
	repositories ListLocationAttributesRepositories
	services     ListLocationAttributesServices
}

// NewListLocationAttributesUseCase creates a new ListLocationAttributesUseCase
func NewListLocationAttributesUseCase(
	repositories ListLocationAttributesRepositories,
	services ListLocationAttributesServices,
) *ListLocationAttributesUseCase {
	return &ListLocationAttributesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListLocationAttributesUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListLocationAttributesUseCase with grouped parameters instead
func NewListLocationAttributesUseCaseUngrouped(
	locationAttributeRepo locationattributepb.LocationAttributeDomainServiceServer,
) *ListLocationAttributesUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListLocationAttributesRepositories{
		LocationAttribute: locationAttributeRepo,
	}

	services := ListLocationAttributesServices{
		AuthorizationService: nil,
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewListLocationAttributesUseCase(repositories, services)
}

func (uc *ListLocationAttributesUseCase) Execute(ctx context.Context, req *locationattributepb.ListLocationAttributesRequest) (*locationattributepb.ListLocationAttributesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityLocationAttribute, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.LocationAttribute.ListLocationAttributes(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.errors.list_failed", "Failed to retrieve location attributes [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListLocationAttributesUseCase) validateInput(ctx context.Context, req *locationattributepb.ListLocationAttributesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location_attribute.validation.request_required", "Request is required for location attributes [DEFAULT]"))
	}
	// List requests typically don't require additional validation beyond the request itself
	return nil
}
