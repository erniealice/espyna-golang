package collection

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection"
)

// ListCollectionsRepositories groups all repository dependencies
type ListCollectionsRepositories struct {
	Collection collectionpb.CollectionDomainServiceServer // Primary entity repository
}

// ListCollectionsServices groups all business service dependencies
type ListCollectionsServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ListCollectionsUseCase handles the business logic for listing collections
type ListCollectionsUseCase struct {
	repositories ListCollectionsRepositories
	services     ListCollectionsServices
}

// NewListCollectionsUseCase creates use case with grouped dependencies
func NewListCollectionsUseCase(
	repositories ListCollectionsRepositories,
	services ListCollectionsServices,
) *ListCollectionsUseCase {
	return &ListCollectionsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list collections operation
func (uc *ListCollectionsUseCase) Execute(ctx context.Context, req *collectionpb.ListCollectionsRequest) (*collectionpb.ListCollectionsResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.errors.authorization_failed", "Authorization failed for collections [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityCollection, ports.ActionList)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.errors.authorization_failed", "Authorization failed for collections [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.errors.authorization_failed", "Authorization failed for collections [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Collection.ListCollections(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.errors.list_failed", "Failed to retrieve collections [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic post-processing (if needed)
	// For example: filtering, sorting, or enriching the results

	return resp, nil
}

// validateInput validates the input request
func (uc *ListCollectionsUseCase) validateInput(ctx context.Context, req *collectionpb.ListCollectionsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection.validation.request_required", "Request is required for collections [DEFAULT]"))
	}
	return nil
}
