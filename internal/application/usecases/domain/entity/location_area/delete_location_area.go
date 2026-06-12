package location_area

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	locationareapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_area"
)

// DeleteLocationAreaRepositories groups all repository dependencies
type DeleteLocationAreaRepositories struct {
	LocationArea locationareapb.LocationAreaDomainServiceServer // Primary entity repository
}

// DeleteLocationAreaServices groups all business service dependencies
type DeleteLocationAreaServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// DeleteLocationAreaUseCase handles the business logic for deleting location areas
type DeleteLocationAreaUseCase struct {
	repositories DeleteLocationAreaRepositories
	services     DeleteLocationAreaServices
}

// NewDeleteLocationAreaUseCase creates use case with grouped dependencies
func NewDeleteLocationAreaUseCase(
	repositories DeleteLocationAreaRepositories,
	services DeleteLocationAreaServices,
) *DeleteLocationAreaUseCase {
	return &DeleteLocationAreaUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete location area operation
func (uc *DeleteLocationAreaUseCase) Execute(ctx context.Context, req *locationareapb.DeleteLocationAreaRequest) (*locationareapb.DeleteLocationAreaResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.LocationArea, entityid.ActionDelete); err != nil {
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
	resp, err := uc.repositories.LocationArea.DeleteLocationArea(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.errors.deletion_failed", "[ERR-DEFAULT] Location area deletion failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteLocationAreaUseCase) validateInput(ctx context.Context, req *locationareapb.DeleteLocationAreaRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeleteLocationAreaUseCase) validateBusinessRules(ctx context.Context, req *locationareapb.DeleteLocationAreaRequest) error {
	// TODO: Add business rules for location area deletion
	// Example: Check if location area has associated locations or other resources
	// For now, allow all deletions

	return nil
}
