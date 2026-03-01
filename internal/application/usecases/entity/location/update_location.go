package location

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
)

// UpdateLocationRepositories groups all repository dependencies
type UpdateLocationRepositories struct {
	Location locationpb.LocationDomainServiceServer // Primary entity repository
}

// UpdateLocationServices groups all business service dependencies
type UpdateLocationServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateLocationUseCase handles the business logic for updating locations
type UpdateLocationUseCase struct {
	repositories UpdateLocationRepositories
	services     UpdateLocationServices
}

// NewUpdateLocationUseCase creates use case with grouped dependencies
func NewUpdateLocationUseCase(
	repositories UpdateLocationRepositories,
	services UpdateLocationServices,
) *UpdateLocationUseCase {
	return &UpdateLocationUseCase{
		repositories: repositories,
		services:     services,
	}
}

func (uc *UpdateLocationUseCase) Execute(ctx context.Context, req *locationpb.UpdateLocationRequest) (*locationpb.UpdateLocationResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityLocation, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichLocationData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.errors.enrichment_failed", "[ERR-DEFAULT] Data enrichment failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Location.UpdateLocation(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.errors.update_failed", "[ERR-DEFAULT] Location update failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateLocationUseCase) validateInput(ctx context.Context, req *locationpb.UpdateLocationRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.data_required", "[ERR-DEFAULT] Location data is required"))
	}

	// Trim leading and trailing spaces
	req.Data.Name = strings.TrimSpace(req.Data.Name)
	req.Data.Address = strings.TrimSpace(req.Data.Address)
	if req.Data.Description != nil {
		trimmed := strings.TrimSpace(*req.Data.Description)
		req.Data.Description = &trimmed
	}

	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.id_required", "[ERR-DEFAULT] Location ID is required"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.name_required", "[ERR-DEFAULT] Name is required"))
	}
	if req.Data.Address == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.address_required", "[ERR-DEFAULT] Address is required"))
	}
	return nil
}

// enrichLocationData adds audit information for updates
func (uc *UpdateLocationUseCase) enrichLocationData(location *locationpb.Location) error {
	now := time.Now()

	// Set location audit fields for modification
	location.DateModified = &[]int64{now.UnixMilli()}[0]
	location.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateLocationUseCase) validateBusinessRules(ctx context.Context, location *locationpb.Location) error {
	// Validate name length
	if len(location.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.name_too_long", "[ERR-DEFAULT] Name must not exceed 100 characters"))
	}

	// Validate address length
	if len(location.Address) > 500 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.address_too_long", "[ERR-DEFAULT] Address must not exceed 500 characters"))
	}

	// Validate description length if provided
	if location.Description != nil && len(*location.Description) > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.description_too_long", "[ERR-DEFAULT] Description must not exceed 1000 characters"))
	}

	return nil
}
