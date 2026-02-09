package product_attribute

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	productattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_attribute"
)

// UpdateProductAttributeUseCase handles the business logic for updating product attributes
// UpdateProductAttributeRepositories groups all repository dependencies
type UpdateProductAttributeRepositories struct {
	ProductAttribute productattributepb.ProductAttributeDomainServiceServer // Primary entity repository
	Product          productpb.ProductDomainServiceServer
	Attribute        attributepb.AttributeDomainServiceServer
}

// UpdateProductAttributeServices groups all business service dependencies
type UpdateProductAttributeServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// UpdateProductAttributeUseCase handles the business logic for updating product attributes
type UpdateProductAttributeUseCase struct {
	repositories UpdateProductAttributeRepositories
	services     UpdateProductAttributeServices
}

// NewUpdateProductAttributeUseCase creates a new UpdateProductAttributeUseCase
func NewUpdateProductAttributeUseCase(
	repositories UpdateProductAttributeRepositories,
	services UpdateProductAttributeServices,
) *UpdateProductAttributeUseCase {
	return &UpdateProductAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update product attribute operation
func (uc *UpdateProductAttributeUseCase) Execute(ctx context.Context, req *productattributepb.UpdateProductAttributeRequest) (*productattributepb.UpdateProductAttributeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductAttribute, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product attribute update within a transaction
func (uc *UpdateProductAttributeUseCase) executeWithTransaction(ctx context.Context, req *productattributepb.UpdateProductAttributeRequest) (*productattributepb.UpdateProductAttributeResponse, error) {
	var result *productattributepb.UpdateProductAttributeResponse

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
func (uc *UpdateProductAttributeUseCase) executeCore(ctx context.Context, req *productattributepb.UpdateProductAttributeRequest) (*productattributepb.UpdateProductAttributeResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductAttribute, ports.ActionUpdate)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichProductAttributeData(req.Data); err != nil {
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
	resp, err := uc.repositories.ProductAttribute.UpdateProductAttribute(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateProductAttributeUseCase) validateInput(ctx context.Context, req *productattributepb.UpdateProductAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.validation.data_required", "Product attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.validation.id_required", "Course attribute ID is required"))
	}
	if req.Data.ProductId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.validation.product_id_required", "Product ID is required [DEFAULT]"))
	}
	if req.Data.AttributeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.validation.attribute_id_required", "Attribute ID is required [DEFAULT]"))
	}
	if req.Data.Value == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.validation.value_required", "Attribute value is required [DEFAULT]"))
	}
	return nil
}

// enrichProductAttributeData adds generated fields and audit information
func (uc *UpdateProductAttributeUseCase) enrichProductAttributeData(productAttribute *productattributepb.ProductAttribute) error {
	now := time.Now()

	// Update audit fields
	productAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	productAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints for product attributes
func (uc *UpdateProductAttributeUseCase) validateBusinessRules(ctx context.Context, productAttribute *productattributepb.ProductAttribute) error {
	// Validate product ID format
	if len(productAttribute.ProductId) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.validation.product_id_min_length", "Product ID must be at least 5 characters long [DEFAULT]"))
	}

	// Validate attribute ID format
	if len(productAttribute.AttributeId) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.validation.attribute_id_min_length", "Attribute ID must be at least 2 characters long [DEFAULT]"))
	}

	// Validate attribute value length
	value := strings.TrimSpace(productAttribute.Value)
	if len(value) < 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.validation.value_not_empty", "Attribute value must not be empty [DEFAULT]"))
	}

	if len(value) > 500 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.validation.value_max_length", "Attribute value cannot exceed 500 characters [DEFAULT]"))
	}

	// Normalize value (trim spaces)
	productAttribute.Value = strings.TrimSpace(productAttribute.Value)

	// Business constraint: Product attribute must be associated with a valid product
	if productAttribute.ProductId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.validation.product_association_required", "Product attribute must be associated with a product [DEFAULT]"))
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateProductAttributeUseCase) validateEntityReferences(ctx context.Context, productAttribute *productattributepb.ProductAttribute) error {
	// Validate Product entity reference
	if productAttribute.ProductId != "" {
		product, err := uc.repositories.Product.ReadProduct(ctx, &productpb.ReadProductRequest{
			Data: &productpb.Product{Id: productAttribute.ProductId},
		})
		if err != nil {
			return err
		}
		if product == nil || product.Data == nil || len(product.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "product_attribute.errors.product_not_found", map[string]interface{}{"productId": productAttribute.ProductId}, "Referenced product not found")
			return errors.New(translatedError)
		}
		if !product.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "product_attribute.errors.product_not_active", map[string]interface{}{"productId": productAttribute.ProductId}, "Referenced product not active")
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
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "product_attribute.errors.attribute_not_found", map[string]interface{}{"attributeId": productAttribute.AttributeId}, "Referenced attribute not found")
			return errors.New(translatedError)
		}
		if !attribute.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "product_attribute.errors.attribute_not_active", map[string]interface{}{"attributeId": productAttribute.AttributeId}, "Referenced attribute not active")
			return errors.New(translatedError)
		}
	}

	return nil
}
