package collection

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection"
)

// DeleteCollectionRepositories groups all repository dependencies
type DeleteCollectionRepositories struct {
	Collection collectionpb.CollectionDomainServiceServer // Primary entity repository
}

// DeleteCollectionServices groups all business service dependencies
type DeleteCollectionServices struct {
	Authorizer ports.Authorizer // Current: RBAC and permissions
	Transactor ports.Transactor // Current: Database transactions
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteCollectionUseCase handles the business logic for deleting collections
type DeleteCollectionUseCase struct {
	repositories DeleteCollectionRepositories
	services     DeleteCollectionServices
}

// NewDeleteCollectionUseCase creates use case with grouped dependencies
func NewDeleteCollectionUseCase(
	repositories DeleteCollectionRepositories,
	services DeleteCollectionServices,
) *DeleteCollectionUseCase {
	return &DeleteCollectionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete collection operation
func (uc *DeleteCollectionUseCase) Execute(ctx context.Context, req *collectionpb.DeleteCollectionRequest) (*collectionpb.DeleteCollectionResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Collection,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes collection deletion within a transaction
func (uc *DeleteCollectionUseCase) executeWithTransaction(ctx context.Context, req *collectionpb.DeleteCollectionRequest) (*collectionpb.DeleteCollectionResponse, error) {
	var result *collectionpb.DeleteCollectionResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.errors.transaction_failed", "Transaction execution failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return result, nil
}

// executeCore contains the core business logic for deleting a collection
func (uc *DeleteCollectionUseCase) executeCore(ctx context.Context, req *collectionpb.DeleteCollectionRequest) (*collectionpb.DeleteCollectionResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.errors.authorization_failed", "Authorization failed for collections [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := entityid.EntityPermission(entityid.Collection, entityid.ActionDelete)
	hasPerm, err := uc.services.Authorizer.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.errors.authorization_failed", "Authorization failed for collections [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.errors.authorization_failed", "Authorization failed for collections [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.Collection.DeleteCollection(ctx, req)
	if err != nil {
		// Check if it's a not found error and convert to translated message
		if strings.Contains(err.Error(), "not found") {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.Translator, "collection.errors.not_found", map[string]interface{}{"collectionId": req.Data.Id}, "Course collection not found")
			return nil, errors.New(translatedError)
		}
		// Other error handling
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.errors.deletion_failed", "Course collection deletion failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteCollectionUseCase) validateInput(ctx context.Context, req *collectionpb.DeleteCollectionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.validation.request_required", "Request is required for collections [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.validation.data_required", "Collection data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.validation.id_required", "Collection ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints before deletion
func (uc *DeleteCollectionUseCase) validateBusinessRules(ctx context.Context, collection *collectionpb.Collection) error {
	// Check if collection has child collections
	if hasChildren, err := uc.hasChildCollections(ctx, collection.Id); err != nil {
		return err
	} else if hasChildren {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.validation.has_child_collections", "Cannot delete collection with child collections [DEFAULT]"))
	}

	// Check if collection is associated with products
	if hasProducts, err := uc.hasAssociatedProducts(ctx, collection.Id); err != nil {
		return err
	} else if hasProducts {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "collection.validation.has_associated_products", "Cannot delete collection with associated products [DEFAULT]"))
	}

	return nil
}

// hasChildCollections checks if the collection has any child collections
func (uc *DeleteCollectionUseCase) hasChildCollections(ctx context.Context, collectionID string) (bool, error) {
	// This would typically query the collection parent repository
	// For now, we'll return false as a placeholder
	// TODO: Implement actual check for child collections
	return false, nil
}

// hasAssociatedProducts checks if the collection has any associated products
func (uc *DeleteCollectionUseCase) hasAssociatedProducts(ctx context.Context, collectionID string) (bool, error) {
	// This would typically query the product collection repository
	// For now, we'll return false as a placeholder
	// TODO: Implement actual check for associated products
	return false, nil
}
