package product_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	productattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_attribute"
)

// CreateProductAttributeUseCase handles the business logic for creating product attributes
// CreateProductAttributeRepositories groups all repository dependencies
type CreateProductAttributeRepositories struct {
	ProductAttribute productattributepb.ProductAttributeDomainServiceServer // Primary entity repository
	Product          productpb.ProductDomainServiceServer
	Attribute        attributepb.AttributeDomainServiceServer
}

// CreateProductAttributeServices groups all business service dependencies
type CreateProductAttributeServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateProductAttributeUseCase handles the business logic for creating product attributes
type CreateProductAttributeUseCase struct {
	repositories CreateProductAttributeRepositories
	services     CreateProductAttributeServices
}

// NewCreateProductAttributeUseCase creates a new CreateProductAttributeUseCase
func NewCreateProductAttributeUseCase(
	repositories CreateProductAttributeRepositories,
	services CreateProductAttributeServices,
) *CreateProductAttributeUseCase {
	return &CreateProductAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create product attribute operation
func (uc *CreateProductAttributeUseCase) Execute(ctx context.Context, req *productattributepb.CreateProductAttributeRequest) (*productattributepb.CreateProductAttributeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityProductAttribute, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product attribute creation within a transaction
func (uc *CreateProductAttributeUseCase) executeWithTransaction(ctx context.Context, req *productattributepb.CreateProductAttributeRequest) (*productattributepb.CreateProductAttributeResponse, error) {
	var result *productattributepb.CreateProductAttributeResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "product_attribute.errors.creation_failed", "Product attribute creation failed [DEFAULT]"), err)
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
func (uc *CreateProductAttributeUseCase) executeCore(ctx context.Context, req *productattributepb.CreateProductAttributeRequest) (*productattributepb.CreateProductAttributeResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.authorization_failed", "Authorization failed for product attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityProductAttribute, ports.ActionCreate)
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
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.validation_failed", "Input validation failed [DEFAULT]"), err)
	}

	// Business logic and enrichment
	if err := uc.enrichProductAttributeData(req.Data); err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]"), err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.entity_reference_validation_failed", "Entity reference validation failed [DEFAULT]"), err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]"), err)
	}

	// Call repository
	resp, err := uc.repositories.ProductAttribute.CreateProductAttribute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.creation_failed", "Product attribute creation failed [DEFAULT]"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *CreateProductAttributeUseCase) validateInput(ctx context.Context, req *productattributepb.CreateProductAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.validation.data_required", "Product attribute data is required [DEFAULT]"))
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
func (uc *CreateProductAttributeUseCase) enrichProductAttributeData(productAttribute *productattributepb.ProductAttribute) error {
	now := time.Now()

	// Generate ProductAttribute ID if not provided
	if productAttribute.Id == "" {
		if uc.services.IDService != nil {
			productAttribute.Id = uc.services.IDService.GenerateID()
		} else {
			// Fallback ID generation when service is not available
			productAttribute.Id = fmt.Sprintf("product-attr-%d", now.UnixNano())
		}
	}

	// Set audit fields
	productAttribute.DateCreated = &[]int64{now.UnixMilli()}[0]
	productAttribute.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	productAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	productAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints for product attributes
func (uc *CreateProductAttributeUseCase) validateBusinessRules(ctx context.Context, productAttribute *productattributepb.ProductAttribute) error {
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
func (uc *CreateProductAttributeUseCase) validateEntityReferences(ctx context.Context, productAttribute *productattributepb.ProductAttribute) error {
	// Validate Product entity reference
	if productAttribute.ProductId != "" {
		product, err := uc.repositories.Product.ReadProduct(ctx, &productpb.ReadProductRequest{
			Data: &productpb.Product{Id: productAttribute.ProductId},
		})
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.product_reference_validation_failed", "Failed to validate product entity reference [DEFAULT]"), err)
		}
		if product == nil || product.Data == nil || len(product.Data) == 0 {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.product_not_found", "Referenced product with ID '%s' does not exist [DEFAULT]"), productAttribute.ProductId)
		}
		if !product.Data[0].Active {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.product_not_active", "Referenced product with ID '%s' is not active [DEFAULT]"), productAttribute.ProductId)
		}
	}

	// Validate Attribute entity reference
	if productAttribute.AttributeId != "" {
		attribute, err := uc.repositories.Attribute.ReadAttribute(ctx, &attributepb.ReadAttributeRequest{
			Data: &attributepb.Attribute{Id: productAttribute.AttributeId},
		})
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.attribute_reference_validation_failed", "Failed to validate attribute entity reference [DEFAULT]"), err)
		}
		if attribute == nil || attribute.Data == nil || len(attribute.Data) == 0 {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.attribute_not_found", "Referenced attribute with ID '%s' does not exist [DEFAULT]"), productAttribute.AttributeId)
		}
		if !attribute.Data[0].Active {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_attribute.errors.attribute_not_active", "Referenced attribute with ID '%s' is not active [DEFAULT]"), productAttribute.AttributeId)
		}
	}

	return nil
}
