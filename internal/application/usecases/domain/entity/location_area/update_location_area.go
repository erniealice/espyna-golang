package location_area

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	locationareapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_area"
)

// UpdateLocationAreaRepositories groups all repository dependencies
type UpdateLocationAreaRepositories struct {
	LocationArea locationareapb.LocationAreaDomainServiceServer // Primary entity repository
}

// UpdateLocationAreaServices groups all business service dependencies
type UpdateLocationAreaServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateLocationAreaUseCase handles the business logic for updating location areas
type UpdateLocationAreaUseCase struct {
	repositories UpdateLocationAreaRepositories
	services     UpdateLocationAreaServices
}

// NewUpdateLocationAreaUseCase creates use case with grouped dependencies
func NewUpdateLocationAreaUseCase(
	repositories UpdateLocationAreaRepositories,
	services UpdateLocationAreaServices,
) *UpdateLocationAreaUseCase {
	return &UpdateLocationAreaUseCase{
		repositories: repositories,
		services:     services,
	}
}

func (uc *UpdateLocationAreaUseCase) Execute(ctx context.Context, req *locationareapb.UpdateLocationAreaRequest) (*locationareapb.UpdateLocationAreaResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.LocationArea,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichLocationAreaData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.errors.enrichment_failed", "[ERR-DEFAULT] Data enrichment failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.LocationArea.UpdateLocationArea(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.errors.update_failed", "[ERR-DEFAULT] Location area update failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateLocationAreaUseCase) validateInput(ctx context.Context, req *locationareapb.UpdateLocationAreaRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.data_required", "[ERR-DEFAULT] Location area data is required"))
	}

	// Trim leading and trailing spaces
	req.Data.Name = strings.TrimSpace(req.Data.Name)
	req.Data.Description = strings.TrimSpace(req.Data.Description)

	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.id_required", "[ERR-DEFAULT] Location area ID is required"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.name_required", "[ERR-DEFAULT] Name is required"))
	}
	return nil
}

// enrichLocationAreaData adds audit information for updates
func (uc *UpdateLocationAreaUseCase) enrichLocationAreaData(locationArea *locationareapb.LocationArea) error {
	now := time.Now()

	// Set location area audit fields for modification
	locationArea.DateModified = &[]int64{now.UnixMilli()}[0]
	locationArea.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateLocationAreaUseCase) validateBusinessRules(ctx context.Context, locationArea *locationareapb.LocationArea) error {
	// Validate name length
	if len(locationArea.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.name_too_long", "[ERR-DEFAULT] Name must not exceed 100 characters"))
	}

	// Validate description length if provided
	if len(locationArea.Description) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.description_too_long", "[ERR-DEFAULT] Description must not exceed 1000 characters"))
	}

	return nil
}
