package collection_attribute

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	collectionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_attribute"
)

// DeleteCollectionAttributeUseCase handles the business logic for deleting product attributes
// DeleteCollectionAttributeRepositories groups all repository dependencies
type DeleteCollectionAttributeRepositories struct {
	CollectionAttribute collectionattributepb.CollectionAttributeDomainServiceServer // Primary entity repository
}

// DeleteCollectionAttributeServices groups all business service dependencies
type DeleteCollectionAttributeServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// DeleteCollectionAttributeUseCase handles the business logic for deleting product attributes
type DeleteCollectionAttributeUseCase struct {
	repositories DeleteCollectionAttributeRepositories
	services     DeleteCollectionAttributeServices
}

// NewDeleteCollectionAttributeUseCase creates a new DeleteCollectionAttributeUseCase
func NewDeleteCollectionAttributeUseCase(
	repositories DeleteCollectionAttributeRepositories,
	services DeleteCollectionAttributeServices,
) *DeleteCollectionAttributeUseCase {
	return &DeleteCollectionAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete product attribute operation
func (uc *DeleteCollectionAttributeUseCase) Execute(ctx context.Context, req *collectionattributepb.DeleteCollectionAttributeRequest) (*collectionattributepb.DeleteCollectionAttributeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityCollectionAttribute, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product attribute deletion within a transaction
func (uc *DeleteCollectionAttributeUseCase) executeWithTransaction(ctx context.Context, req *collectionattributepb.DeleteCollectionAttributeRequest) (*collectionattributepb.DeleteCollectionAttributeResponse, error) {
	var result *collectionattributepb.DeleteCollectionAttributeResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return result, nil
}

// executeCore contains the core business logic for deleting a product attribute
func (uc *DeleteCollectionAttributeUseCase) executeCore(ctx context.Context, req *collectionattributepb.DeleteCollectionAttributeRequest) (*collectionattributepb.DeleteCollectionAttributeResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityCollectionAttribute, ports.ActionDelete)
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

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.CollectionAttribute.DeleteCollectionAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.deletion_failed", "Collection attribute deletion failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteCollectionAttributeUseCase) validateInput(ctx context.Context, req *collectionattributepb.DeleteCollectionAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.data_required", "Collection attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.id_required", "Collection attribute ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for product attribute deletion
func (uc *DeleteCollectionAttributeUseCase) validateBusinessRules(ctx context.Context, req *collectionattributepb.DeleteCollectionAttributeRequest) error {
	// Additional business rule validation can be added here
	// For example: check if product attribute is referenced by other entities
	if uc.isCollectionAttributeInUse(ctx, req.Data.CollectionId, req.Data.AttributeId) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.in_use", "Collection attribute is currently in use and cannot be deleted [DEFAULT]"))
	}
	return nil
}

// isCollectionAttributeInUse checks if the product attribute is referenced by other entities
func (uc *DeleteCollectionAttributeUseCase) isCollectionAttributeInUse(ctx context.Context, productID, attributeID string) bool {
	// Placeholder for actual implementation
	// TODO: Implement actual check for product attribute usage
	return false
}
