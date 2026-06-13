package location_area

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	locationareapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_area"
)

// GetLocationAreaItemPageDataRepositories groups all repository dependencies
type GetLocationAreaItemPageDataRepositories struct {
	LocationArea locationareapb.LocationAreaDomainServiceServer // Primary entity repository
}

// GetLocationAreaItemPageDataServices groups all business service dependencies
type GetLocationAreaItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetLocationAreaItemPageDataUseCase handles the business logic for getting location area item page data
type GetLocationAreaItemPageDataUseCase struct {
	repositories GetLocationAreaItemPageDataRepositories
	services     GetLocationAreaItemPageDataServices
}

// NewGetLocationAreaItemPageDataUseCase creates use case with grouped dependencies
func NewGetLocationAreaItemPageDataUseCase(
	repositories GetLocationAreaItemPageDataRepositories,
	services GetLocationAreaItemPageDataServices,
) *GetLocationAreaItemPageDataUseCase {
	return &GetLocationAreaItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get location area item page data operation
func (uc *GetLocationAreaItemPageDataUseCase) Execute(ctx context.Context, req *locationareapb.GetLocationAreaItemPageDataRequest) (*locationareapb.GetLocationAreaItemPageDataResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.LocationArea,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.LocationArea.GetLocationAreaItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load location area details")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetLocationAreaItemPageDataUseCase) validateInput(ctx context.Context, req *locationareapb.GetLocationAreaItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	// Validate location area ID
	if req.LocationAreaId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.location_area_id_required", "[ERR-DEFAULT] Location area ID is required"))
	}

	// Basic ID format validation
	if len(req.LocationAreaId) < 3 || len(req.LocationAreaId) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.invalid_location_area_id_format", "[ERR-DEFAULT] Invalid location area ID format"))
	}

	// Ensure ID doesn't contain invalid characters
	if strings.ContainsAny(req.LocationAreaId, " \t\n\r") {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.location_area_id_invalid_characters", "[ERR-DEFAULT] Location area ID contains invalid characters"))
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting item page data
func (uc *GetLocationAreaItemPageDataUseCase) validateBusinessRules(ctx context.Context, req *locationareapb.GetLocationAreaItemPageDataRequest) error {
	// Check authorization for viewing specific location area
	// This would typically involve checking user permissions for the specific location area
	// For now, we'll allow all authenticated users to view location area details

	return nil
}
