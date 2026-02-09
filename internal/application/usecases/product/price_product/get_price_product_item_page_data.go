package price_product

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
)

type GetPriceProductItemPageDataRepositories struct {
	PriceProduct priceproductpb.PriceProductDomainServiceServer
}

type GetPriceProductItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetPriceProductItemPageDataUseCase handles the business logic for getting price product item page data
type GetPriceProductItemPageDataUseCase struct {
	repositories GetPriceProductItemPageDataRepositories
	services     GetPriceProductItemPageDataServices
}

// NewGetPriceProductItemPageDataUseCase creates a new GetPriceProductItemPageDataUseCase
func NewGetPriceProductItemPageDataUseCase(
	repositories GetPriceProductItemPageDataRepositories,
	services GetPriceProductItemPageDataServices,
) *GetPriceProductItemPageDataUseCase {
	return &GetPriceProductItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get price product item page data operation
func (uc *GetPriceProductItemPageDataUseCase) Execute(
	ctx context.Context,
	req *priceproductpb.GetPriceProductItemPageDataRequest,
) (*priceproductpb.GetPriceProductItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPriceProduct, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.PriceProductId); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes price product item page data retrieval within a transaction
func (uc *GetPriceProductItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *priceproductpb.GetPriceProductItemPageDataRequest,
) (*priceproductpb.GetPriceProductItemPageDataResponse, error) {
	var result *priceproductpb.GetPriceProductItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"price_product.errors.item_page_data_failed",
				"price product item page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting price product item page data
func (uc *GetPriceProductItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *priceproductpb.GetPriceProductItemPageDataRequest,
) (*priceproductpb.GetPriceProductItemPageDataResponse, error) {
	// Create read request for the price product
	readReq := &priceproductpb.ReadPriceProductRequest{
		Data: &priceproductpb.PriceProduct{
			Id: req.PriceProductId,
		},
	}

	// Retrieve the price product
	readResp, err := uc.repositories.PriceProduct.ReadPriceProduct(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_product.errors.read_failed",
			"failed to retrieve price product: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_product.errors.not_found",
			"price product not found",
		))
	}

	// Get the price product (should be only one)
	priceProduct := readResp.Data[0]

	// Validate that we got the expected price product
	if priceProduct.Id != req.PriceProductId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_product.errors.id_mismatch",
			"retrieved price product ID does not match requested ID",
		))
	}

	// TODO: In a real implementation, you might want to:
	// 1. Load related data (product details, currency details, etc.) if not already populated
	// 2. Apply business rules for data visibility/access control
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging for item access

	// For now, return the price product as-is
	return &priceproductpb.GetPriceProductItemPageDataResponse{
		PriceProduct: priceProduct,
		Success:      true,
	}, nil
}

// validateInput validates the input request
func (uc *GetPriceProductItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *priceproductpb.GetPriceProductItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_product.validation.request_required",
			"request is required",
		))
	}

	if req.PriceProductId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_product.validation.id_required",
			"price product ID is required",
		))
	}

	return nil
}

// validateBusinessRules enforces business constraints for reading price product item page data
func (uc *GetPriceProductItemPageDataUseCase) validateBusinessRules(
	ctx context.Context,
	priceProductId string,
) error {
	// Validate price product ID format
	if len(priceProductId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_product.validation.id_too_short",
			"price product ID is too short",
		))
	}

	// Additional business rules could be added here:
	// - Check user permissions to access this price product
	// - Validate price product belongs to the current user's organization
	// - Check if price product is in a state that allows viewing
	// - Rate limiting for price product access
	// - Audit logging requirements

	return nil
}

// Optional: Helper methods for future enhancements

// loadRelatedData loads related entities like product details
// This would be called from executeCore if needed
func (uc *GetPriceProductItemPageDataUseCase) loadRelatedData(
	ctx context.Context,
	priceProduct *priceproductpb.PriceProduct,
) error {
	// TODO: Implement loading of related data
	// This could involve calls to product repositories
	// to populate the nested product objects if they're not already loaded

	// Example implementation would be:
	// if priceProduct.Product == nil && priceProduct.ProductId != "" {
	//     // Load product data
	// }

	return nil
}

// applyDataTransformation applies any necessary data transformations for the frontend
func (uc *GetPriceProductItemPageDataUseCase) applyDataTransformation(
	ctx context.Context,
	priceProduct *priceproductpb.PriceProduct,
) *priceproductpb.PriceProduct {
	// TODO: Apply any transformations needed for optimal frontend consumption
	// This could include:
	// - Formatting prices and currency
	// - Computing derived fields
	// - Applying localization
	// - Sanitizing sensitive data

	return priceProduct
}

// checkAccessPermissions validates user has permission to access this price product
func (uc *GetPriceProductItemPageDataUseCase) checkAccessPermissions(
	ctx context.Context,
	priceProductId string,
) error {
	// TODO: Implement proper access control
	// This could involve:
	// - Checking user role/permissions
	// - Validating price product belongs to user's organization
	// - Applying multi-tenant access controls

	return nil
}
