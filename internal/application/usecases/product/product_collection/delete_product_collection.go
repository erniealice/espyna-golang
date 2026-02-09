package product_collection

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	productcollectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_collection"
)

// DeleteProductCollectionUseCase handles the business logic for deleting product collections
// DeleteProductCollectionRepositories groups all repository dependencies
type DeleteProductCollectionRepositories struct {
	ProductCollection productcollectionpb.ProductCollectionDomainServiceServer // Primary entity repository
}

// DeleteProductCollectionServices groups all business service dependencies
type DeleteProductCollectionServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// DeleteProductCollectionUseCase handles the business logic for deleting product collections
type DeleteProductCollectionUseCase struct {
	repositories DeleteProductCollectionRepositories
	services     DeleteProductCollectionServices
}

// NewDeleteProductCollectionUseCase creates a new DeleteProductCollectionUseCase
func NewDeleteProductCollectionUseCase(
	repositories DeleteProductCollectionRepositories,
	services DeleteProductCollectionServices,
) *DeleteProductCollectionUseCase {
	return &DeleteProductCollectionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete product collection operation
func (uc *DeleteProductCollectionUseCase) Execute(ctx context.Context, req *productcollectionpb.DeleteProductCollectionRequest) (*productcollectionpb.DeleteProductCollectionResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductCollection, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product collection deletion within a transaction
func (uc *DeleteProductCollectionUseCase) executeWithTransaction(ctx context.Context, req *productcollectionpb.DeleteProductCollectionRequest) (*productcollectionpb.DeleteProductCollectionResponse, error) {
	var result *productcollectionpb.DeleteProductCollectionResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return result, nil
}

// executeCore contains the core business logic for deleting a product collection
func (uc *DeleteProductCollectionUseCase) executeCore(ctx context.Context, req *productcollectionpb.DeleteProductCollectionRequest) (*productcollectionpb.DeleteProductCollectionResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.authorization_failed", "Authorization failed for product collections [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductCollection, ports.ActionDelete)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.authorization_failed", "Authorization failed for product collections [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.authorization_failed", "Authorization failed for product collections [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.input_validation_failed", "Input validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductCollection.DeleteProductCollection(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteProductCollectionUseCase) validateInput(ctx context.Context, req *productcollectionpb.DeleteProductCollectionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.validation.data_required", "Product collection data is required [DEFAULT]"))
	}
	if req.Data.ProductId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.validation.product_id_required", "Product ID is required [DEFAULT]"))
	}
	if req.Data.CollectionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.validation.collection_id_required", "Collection ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for product collection deletion
func (uc *DeleteProductCollectionUseCase) validateBusinessRules(ctx context.Context, req *productcollectionpb.DeleteProductCollectionRequest) error {
	// Additional business rule validation can be added here
	// For example: check if product collection is referenced by other entities
	if uc.isProductCollectionInUse(ctx, req.Data.ProductId, req.Data.CollectionId) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.in_use", "Product collection is currently in use and cannot be deleted [DEFAULT]"))
	}
	return nil
}

// isProductCollectionInUse checks if the product collection is referenced by other entities
func (uc *DeleteProductCollectionUseCase) isProductCollectionInUse(ctx context.Context, productID, collectionID string) bool {
	// Placeholder for actual implementation
	// TODO: Implement actual check for product collection usage
	return false
}
