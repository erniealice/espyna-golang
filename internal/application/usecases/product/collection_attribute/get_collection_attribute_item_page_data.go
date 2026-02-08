package collection_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	collectionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_attribute"
)

type GetCollectionAttributeItemPageDataRepositories struct {
	CollectionAttribute collectionattributepb.CollectionAttributeDomainServiceServer
}

type GetCollectionAttributeItemPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetCollectionAttributeItemPageDataUseCase handles the business logic for getting product attribute item page data
type GetCollectionAttributeItemPageDataUseCase struct {
	repositories GetCollectionAttributeItemPageDataRepositories
	services     GetCollectionAttributeItemPageDataServices
}

// NewGetCollectionAttributeItemPageDataUseCase creates a new GetCollectionAttributeItemPageDataUseCase
func NewGetCollectionAttributeItemPageDataUseCase(
	repositories GetCollectionAttributeItemPageDataRepositories,
	services GetCollectionAttributeItemPageDataServices,
) *GetCollectionAttributeItemPageDataUseCase {
	return &GetCollectionAttributeItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get product attribute item page data operation
func (uc *GetCollectionAttributeItemPageDataUseCase) Execute(
	ctx context.Context,
	req *collectionattributepb.GetCollectionAttributeItemPageDataRequest,
) (*collectionattributepb.GetCollectionAttributeItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.CollectionAttributeId); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product attribute item page data retrieval within a transaction
func (uc *GetCollectionAttributeItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *collectionattributepb.GetCollectionAttributeItemPageDataRequest,
) (*collectionattributepb.GetCollectionAttributeItemPageDataResponse, error) {
	var result *collectionattributepb.GetCollectionAttributeItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"collection_attribute.errors.item_page_data_failed",
				"product attribute item page data retrieval failed: %w",
			), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting product attribute item page data
func (uc *GetCollectionAttributeItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *collectionattributepb.GetCollectionAttributeItemPageDataRequest,
) (*collectionattributepb.GetCollectionAttributeItemPageDataResponse, error) {
	// Create read request for the product attribute
	readReq := &collectionattributepb.ReadCollectionAttributeRequest{
		Data: &collectionattributepb.CollectionAttribute{
			Id: req.CollectionAttributeId,
		},
	}

	// Retrieve the product attribute
	readResp, err := uc.repositories.CollectionAttribute.ReadCollectionAttribute(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"collection_attribute.errors.read_failed",
			"failed to retrieve product attribute: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"collection_attribute.errors.not_found",
			"product attribute not found",
		))
	}

	// Get the product attribute (should be only one)
	productAttribute := readResp.Data[0]

	// Validate that we got the expected product attribute
	if productAttribute.Id != req.CollectionAttributeId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"collection_attribute.errors.id_mismatch",
			"retrieved product attribute ID does not match requested ID",
		))
	}

	// TODO: In a real implementation, you might want to:
	// 1. Load related data (product details, attribute details, etc.) if not already populated
	// 2. Apply business rules for data visibility/access control
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging for item access

	// For now, return the product attribute as-is
	return &collectionattributepb.GetCollectionAttributeItemPageDataResponse{
		CollectionAttribute: productAttribute,
		Success:             true,
	}, nil
}

// validateInput validates the input request
func (uc *GetCollectionAttributeItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *collectionattributepb.GetCollectionAttributeItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"collection_attribute.validation.request_required",
			"Request is required for collection attributes [DEFAULT]",
		))
	}

	// Validate collection attribute ID - uses direct field NOT nested Data
	if strings.TrimSpace(req.CollectionAttributeId) == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"collection_attribute.validation.id_required",
			"Collection attribute ID is required [DEFAULT]",
		))
	}

	// Basic ID format validation
	if len(req.CollectionAttributeId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"collection_attribute.validation.id_too_short",
			"Collection attribute ID must be at least 3 characters [DEFAULT]",
		))
	}

	return nil
}

// validateBusinessRules enforces business constraints for reading product attribute item page data
func (uc *GetCollectionAttributeItemPageDataUseCase) validateBusinessRules(
	ctx context.Context,
	productAttributeId string,
) error {
	// Validate product attribute ID format
	if len(productAttributeId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"collection_attribute.validation.id_too_short",
			"product attribute ID is too short",
		))
	}

	// Additional business rules could be added here:
	// - Check user permissions to access this product attribute
	// - Validate product attribute belongs to the current user's organization
	// - Check if product attribute is in a state that allows viewing
	// - Rate limiting for product attribute access
	// - Audit logging requirements

	return nil
}

// Optional: Helper methods for future enhancements

// loadRelatedData loads related entities like product and attribute details
// This would be called from executeCore if needed
func (uc *GetCollectionAttributeItemPageDataUseCase) loadRelatedData(
	ctx context.Context,
	productAttribute *collectionattributepb.CollectionAttribute,
) error {
	// TODO: Implement loading of related data
	// This could involve calls to product and attribute repositories
	// to populate the nested objects if they're not already loaded

	// Example implementation would be:
	// if productAttribute.Collection == nil && productAttribute.CollectionId != "" {
	//     // Load product data
	// }
	// if productAttribute.Attribute == nil && productAttribute.AttributeId != "" {
	//     // Load attribute data
	// }

	return nil
}

// applyDataTransformation applies any necessary data transformations for the frontend
func (uc *GetCollectionAttributeItemPageDataUseCase) applyDataTransformation(
	ctx context.Context,
	productAttribute *collectionattributepb.CollectionAttribute,
) *collectionattributepb.CollectionAttribute {
	// TODO: Apply any transformations needed for optimal frontend consumption
	// This could include:
	// - Formatting attribute values
	// - Computing derived fields
	// - Applying localization
	// - Sanitizing sensitive data

	return productAttribute
}

// checkAccessPermissions validates user has permission to access this product attribute
func (uc *GetCollectionAttributeItemPageDataUseCase) checkAccessPermissions(
	ctx context.Context,
	productAttributeId string,
) error {
	// TODO: Implement proper access control
	// This could involve:
	// - Checking user role/permissions
	// - Validating product attribute belongs to user's organization
	// - Applying multi-tenant access controls

	return nil
}
