package collection_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	collectionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_attribute"
)

// ListCollectionAttributesUseCase handles the business logic for listing product attributes
// ListCollectionAttributesRepositories groups all repository dependencies
type ListCollectionAttributesRepositories struct {
	CollectionAttribute collectionattributepb.CollectionAttributeDomainServiceServer // Primary entity repository
}

// ListCollectionAttributesServices groups all business service dependencies
type ListCollectionAttributesServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.CollectionAttribute,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.CollectionAttribute, entityid.ActionList)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_attribute.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.CollectionAttribute.ListCollectionAttributes(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_attribute.errors.list_failed", "Failed to retrieve product attributes [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListCollectionAttributesUseCase) validateInput(ctx context.Context, req *collectionattributepb.ListCollectionAttributesRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	// Additional validation can be added here if needed
	return nil
}
