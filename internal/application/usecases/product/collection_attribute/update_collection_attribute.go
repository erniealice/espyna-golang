package collection_attribute

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection"
	collectionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_attribute"
)

// UpdateCollectionAttributeUseCase handles the business logic for updating product attributes
// UpdateCollectionAttributeRepositories groups all repository dependencies
type UpdateCollectionAttributeRepositories struct {
	CollectionAttribute collectionattributepb.CollectionAttributeDomainServiceServer // Primary entity repository
	Collection          collectionpb.CollectionDomainServiceServer
	Attribute           attributepb.AttributeDomainServiceServer
}

// UpdateCollectionAttributeServices groups all business service dependencies
type UpdateCollectionAttributeServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// UpdateCollectionAttributeUseCase handles the business logic for updating product attributes
type UpdateCollectionAttributeUseCase struct {
	repositories UpdateCollectionAttributeRepositories
	services     UpdateCollectionAttributeServices
}

// NewUpdateCollectionAttributeUseCase creates a new UpdateCollectionAttributeUseCase
func NewUpdateCollectionAttributeUseCase(
	repositories UpdateCollectionAttributeRepositories,
	services UpdateCollectionAttributeServices,
) *UpdateCollectionAttributeUseCase {
	return &UpdateCollectionAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update product attribute operation
func (uc *UpdateCollectionAttributeUseCase) Execute(ctx context.Context, req *collectionattributepb.UpdateCollectionAttributeRequest) (*collectionattributepb.UpdateCollectionAttributeResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product attribute update within a transaction
func (uc *UpdateCollectionAttributeUseCase) executeWithTransaction(ctx context.Context, req *collectionattributepb.UpdateCollectionAttributeRequest) (*collectionattributepb.UpdateCollectionAttributeResponse, error) {
	var result *collectionattributepb.UpdateCollectionAttributeResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
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
func (uc *UpdateCollectionAttributeUseCase) executeCore(ctx context.Context, req *collectionattributepb.UpdateCollectionAttributeRequest) (*collectionattributepb.UpdateCollectionAttributeResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityCollectionAttribute, ports.ActionUpdate)
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

	// Business logic and enrichment
	if err := uc.enrichCollectionAttributeData(req.Data); err != nil {
		return nil, err
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.CollectionAttribute.UpdateCollectionAttribute(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateCollectionAttributeUseCase) validateInput(ctx context.Context, req *collectionattributepb.UpdateCollectionAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.data_required", "Collection attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.id_required", "Course attribute ID is required"))
	}
	if req.Data.CollectionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.collection_id_required", "Collection ID is required [DEFAULT]"))
	}
	if req.Data.AttributeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.attribute_id_required", "Attribute ID is required [DEFAULT]"))
	}
	if req.Data.Value == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.value_required", "Attribute value is required [DEFAULT]"))
	}
	return nil
}

// enrichCollectionAttributeData adds generated fields and audit information
func (uc *UpdateCollectionAttributeUseCase) enrichCollectionAttributeData(productAttribute *collectionattributepb.CollectionAttribute) error {
	now := time.Now()

	// Update audit fields
	productAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	productAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints for product attributes
func (uc *UpdateCollectionAttributeUseCase) validateBusinessRules(ctx context.Context, productAttribute *collectionattributepb.CollectionAttribute) error {
	// Validate product ID format
	if len(productAttribute.CollectionId) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.collection_id_min_length", "Collection ID must be at least 5 characters long [DEFAULT]"))
	}

	// Validate attribute ID format
	if len(productAttribute.AttributeId) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.attribute_id_min_length", "Attribute ID must be at least 2 characters long [DEFAULT]"))
	}

	// Validate attribute value length
	value := strings.TrimSpace(productAttribute.Value)
	if len(value) < 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.value_not_empty", "Attribute value must not be empty [DEFAULT]"))
	}

	if len(value) > 500 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.value_max_length", "Attribute value cannot exceed 500 characters [DEFAULT]"))
	}

	// Normalize value (trim spaces)
	productAttribute.Value = strings.TrimSpace(productAttribute.Value)

	// Business constraint: Collection attribute must be associated with a valid product
	if productAttribute.CollectionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "collection_attribute.validation.product_association_required", "Collection attribute must be associated with a product [DEFAULT]"))
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateCollectionAttributeUseCase) validateEntityReferences(ctx context.Context, productAttribute *collectionattributepb.CollectionAttribute) error {
	// Validate Collection entity reference
	if productAttribute.CollectionId != "" {
		product, err := uc.repositories.Collection.ReadCollection(ctx, &collectionpb.ReadCollectionRequest{
			Data: &collectionpb.Collection{Id: productAttribute.CollectionId},
		})
		if err != nil {
			return err
		}
		if product == nil || product.Data == nil || len(product.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "collection_attribute.errors.product_not_found", map[string]interface{}{"productId": productAttribute.CollectionId}, "Referenced product not found")
			return errors.New(translatedError)
		}
		if !product.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "collection_attribute.errors.product_not_active", map[string]interface{}{"productId": productAttribute.CollectionId}, "Referenced product not active")
			return errors.New(translatedError)
		}
	}

	// Validate Attribute entity reference
	if productAttribute.AttributeId != "" {
		attribute, err := uc.repositories.Attribute.ReadAttribute(ctx, &attributepb.ReadAttributeRequest{
			Data: &attributepb.Attribute{Id: productAttribute.AttributeId},
		})
		if err != nil {
			return err
		}
		if attribute == nil || attribute.Data == nil || len(attribute.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "collection_attribute.errors.attribute_not_found", map[string]interface{}{"attributeId": productAttribute.AttributeId}, "Referenced attribute not found")
			return errors.New(translatedError)
		}
		if !attribute.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "collection_attribute.errors.attribute_not_active", map[string]interface{}{"attributeId": productAttribute.AttributeId}, "Referenced attribute not active")
			return errors.New(translatedError)
		}
	}

	return nil
}
