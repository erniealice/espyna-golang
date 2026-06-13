package collection_method

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
)

// DeleteCollectionMethodRepositories groups all repository dependencies
type DeleteCollectionMethodRepositories struct {
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer // Primary entity repository
}

// DeleteCollectionMethodServices groups all business service dependencies
type DeleteCollectionMethodServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteCollectionMethodUseCase handles the business logic for deleting collection methods
type DeleteCollectionMethodUseCase struct {
	repositories DeleteCollectionMethodRepositories
	services     DeleteCollectionMethodServices
}

// NewDeleteCollectionMethodUseCase creates use case with grouped dependencies
func NewDeleteCollectionMethodUseCase(
	repositories DeleteCollectionMethodRepositories,
	services DeleteCollectionMethodServices,
) *DeleteCollectionMethodUseCase {
	return &DeleteCollectionMethodUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete collection method operation
func (uc *DeleteCollectionMethodUseCase) Execute(ctx context.Context, req *collectionmethodpb.DeleteCollectionMethodRequest) (*collectionmethodpb.DeleteCollectionMethodResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.CollectionMethod,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.CollectionMethod.DeleteCollectionMethod(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.errors.deletion_failed", "[ERR-DEFAULT] Collection method deletion failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteCollectionMethodUseCase) validateInput(ctx context.Context, req *collectionmethodpb.DeleteCollectionMethodRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for deletion
func (uc *DeleteCollectionMethodUseCase) validateBusinessRules(ctx context.Context, req *collectionmethodpb.DeleteCollectionMethodRequest) error {
	// TODO: Add business rules for collection method deletion
	// Example: Check if collection method is referenced by revenue payments
	// For now, allow all deletions

	return nil
}
