package product_price_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
)

type GetProductPricePlanItemPageDataRepositories struct {
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer
}

type GetProductPricePlanItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetProductPricePlanItemPageDataUseCase handles the business logic for getting product price plan item page data
type GetProductPricePlanItemPageDataUseCase struct {
	repositories GetProductPricePlanItemPageDataRepositories
	services     GetProductPricePlanItemPageDataServices
}

// NewGetProductPricePlanItemPageDataUseCase creates a new GetProductPricePlanItemPageDataUseCase
func NewGetProductPricePlanItemPageDataUseCase(
	repositories GetProductPricePlanItemPageDataRepositories,
	services GetProductPricePlanItemPageDataServices,
) *GetProductPricePlanItemPageDataUseCase {
	return &GetProductPricePlanItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get product price plan item page data operation
func (uc *GetProductPricePlanItemPageDataUseCase) Execute(
	ctx context.Context,
	req *productpriceplanpb.GetProductPricePlanItemPageDataRequest,
) (*productpriceplanpb.GetProductPricePlanItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPricePlan, ports.ActionList); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	return uc.executeCore(ctx, req)
}

func (uc *GetProductPricePlanItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *productpriceplanpb.GetProductPricePlanItemPageDataRequest,
) (*productpriceplanpb.GetProductPricePlanItemPageDataResponse, error) {
	var result *productpriceplanpb.GetProductPricePlanItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"product_price_plan.errors.item_page_data_failed",
				"product price plan item page data retrieval failed: %w",
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

func (uc *GetProductPricePlanItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *productpriceplanpb.GetProductPricePlanItemPageDataRequest,
) (*productpriceplanpb.GetProductPricePlanItemPageDataResponse, error) {
	readReq := &productpriceplanpb.ReadProductPricePlanRequest{
		Data: &productpriceplanpb.ProductPricePlan{
			Id: req.ProductPricePlanId,
		},
	}

	readResp, err := uc.repositories.ProductPricePlan.ReadProductPricePlan(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"product_price_plan.errors.read_failed",
			"failed to retrieve product price plan: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"product_price_plan.errors.not_found",
			"product price plan not found",
		))
	}

	productPricePlan := readResp.Data[0]

	return &productpriceplanpb.GetProductPricePlanItemPageDataResponse{
		ProductPricePlan: productPricePlan,
		Success:          true,
	}, nil
}

func (uc *GetProductPricePlanItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *productpriceplanpb.GetProductPricePlanItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"product_price_plan.validation.request_required",
			"request is required",
		))
	}

	if req.ProductPricePlanId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"product_price_plan.validation.id_required",
			"product price plan ID is required",
		))
	}

	return nil
}
