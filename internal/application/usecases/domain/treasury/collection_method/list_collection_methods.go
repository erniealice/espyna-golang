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

// ListCollectionMethodsRepositories groups all repository dependencies
type ListCollectionMethodsRepositories struct {
	CollectionMethod collectionmethodpb.CollectionMethodDomainServiceServer // Primary entity repository
}

// ListCollectionMethodsServices groups all business service dependencies
type ListCollectionMethodsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListCollectionMethodsUseCase handles the business logic for listing collection methods
type ListCollectionMethodsUseCase struct {
	repositories ListCollectionMethodsRepositories
	services     ListCollectionMethodsServices
}

// NewListCollectionMethodsUseCase creates use case with grouped dependencies
func NewListCollectionMethodsUseCase(
	repositories ListCollectionMethodsRepositories,
	services ListCollectionMethodsServices,
) *ListCollectionMethodsUseCase {
	return &ListCollectionMethodsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list collection methods operation
func (uc *ListCollectionMethodsUseCase) Execute(ctx context.Context, req *collectionmethodpb.ListCollectionMethodsRequest) (*collectionmethodpb.ListCollectionMethodsResponse, error) {
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
	resp, err := uc.repositories.CollectionMethod.ListCollectionMethods(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.errors.list_failed", "[ERR-DEFAULT] Failed to list collection methods")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListCollectionMethodsUseCase) validateInput(ctx context.Context, req *collectionmethodpb.ListCollectionMethodsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_method.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for listing
func (uc *ListCollectionMethodsUseCase) validateBusinessRules(ctx context.Context, req *collectionmethodpb.ListCollectionMethodsRequest) error {
	// No additional business rules for listing collection methods
	return nil
}
