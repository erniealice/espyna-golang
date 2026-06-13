package location

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
)

// ListLocationsRepositories groups all repository dependencies
type ListLocationsRepositories struct {
	Location locationpb.LocationDomainServiceServer // Primary entity repository
}

// ListLocationsServices groups all business service dependencies
type ListLocationsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Location,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Location.ListLocations(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location.errors.list_failed", "[ERR-DEFAULT] Failed to list locations")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListLocationsUseCase) validateInput(ctx context.Context, req *locationpb.ListLocationsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for listing
func (uc *ListLocationsUseCase) validateBusinessRules(ctx context.Context, req *locationpb.ListLocationsRequest) error {
	// No additional business rules for listing locations
	// Pagination is not supported in current protobuf definition
	return nil
}
