package price_list

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pricelistpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_list"
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
)

type GetPriceListItemPageDataRepositories struct {
	PriceList    pricelistpb.PriceListDomainServiceServer
	PriceProduct priceproductpb.PriceProductDomainServiceServer
}

type GetPriceListItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetPriceListItemPageDataUseCase handles the business logic for getting price list item page data
type GetPriceListItemPageDataUseCase struct {
	repositories GetPriceListItemPageDataRepositories
	services     GetPriceListItemPageDataServices
}

// NewGetPriceListItemPageDataUseCase creates a new GetPriceListItemPageDataUseCase
func NewGetPriceListItemPageDataUseCase(
	repositories GetPriceListItemPageDataRepositories,
	services GetPriceListItemPageDataServices,
) *GetPriceListItemPageDataUseCase {
	return &GetPriceListItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get price list item page data operation
func (uc *GetPriceListItemPageDataUseCase) Execute(
	ctx context.Context,
	req *pricelistpb.GetPriceListItemPageDataRequest,
) (*pricelistpb.GetPriceListItemPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPriceList, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.PriceListId); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes price list item page data retrieval within a transaction
func (uc *GetPriceListItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *pricelistpb.GetPriceListItemPageDataRequest,
) (*pricelistpb.GetPriceListItemPageDataResponse, error) {
	var result *pricelistpb.GetPriceListItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"price_list.errors.item_page_data_failed",
				"price list item page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting price list item page data
func (uc *GetPriceListItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *pricelistpb.GetPriceListItemPageDataRequest,
) (*pricelistpb.GetPriceListItemPageDataResponse, error) {
	// Create read request for the price list
	readReq := &pricelistpb.ReadPriceListRequest{
		Data: &pricelistpb.PriceList{
			Id: req.PriceListId,
		},
	}

	// Retrieve the price list
	readResp, err := uc.repositories.PriceList.ReadPriceList(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_list.errors.read_failed",
			"failed to retrieve price list: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_list.errors.not_found",
			"price list not found",
		))
	}

	// Get the price list (should be only one)
	priceList := readResp.Data[0]

	// Validate that we got the expected price list
	if priceList.Id != req.PriceListId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_list.errors.id_mismatch",
			"retrieved price list ID does not match requested ID",
		))
	}

	// Fetch price products associated with this price list
	listPriceProductsReq := &priceproductpb.ListPriceProductsRequest{}
	priceProductsResp, err := uc.repositories.PriceProduct.ListPriceProducts(ctx, listPriceProductsReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_list.errors.price_products_failed",
			"failed to retrieve price products for price list: %w",
		), err)
	}

	// Filter price products by price_list_id
	var filteredPriceProducts []*priceproductpb.PriceProduct
	if priceProductsResp != nil && priceProductsResp.Data != nil {
		for _, pp := range priceProductsResp.Data {
			if pp.PriceListId != nil && *pp.PriceListId == req.PriceListId {
				filteredPriceProducts = append(filteredPriceProducts, pp)
			}
		}
	}

	return &pricelistpb.GetPriceListItemPageDataResponse{
		PriceList:     priceList,
		PriceProducts: filteredPriceProducts,
		Success:       true,
	}, nil
}

// validateInput validates the input request
func (uc *GetPriceListItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *pricelistpb.GetPriceListItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_list.validation.request_required",
			"request is required",
		))
	}

	if req.PriceListId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_list.validation.id_required",
			"price list ID is required",
		))
	}

	return nil
}

// validateBusinessRules enforces business constraints for reading price list item page data
func (uc *GetPriceListItemPageDataUseCase) validateBusinessRules(
	ctx context.Context,
	priceListId string,
) error {
	if len(priceListId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"price_list.validation.id_too_short",
			"price list ID is too short",
		))
	}

	return nil
}
