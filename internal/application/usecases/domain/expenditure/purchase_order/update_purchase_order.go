package purchaseorder

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	purchaseorderpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order"
)

// UpdatePurchaseOrderRepositories groups all repository dependencies
type UpdatePurchaseOrderRepositories struct {
	PurchaseOrder purchaseorderpb.PurchaseOrderDomainServiceServer
}

// UpdatePurchaseOrderServices groups all business service dependencies
type UpdatePurchaseOrderServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdatePurchaseOrderUseCase handles the business logic for updating purchase orders
type UpdatePurchaseOrderUseCase struct {
	repositories UpdatePurchaseOrderRepositories
	services     UpdatePurchaseOrderServices
}

// NewUpdatePurchaseOrderUseCase creates use case with grouped dependencies
func NewUpdatePurchaseOrderUseCase(
	repositories UpdatePurchaseOrderRepositories,
	services UpdatePurchaseOrderServices,
) *UpdatePurchaseOrderUseCase {
	return &UpdatePurchaseOrderUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update purchase order operation
func (uc *UpdatePurchaseOrderUseCase) Execute(ctx context.Context, req *purchaseorderpb.UpdatePurchaseOrderRequest) (*purchaseorderpb.UpdatePurchaseOrderResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityPurchaseOrder, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *purchaseorderpb.UpdatePurchaseOrderResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("purchase order update failed: %w", err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return uc.executeCore(ctx, req)
}

func (uc *UpdatePurchaseOrderUseCase) executeCore(ctx context.Context, req *purchaseorderpb.UpdatePurchaseOrderRequest) (*purchaseorderpb.UpdatePurchaseOrderResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order.validation.id_required", "Purchase order ID is required [DEFAULT]"))
	}

	// Set date_modified
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.PurchaseOrder.UpdatePurchaseOrder(ctx, req)
}
