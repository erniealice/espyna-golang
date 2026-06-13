package product_plan

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
)

// DeleteProductPlanUseCase handles the business logic for deleting product plans
type DeleteProductPlanUseCase struct {
	repositories DeleteProductPlanRepositories
	services     DeleteProductPlanServices
}

// NewDeleteProductPlanUseCase creates a new DeleteProductPlanUseCase
func NewDeleteProductPlanUseCase(
	repositories DeleteProductPlanRepositories,
	services DeleteProductPlanServices,
) *DeleteProductPlanUseCase {
	return &DeleteProductPlanUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete product plan operation
func (uc *DeleteProductPlanUseCase) Execute(ctx context.Context, req *productplanpb.DeleteProductPlanRequest) (*productplanpb.DeleteProductPlanResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.ProductPlan,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes product plan deletion within a transaction
func (uc *DeleteProductPlanUseCase) executeWithTransaction(ctx context.Context, req *productplanpb.DeleteProductPlanRequest) (*productplanpb.DeleteProductPlanResponse, error) {
	var result *productplanpb.DeleteProductPlanResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_plan.errors.transaction_failed", "Transaction execution failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return result, nil
}

// executeCore contains the core business logic for deleting a product plan
func (uc *DeleteProductPlanUseCase) executeCore(ctx context.Context, req *productplanpb.DeleteProductPlanRequest) (*productplanpb.DeleteProductPlanResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.ProductPlan.DeleteProductPlan(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteProductPlanUseCase) validateInput(ctx context.Context, req *productplanpb.DeleteProductPlanRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_plan.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_plan.validation.data_required", "Product plan data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_plan.validation.id_required", "Product plan ID is required [DEFAULT]"))
	}
	return nil
}

// validateBusinessRules enforces business constraints for product plan deletion
func (uc *DeleteProductPlanUseCase) validateBusinessRules(ctx context.Context, req *productplanpb.DeleteProductPlanRequest) error {
	// Additional business rule validation can be added here
	// For example: check if product plan is referenced by active subscriptions
	if uc.isProductPlanInUse(ctx, req.Data.Id) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "product_plan.errors.in_use", "Product plan is currently in use and cannot be deleted [DEFAULT]"))
	}
	return nil
}

// isProductPlanInUse checks if the product plan is referenced by other entities (e.g., active subscriptions)
func (uc *DeleteProductPlanUseCase) isProductPlanInUse(ctx context.Context, productPlanID string) bool {
	// Placeholder for actual implementation
	// TODO: Implement actual check for product plan usage
	return false
}
