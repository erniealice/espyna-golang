package product_collection

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productcollectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_collection"
)

type GetProductCollectionItemPageDataRepositories struct {
	ProductCollection productcollectionpb.ProductCollectionDomainServiceServer
}

type GetProductCollectionItemPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetProductCollectionItemPageDataUseCase handles the business logic for getting product collection item page data
type GetProductCollectionItemPageDataUseCase struct {
	repositories GetProductCollectionItemPageDataRepositories
	services     GetProductCollectionItemPageDataServices
}

// NewGetProductCollectionItemPageDataUseCase creates a new GetProductCollectionItemPageDataUseCase
func NewGetProductCollectionItemPageDataUseCase(
	repositories GetProductCollectionItemPageDataRepositories,
	services GetProductCollectionItemPageDataServices,
) *GetProductCollectionItemPageDataUseCase {
	return &GetProductCollectionItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get product collection item page data operation
func (uc *GetProductCollectionItemPageDataUseCase) Execute(
	ctx context.Context,
	req *productcollectionpb.GetProductCollectionItemPageDataRequest,
) (*productcollectionpb.GetProductCollectionItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.ProductCollectionId); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product collection item page data retrieval within a transaction
func (uc *GetProductCollectionItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *productcollectionpb.GetProductCollectionItemPageDataRequest,
) (*productcollectionpb.GetProductCollectionItemPageDataResponse, error) {
	var result *productcollectionpb.GetProductCollectionItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"product_collection.errors.item_page_data_failed",
				"product collection item page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting product collection item page data
func (uc *GetProductCollectionItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *productcollectionpb.GetProductCollectionItemPageDataRequest,
) (*productcollectionpb.GetProductCollectionItemPageDataResponse, error) {
	// Create read request for the product collection
	readReq := &productcollectionpb.ReadProductCollectionRequest{
		Data: &productcollectionpb.ProductCollection{
			Id: req.ProductCollectionId,
		},
	}

	// Retrieve the product collection
	readResp, err := uc.repositories.ProductCollection.ReadProductCollection(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"product_collection.errors.read_failed",
			"failed to retrieve product collection: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"product_collection.errors.not_found",
			"product collection not found",
		))
	}

	// Get the product collection (should be only one)
	productCollection := readResp.Data[0]

	// Validate that we got the expected product collection
	if productCollection.Id != req.ProductCollectionId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"product_collection.errors.id_mismatch",
			"retrieved product collection ID does not match requested ID",
		))
	}

	return &productcollectionpb.GetProductCollectionItemPageDataResponse{
		ProductCollection: productCollection,
		Success:           true,
	}, nil
}

// validateInput validates the input request
func (uc *GetProductCollectionItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *productcollectionpb.GetProductCollectionItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"product_collection.validation.request_required",
			"request is required",
		))
	}

	if req.ProductCollectionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"product_collection.validation.id_required",
			"product collection ID is required",
		))
	}

	return nil
}

// validateBusinessRules enforces business constraints for reading product collection item page data
func (uc *GetProductCollectionItemPageDataUseCase) validateBusinessRules(
	ctx context.Context,
	productCollectionId string,
) error {
	// Validate product collection ID format
	if len(productCollectionId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"product_collection.validation.id_too_short",
			"product collection ID is too short",
		))
	}

	return nil
}
