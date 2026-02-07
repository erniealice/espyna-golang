package product_collection

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	collectionpb "leapfor.xyz/esqyma/golang/v1/domain/product/collection"
	productpb "leapfor.xyz/esqyma/golang/v1/domain/product/product"
	productcollectionpb "leapfor.xyz/esqyma/golang/v1/domain/product/product_collection"
)

// CreateProductCollectionUseCase handles the business logic for creating product collections
// CreateProductCollectionRepositories groups all repository dependencies
type CreateProductCollectionRepositories struct {
	ProductCollection productcollectionpb.ProductCollectionDomainServiceServer // Primary entity repository
	Product           productpb.ProductDomainServiceServer
	Collection        collectionpb.CollectionDomainServiceServer
}

// CreateProductCollectionServices groups all business service dependencies
type CreateProductCollectionServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateProductCollectionUseCase handles the business logic for creating product collections
type CreateProductCollectionUseCase struct {
	repositories CreateProductCollectionRepositories
	services     CreateProductCollectionServices
}

// NewCreateProductCollectionUseCase creates a new CreateProductCollectionUseCase
func NewCreateProductCollectionUseCase(
	repositories CreateProductCollectionRepositories,
	services CreateProductCollectionServices,
) *CreateProductCollectionUseCase {
	return &CreateProductCollectionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create product collection operation
func (uc *CreateProductCollectionUseCase) Execute(ctx context.Context, req *productcollectionpb.CreateProductCollectionRequest) (*productcollectionpb.CreateProductCollectionResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product collection creation within a transaction
func (uc *CreateProductCollectionUseCase) executeWithTransaction(ctx context.Context, req *productcollectionpb.CreateProductCollectionRequest) (*productcollectionpb.CreateProductCollectionResponse, error) {
	var result *productcollectionpb.CreateProductCollectionResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "product_collection.errors.creation_failed", "Product collection creation failed [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic (moved from original Execute method)
func (uc *CreateProductCollectionUseCase) executeCore(ctx context.Context, req *productcollectionpb.CreateProductCollectionRequest) (*productcollectionpb.CreateProductCollectionResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.authorization_failed", "Authorization failed for product collections [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductCollection, ports.ActionCreate)
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

	// Business logic and enrichment
	if err := uc.enrichProductCollectionData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.reference_validation_failed", "Entity reference validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.ProductCollection.CreateProductCollection(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.creation_failed", "Product collection creation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *CreateProductCollectionUseCase) validateInput(ctx context.Context, req *productcollectionpb.CreateProductCollectionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.validation.data_required", "Product collection data is required [DEFAULT]"))
	}
	// ProductCollection doesn't have Name field - removed invalid check
	if req.Data.ProductId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.validation.product_id_required", "Product ID is required [DEFAULT]"))
	}
	if req.Data.CollectionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.validation.collection_id_required", "Collection ID is required [DEFAULT]"))
	}
	return nil
}

// enrichProductCollectionData adds generated fields and audit information
func (uc *CreateProductCollectionUseCase) enrichProductCollectionData(productCollection *productcollectionpb.ProductCollection) error {
	now := time.Now()

	// Generate ProductCollection ID if not provided
	if productCollection.Id == "" {
		productCollection.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	productCollection.DateCreated = &[]int64{now.UnixMilli()}[0]
	productCollection.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	productCollection.DateModified = &[]int64{now.UnixMilli()}[0]
	productCollection.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	productCollection.Active = true

	return nil
}

// validateBusinessRules enforces business constraints for product collections
func (uc *CreateProductCollectionUseCase) validateBusinessRules(ctx context.Context, productCollection *productcollectionpb.ProductCollection) error {
	// Validate product ID format
	if len(productCollection.ProductId) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.validation.product_id_min_length", "Product ID must be at least 5 characters long [DEFAULT]"))
	}

	// Validate collection ID format
	if len(productCollection.CollectionId) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.validation.collection_id_min_length", "Collection ID must be at least 2 characters long [DEFAULT]"))
	}

	// Business constraint: Product collection must be associated with valid product and collection
	if productCollection.ProductId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.validation.product_association_required", "Product collection must be associated with a product [DEFAULT]"))
	}

	if productCollection.CollectionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.validation.collection_association_required", "Product collection must be associated with a collection [DEFAULT]"))
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateProductCollectionUseCase) validateEntityReferences(ctx context.Context, productCollection *productcollectionpb.ProductCollection) error {
	// Validate Product entity reference
	if productCollection.ProductId != "" {
		product, err := uc.repositories.Product.ReadProduct(ctx, &productpb.ReadProductRequest{
			Data: &productpb.Product{Id: productCollection.ProductId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.product_reference_validation_failed", "Failed to validate product entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if product == nil || product.Data == nil || len(product.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.product_not_found", "Referenced product with ID '{productId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{productId}", productCollection.ProductId)
			return errors.New(translatedError)
		}
		if !product.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.product_not_active", "Referenced product with ID '{productId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{productId}", productCollection.ProductId)
			return errors.New(translatedError)
		}
	}

	// Validate Collection entity reference
	if productCollection.CollectionId != "" {
		collection, err := uc.repositories.Collection.ReadCollection(ctx, &collectionpb.ReadCollectionRequest{
			Data: &collectionpb.Collection{Id: productCollection.CollectionId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.collection_reference_validation_failed", "Failed to validate collection entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if collection == nil || collection.Data == nil || len(collection.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.collection_not_found", "Referenced collection with ID '{collectionId}' does not exist [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{collectionId}", productCollection.CollectionId)
			return errors.New(translatedError)
		}
		if !collection.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_collection.errors.collection_not_active", "Referenced collection with ID '{collectionId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{collectionId}", productCollection.CollectionId)
			return errors.New(translatedError)
		}
	}

	return nil
}
