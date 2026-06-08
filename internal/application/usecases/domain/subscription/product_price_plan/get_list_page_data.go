package product_price_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
)

type GetProductPricePlanListPageDataRepositories struct {
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer
}

type GetProductPricePlanListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetProductPricePlanListPageDataUseCase handles the business logic for getting product price plan list page data
type GetProductPricePlanListPageDataUseCase struct {
	repositories GetProductPricePlanListPageDataRepositories
	services     GetProductPricePlanListPageDataServices
	processor    *listdata.ListDataProcessor
}

// NewGetProductPricePlanListPageDataUseCase creates a new GetProductPricePlanListPageDataUseCase
func NewGetProductPricePlanListPageDataUseCase(
	repositories GetProductPricePlanListPageDataRepositories,
	services GetProductPricePlanListPageDataServices,
) *GetProductPricePlanListPageDataUseCase {
	return &GetProductPricePlanListPageDataUseCase{
		repositories: repositories,
		services:     services,
		processor:    listdata.NewListDataProcessor(),
	}
}

// Execute performs the get product price plan list page data operation
func (uc *GetProductPricePlanListPageDataUseCase) Execute(
	ctx context.Context,
	req *productpriceplanpb.GetProductPricePlanListPageDataRequest,
) (*productpriceplanpb.GetProductPricePlanListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.PricePlan, entityid.ActionList); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	return uc.executeCore(ctx, req)
}

func (uc *GetProductPricePlanListPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *productpriceplanpb.GetProductPricePlanListPageDataRequest,
) (*productpriceplanpb.GetProductPricePlanListPageDataResponse, error) {
	var result *productpriceplanpb.GetProductPricePlanListPageDataResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.Translator,
				"product_price_plan.errors.list_page_data_failed",
				"product price plan list page data retrieval failed: %w",
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

func (uc *GetProductPricePlanListPageDataUseCase) executeCore(
	ctx context.Context,
	req *productpriceplanpb.GetProductPricePlanListPageDataRequest,
) (*productpriceplanpb.GetProductPricePlanListPageDataResponse, error) {
	listReq := &productpriceplanpb.ListProductPricePlansRequest{}
	listResp, err := uc.repositories.ProductPricePlan.ListProductPricePlans(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"product_price_plan.errors.list_failed",
			"failed to retrieve product price plans: %w",
		), err)
	}

	if listResp == nil || len(listResp.Data) == 0 {
		emptyPagination := uc.processor.GetPaginationUtils().CreatePaginationResponse(req.Pagination, 0, false)
		return &productpriceplanpb.GetProductPricePlanListPageDataResponse{
			ProductPricePlanList: []*productpriceplanpb.ProductPricePlan{},
			Pagination:           emptyPagination,
			SearchResults:        []*commonpb.SearchResult{},
			Success:              true,
		}, nil
	}

	result, err := uc.processor.ProcessListRequest(
		listResp.Data,
		req.Pagination,
		req.Filters,
		req.Sort,
		req.Search,
	)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"product_price_plan.errors.processing_failed",
			"failed to process product price plan list data: %w",
		), err)
	}

	productPricePlans := make([]*productpriceplanpb.ProductPricePlan, len(result.Items))
	for i, item := range result.Items {
		if productPricePlan, ok := item.(*productpriceplanpb.ProductPricePlan); ok {
			productPricePlans[i] = productPricePlan
		} else {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx,
				uc.services.Translator,
				"product_price_plan.errors.type_conversion_failed",
				"failed to convert item to product price plan type",
			))
		}
	}

	searchResults := make([]*commonpb.SearchResult, len(result.SearchResults))
	for i, searchResult := range result.SearchResults {
		searchResults[i] = &commonpb.SearchResult{
			Score:      searchResult.Score,
			Highlights: searchResult.Highlights,
		}
	}

	return &productpriceplanpb.GetProductPricePlanListPageDataResponse{
		ProductPricePlanList: productPricePlans,
		Pagination:           result.PaginationResponse,
		SearchResults:        searchResults,
		Success:              true,
	}, nil
}

func (uc *GetProductPricePlanListPageDataUseCase) validateInput(
	ctx context.Context,
	req *productpriceplanpb.GetProductPricePlanListPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"product_price_plan.validation.request_required",
			"request is required",
		))
	}
	return nil
}
