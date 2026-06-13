package collection_method

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
	collectionmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method"
)

// UpdateCollectionMethodRepositories groups all repository dependencies
type UpdateCollectionMethodRepositories struct {
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer // Primary entity repository
}

// UpdateCollectionMethodServices groups all business service dependencies
type UpdateCollectionMethodServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateCollectionMethodUseCase handles the business logic for updating collection methods
type UpdateCollectionMethodUseCase struct {
	repositories UpdateCollectionMethodRepositories
	services     UpdateCollectionMethodServices
}

// NewUpdateCollectionMethodUseCase creates use case with grouped dependencies
func NewUpdateCollectionMethodUseCase(
	repositories UpdateCollectionMethodRepositories,
	services UpdateCollectionMethodServices,
) *UpdateCollectionMethodUseCase {
	return &UpdateCollectionMethodUseCase{
		repositories: repositories,
		services:     services,
	}
}

func (uc *UpdateCollectionMethodUseCase) Execute(ctx context.Context, req *collectionmethodpb.UpdateCollectionMethodRequest) (*collectionmethodpb.UpdateCollectionMethodResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.CollectionMethod,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichCollectionMethodData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.errors.enrichment_failed", "[ERR-DEFAULT] Data enrichment failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.errors.business_rule_validation_failed", "[ERR-DEFAULT] Business rule validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.CollectionMethod.UpdateCollectionMethod(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.errors.update_failed", "[ERR-DEFAULT] Collection method update failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateCollectionMethodUseCase) validateInput(ctx context.Context, req *collectionmethodpb.UpdateCollectionMethodRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.data_required", "[ERR-DEFAULT] Collection method data is required"))
	}

	// Trim leading and trailing spaces
	req.Data.Name = strings.TrimSpace(req.Data.Name)
	if req.Data.ProviderName != nil {
		trimmed := strings.TrimSpace(*req.Data.ProviderName)
		req.Data.ProviderName = &trimmed
	}

	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.id_required", "[ERR-DEFAULT] Collection method ID is required"))
	}
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.name_required", "[ERR-DEFAULT] Name is required"))
	}
	return nil
}

// enrichCollectionMethodData adds audit information for updates
func (uc *UpdateCollectionMethodUseCase) enrichCollectionMethodData(collectionMethod *collectionmethodpb.CollectionMethod) error {
	now := time.Now()

	// Set audit fields for modification
	collectionMethod.DateModified = &[]int64{now.UnixMilli()}[0]
	collectionMethod.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateCollectionMethodUseCase) validateBusinessRules(ctx context.Context, collectionMethod *collectionmethodpb.CollectionMethod) error {
	// Validate name length
	if len(collectionMethod.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.name_too_long", "[ERR-DEFAULT] Name must not exceed 100 characters"))
	}

	return nil
}
