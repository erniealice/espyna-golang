package collection_method

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
)

// GetCollectionMethodItemPageDataRepositories groups all repository dependencies
type GetCollectionMethodItemPageDataRepositories struct {
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer // Primary entity repository
}

// GetCollectionMethodItemPageDataServices groups all business service dependencies
type GetCollectionMethodItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetCollectionMethodItemPageDataUseCase handles the business logic for getting collection method item page data
type GetCollectionMethodItemPageDataUseCase struct {
	repositories GetCollectionMethodItemPageDataRepositories
	services     GetCollectionMethodItemPageDataServices
}

// NewGetCollectionMethodItemPageDataUseCase creates use case with grouped dependencies
func NewGetCollectionMethodItemPageDataUseCase(
	repositories GetCollectionMethodItemPageDataRepositories,
	services GetCollectionMethodItemPageDataServices,
) *GetCollectionMethodItemPageDataUseCase {
	return &GetCollectionMethodItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get collection method item page data operation
func (uc *GetCollectionMethodItemPageDataUseCase) Execute(ctx context.Context, req *collectionmethodpb.GetCollectionMethodItemPageDataRequest) (*collectionmethodpb.GetCollectionMethodItemPageDataResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.CollectionMethod,
		Action: entityid.ActionList,
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
	resp, err := uc.repositories.CollectionMethod.GetCollectionMethodItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load collection method details")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetCollectionMethodItemPageDataUseCase) validateInput(ctx context.Context, req *collectionmethodpb.GetCollectionMethodItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	// Validate collection method ID
	if req.CollectionMethodId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.collection_method_id_required", "[ERR-DEFAULT] Collection method ID is required"))
	}

	// Basic ID format validation
	if len(req.CollectionMethodId) < 3 || len(req.CollectionMethodId) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.invalid_collection_method_id_format", "[ERR-DEFAULT] Invalid collection method ID format"))
	}

	// Ensure ID doesn't contain invalid characters
	if strings.ContainsAny(req.CollectionMethodId, " \t\n\r") {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.collection_method_id_invalid_characters", "[ERR-DEFAULT] Collection method ID contains invalid characters"))
	}

	return nil
}

// validateBusinessRules enforces business constraints for getting item page data
func (uc *GetCollectionMethodItemPageDataUseCase) validateBusinessRules(ctx context.Context, req *collectionmethodpb.GetCollectionMethodItemPageDataRequest) error {
	// For now, we'll allow all authenticated users to view collection method details
	return nil
}
