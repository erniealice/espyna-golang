package collection_attribute

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	collectionattributepb "leapfor.xyz/esqyma/golang/v1/domain/product/collection_attribute"
)

// ListCollectionAttributesUseCase handles the business logic for listing product attributes
// ListCollectionAttributesRepositories groups all repository dependencies
type ListCollectionAttributesRepositories struct {
	CollectionAttribute collectionattributepb.CollectionAttributeDomainServiceServer // Primary entity repository
}

// ListCollectionAttributesServices groups all business service dependencies
type ListCollectionAttributesServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ListCollectionAttributesUseCase handles the business logic for listing product attributes
type ListCollectionAttributesUseCase struct {
	repositories ListCollectionAttributesRepositories
	services     ListCollectionAttributesServices
}

// NewListCollectionAttributesUseCase creates a new ListCollectionAttributesUseCase
func NewListCollectionAttributesUseCase(
	repositories ListCollectionAttributesRepositories,
	services ListCollectionAttributesServices,
) *ListCollectionAttributesUseCase {
	return &ListCollectionAttributesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list product attributes operation
func (uc *ListCollectionAttributesUseCase) Execute(ctx context.Context, req *collectionattributepb.ListCollectionAttributesRequest) (*collectionattributepb.ListCollectionAttributesResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityCollectionAttribute, ports.ActionList)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.CollectionAttribute.ListCollectionAttributes(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.list_failed", "Failed to retrieve product attributes [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListCollectionAttributesUseCase) validateInput(ctx context.Context, req *collectionattributepb.ListCollectionAttributesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	// Additional validation can be added here if needed
	return nil
}
