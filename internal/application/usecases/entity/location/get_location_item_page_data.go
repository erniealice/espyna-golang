package location

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
)

// GetLocationItemPageDataRepositories groups all repository dependencies
type GetLocationItemPageDataRepositories struct {
	Location locationpb.LocationDomainServiceServer // Primary entity repository
}

// GetLocationItemPageDataServices groups all business service dependencies
type GetLocationItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetLocationItemPageDataUseCase handles the business logic for getting location item page data
type GetLocationItemPageDataUseCase struct {
	repositories GetLocationItemPageDataRepositories
	services     GetLocationItemPageDataServices
}

// NewGetLocationItemPageDataUseCase creates use case with grouped dependencies
func NewGetLocationItemPageDataUseCase(
	repositories GetLocationItemPageDataRepositories,
	services GetLocationItemPageDataServices,
) *GetLocationItemPageDataUseCase {
	return &GetLocationItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get location item page data operation
func (uc *GetLocationItemPageDataUseCase) Execute(ctx context.Context, req *locationpb.GetLocationItemPageDataRequest) (*locationpb.GetLocationItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityLocation, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Location.GetLocationItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load location details")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetLocationItemPageDataUseCase) validateInput(ctx context.Context, req *locationpb.GetLocationItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	// Validate location ID
	if req.LocationId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.location_id_required", "[ERR-DEFAULT] Location ID is required"))
	}

	// Basic ID format validation
	if len(req.LocationId) < 3 || len(req.LocationId) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.invalid_location_id_format", "[ERR-DEFAULT] Invalid location ID format"))
	}

	// Ensure ID doesn't contain invalid characters
	if strings.ContainsAny(req.LocationId, " \t\n\r") {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.location_id_invalid_characters", "[ERR-DEFAULT] Location ID contains invalid characters"))
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting item page data
func (uc *GetLocationItemPageDataUseCase) validateBusinessRules(ctx context.Context, req *locationpb.GetLocationItemPageDataRequest) error {
	// Check authorization for viewing specific location
	// This would typically involve checking user permissions for the specific location
	// For now, we'll allow all authenticated users to view location details

	return nil
}
