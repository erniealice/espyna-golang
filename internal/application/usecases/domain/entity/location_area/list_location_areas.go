package location_area

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	locationareapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_area"
)

// ListLocationAreasRepositories groups all repository dependencies
type ListLocationAreasRepositories struct {
	LocationArea locationareapb.LocationAreaDomainServiceServer // Primary entity repository
}

// ListLocationAreasServices groups all business service dependencies
type ListLocationAreasServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListLocationAreasUseCase handles the business logic for listing location areas
type ListLocationAreasUseCase struct {
	repositories ListLocationAreasRepositories
	services     ListLocationAreasServices
}

// NewListLocationAreasUseCase creates use case with grouped dependencies
func NewListLocationAreasUseCase(
	repositories ListLocationAreasRepositories,
	services ListLocationAreasServices,
) *ListLocationAreasUseCase {
	return &ListLocationAreasUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list location areas operation
func (uc *ListLocationAreasUseCase) Execute(ctx context.Context, req *locationareapb.ListLocationAreasRequest) (*locationareapb.ListLocationAreasResponse, error) {
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
	resp, err := uc.repositories.LocationArea.ListLocationAreas(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.errors.list_failed", "[ERR-DEFAULT] Failed to list location areas")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListLocationAreasUseCase) validateInput(ctx context.Context, req *locationareapb.ListLocationAreasRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_area.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for listing
func (uc *ListLocationAreasUseCase) validateBusinessRules(ctx context.Context, req *locationareapb.ListLocationAreasRequest) error {
	// No additional business rules for listing location areas
	return nil
}
