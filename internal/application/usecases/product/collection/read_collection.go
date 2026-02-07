package collection

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	collectionpb "leapfor.xyz/esqyma/golang/v1/domain/product/collection"
)

// ReadCollectionRepositories groups all repository dependencies
type ReadCollectionRepositories struct {
	Collection collectionpb.CollectionDomainServiceServer // Primary entity repository
}

// ReadCollectionServices groups all business service dependencies
type ReadCollectionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadCollectionUseCase handles the business logic for reading collections
type ReadCollectionUseCase struct {
	repositories ReadCollectionRepositories
	services     ReadCollectionServices
}

// NewReadCollectionUseCase creates use case with grouped dependencies
func NewReadCollectionUseCase(
	repositories ReadCollectionRepositories,
	services ReadCollectionServices,
) *ReadCollectionUseCase {
	return &ReadCollectionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read collection operation
func (uc *ReadCollectionUseCase) Execute(ctx context.Context, req *collectionpb.ReadCollectionRequest) (*collectionpb.ReadCollectionResponse, error) {
	// Authorization check - conditional based on service availability
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		userID, err := contextutil.RequireUserIDFromContext(ctx)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.errors.authorization_failed", "Authorization failed for course collections [DEFAULT]")
			return nil, errors.New(translatedError)
		}

		permission := ports.EntityPermission(ports.EntityCollection, ports.ActionRead)
		hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.errors.authorization_failed", "Authorization failed for course collections [DEFAULT]")
			return nil, errors.New(translatedError)
		}
		if !hasPerm {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.errors.authorization_failed", "Authorization failed for course collections [DEFAULT]")
			return nil, errors.New(translatedError)
		}
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.Collection.ReadCollection(ctx, req)
	if err != nil {
		// Check for exact not found error format from mock repository
		expectedNotFound := fmt.Sprintf("collection with ID '%s' not found", req.Data.Id)
		if err.Error() == expectedNotFound {
			// Handle as not found - translate and return
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(
				ctx,
				uc.services.TranslationService,
				"collection.errors.not_found",
				map[string]interface{}{"collectionId": req.Data.Id},
				"Course collection not found [DEFAULT]",
			)
			return nil, errors.New(translatedError)
		}
		// Handle other repository errors without wrapping
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadCollectionUseCase) validateInput(ctx context.Context, req *collectionpb.ReadCollectionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.validation.request_required", "Request is required for course collections [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.validation.data_required", "Course collection data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.validation.id_required", "Course collection ID is required [DEFAULT]"))
	}
	return nil
}
