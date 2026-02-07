package location

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	locationpb "leapfor.xyz/esqyma/golang/v1/domain/entity/location"
)

// ListLocationsRepositories groups all repository dependencies
type ListLocationsRepositories struct {
	Location locationpb.LocationDomainServiceServer // Primary entity repository
}

// ListLocationsServices groups all business service dependencies
type ListLocationsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListLocationsUseCase handles the business logic for listing locations
type ListLocationsUseCase struct {
	repositories ListLocationsRepositories
	services     ListLocationsServices
}

// NewListLocationsUseCase creates use case with grouped dependencies
func NewListLocationsUseCase(
	repositories ListLocationsRepositories,
	services ListLocationsServices,
) *ListLocationsUseCase {
	return &ListLocationsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list locations operation
func (uc *ListLocationsUseCase) Execute(ctx context.Context, req *locationpb.ListLocationsRequest) (*locationpb.ListLocationsResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.errors.input_validation_failed", "")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.errors.business_rule_validation_failed", "")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Location.ListLocations(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.errors.list_failed", "")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListLocationsUseCase) validateInput(ctx context.Context, req *locationpb.ListLocationsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.request_required", ""))
	}
	return nil
}

// validateBusinessRules enforces business constraints for listing
func (uc *ListLocationsUseCase) validateBusinessRules(ctx context.Context, req *locationpb.ListLocationsRequest) error {
	// No additional business rules for listing locations
	// Pagination is not supported in current protobuf definition
	return nil
}
