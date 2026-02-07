package collection_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	collectionattributepb "leapfor.xyz/esqyma/golang/v1/domain/product/collection_attribute"
)

// ReadCollectionAttributeUseCase handles the business logic for reading a product attribute
// ReadCollectionAttributeRepositories groups all repository dependencies
type ReadCollectionAttributeRepositories struct {
	CollectionAttribute collectionattributepb.CollectionAttributeDomainServiceServer // Primary entity repository
}

// ReadCollectionAttributeServices groups all business service dependencies
type ReadCollectionAttributeServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// ReadCollectionAttributeUseCase handles the business logic for reading a product attribute
type ReadCollectionAttributeUseCase struct {
	repositories ReadCollectionAttributeRepositories
	services     ReadCollectionAttributeServices
}

// NewReadCollectionAttributeUseCase creates a new ReadCollectionAttributeUseCase
func NewReadCollectionAttributeUseCase(
	repositories ReadCollectionAttributeRepositories,
	services ReadCollectionAttributeServices,
) *ReadCollectionAttributeUseCase {
	return &ReadCollectionAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read product attribute operation
func (uc *ReadCollectionAttributeUseCase) Execute(ctx context.Context, req *collectionattributepb.ReadCollectionAttributeRequest) (*collectionattributepb.ReadCollectionAttributeResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityCollectionAttribute, ports.ActionRead)
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
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.CollectionAttribute.ReadCollectionAttribute(ctx, req)
	if err != nil {
		// Check if it's a not found error and convert to translated message
		if strings.Contains(err.Error(), "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "collection_attribute.errors.not_found", map[string]interface{}{"productAttributeId": req.Data.Id}, "Collection attribute not found")
			return nil, errors.New(translatedError)
		}
		// Other repository errors
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.read_failed", "Failed to read product attribute")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadCollectionAttributeUseCase) validateInput(ctx context.Context, req *collectionattributepb.ReadCollectionAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.request_required", "request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.data_required", "product attribute data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.id_required", "product attribute ID is required"))
	}
	return nil
}
