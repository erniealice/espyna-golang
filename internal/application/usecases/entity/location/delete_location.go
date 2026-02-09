package location

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
)

// DeleteLocationRepositories groups all repository dependencies
type DeleteLocationRepositories struct {
	Location locationpb.LocationDomainServiceServer // Primary entity repository
}

// DeleteLocationServices groups all business service dependencies
type DeleteLocationServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteLocationUseCase handles the business logic for deleting locations
type DeleteLocationUseCase struct {
	repositories DeleteLocationRepositories
	services     DeleteLocationServices
}

// NewDeleteLocationUseCase creates use case with grouped dependencies
func NewDeleteLocationUseCase(
	repositories DeleteLocationRepositories,
	services DeleteLocationServices,
) *DeleteLocationUseCase {
	return &DeleteLocationUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete location operation
func (uc *DeleteLocationUseCase) Execute(ctx context.Context, req *locationpb.DeleteLocationRequest) (*locationpb.DeleteLocationResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityLocation, ports.ActionDelete); err != nil {
		return nil, err
	}

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
	resp, err := uc.repositories.Location.DeleteLocation(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.errors.deletion_failed", "")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteLocationUseCase) validateInput(ctx context.Context, req *locationpb.DeleteLocationRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.request_required", ""))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.data_required", ""))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "location.validation.id_required", ""))
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeleteLocationUseCase) validateBusinessRules(ctx context.Context, req *locationpb.DeleteLocationRequest) error {
	// TODO: Add business rules for location deletion
	// Example: Check if location has associated events, staff, or other resources
	// For now, allow all deletions

	return nil
}
