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

// DeleteProductPricePlanUseCase handles the business logic for deleting product price plans
type DeleteProductPricePlanUseCase struct {
	repositories DeleteProductPricePlanRepositories
	services     DeleteProductPricePlanServices
}

// NewDeleteProductPricePlanUseCase creates a new DeleteProductPricePlanUseCase
func NewDeleteProductPricePlanUseCase(
	repositories DeleteProductPricePlanRepositories,
	services DeleteProductPricePlanServices,
) *DeleteProductPricePlanUseCase {
	return &DeleteProductPricePlanUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete product price plan operation
func (uc *DeleteProductPricePlanUseCase) Execute(ctx context.Context, req *productpriceplanpb.DeleteProductPricePlanRequest) (*productpriceplanpb.DeleteProductPricePlanResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPricePlan, ports.ActionDelete); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	return uc.executeCore(ctx, req)
}

func (uc *DeleteProductPricePlanUseCase) executeWithTransaction(ctx context.Context, req *productpriceplanpb.DeleteProductPricePlanRequest) (*productpriceplanpb.DeleteProductPricePlanResponse, error) {
	var result *productpriceplanpb.DeleteProductPricePlanResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_price_plan.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return result, nil
}

func (uc *DeleteProductPricePlanUseCase) executeCore(ctx context.Context, req *productpriceplanpb.DeleteProductPricePlanRequest) (*productpriceplanpb.DeleteProductPricePlanResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.ProductPricePlan.DeleteProductPricePlan(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (uc *DeleteProductPricePlanUseCase) validateInput(ctx context.Context, req *productpriceplanpb.DeleteProductPricePlanRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_price_plan.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_price_plan.validation.data_required", "Product price plan data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "product_price_plan.validation.id_required", "Product price plan ID is required [DEFAULT]"))
	}
	return nil
}
