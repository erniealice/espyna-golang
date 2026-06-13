package location_attribute

import (
	"context"
	"errors"
	"fmt" // Add fmt import

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	locationattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_attribute"
)

// DeleteLocationAttributeRepositories groups all repository dependencies
type DeleteLocationAttributeRepositories struct {
	LocationAttribute locationattributepb.LocationAttributeDomainServiceServer
}

// DeleteLocationAttributeServices groups all business service dependencies
type DeleteLocationAttributeServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteLocationAttributeUseCase handles the business logic for deleting location attributes
type DeleteLocationAttributeUseCase struct {
	repositories DeleteLocationAttributeRepositories // Changed
	services     DeleteLocationAttributeServices     // Changed
}

// NewDeleteLocationAttributeUseCase creates a new DeleteLocationAttributeUseCase
func NewDeleteLocationAttributeUseCase(
	repositories DeleteLocationAttributeRepositories, // Changed
	services DeleteLocationAttributeServices, // Changed
) *DeleteLocationAttributeUseCase {
	return &DeleteLocationAttributeUseCase{
		repositories: repositories, // Changed
		services:     services,     // Changed
	}
}

// NewDeleteLocationAttributeUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteLocationAttributeUseCase with grouped parameters instead
// NewDeleteLocationAttributeUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteLocationAttributeUseCase with grouped parameters instead
func NewDeleteLocationAttributeUseCaseUngrouped(
	locationAttributeRepo locationattributepb.LocationAttributeDomainServiceServer,
) *DeleteLocationAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteLocationAttributeRepositories{
		LocationAttribute: locationAttributeRepo,
	}

	services := DeleteLocationAttributeServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewDeleteLocationAttributeUseCase(repositories, services)
}

func (uc *DeleteLocationAttributeUseCase) Execute(ctx context.Context, req *locationattributepb.DeleteLocationAttributeRequest) (*locationattributepb.DeleteLocationAttributeResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.LocationAttribute,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes location attribute deletion within a transaction
func (uc *DeleteLocationAttributeUseCase) executeWithTransaction(ctx context.Context, req *locationattributepb.DeleteLocationAttributeRequest) (*locationattributepb.DeleteLocationAttributeResponse, error) {
	var result *locationattributepb.DeleteLocationAttributeResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "location_attribute.errors.deletion_failed", "Location attribute deletion failed [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic (moved from original Execute method)
func (uc *DeleteLocationAttributeUseCase) executeCore(ctx context.Context, req *locationattributepb.DeleteLocationAttributeRequest) (*locationattributepb.DeleteLocationAttributeResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_attribute.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.LocationAttribute.DeleteLocationAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_attribute.errors.deletion_failed", "Location attribute deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteLocationAttributeUseCase) validateInput(ctx context.Context, req *locationattributepb.DeleteLocationAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_attribute.validation.request_required", "Request is required for location attributes [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_attribute.validation.data_required", "Location attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "location_attribute.validation.id_required", "Location attribute ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeleteLocationAttributeUseCase) validateBusinessRules(ctx context.Context, req *locationattributepb.DeleteLocationAttributeRequest) error {
	// TODO: Additional business rules
	// Example: Check if attribute is required and cannot be deleted
	// Example: Check permissions for deleting this attribute
	// Example: Validate cascading effects
	// For now, allow all deletions

	return nil
}
