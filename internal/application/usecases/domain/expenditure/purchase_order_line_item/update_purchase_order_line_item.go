package purchaseorderlineitem

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	purchaseorderlineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order_line_item"
)

// UpdatePurchaseOrderLineItemRepositories groups all repository dependencies
type UpdatePurchaseOrderLineItemRepositories struct {
	PurchaseOrderLineItem purchaseorderlineitempb.PurchaseOrderLineItemDomainServiceServer
}

// UpdatePurchaseOrderLineItemServices groups all business service dependencies
type UpdatePurchaseOrderLineItemServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdatePurchaseOrderLineItemUseCase handles the business logic for updating purchase order line items
type UpdatePurchaseOrderLineItemUseCase struct {
	repositories UpdatePurchaseOrderLineItemRepositories
	services     UpdatePurchaseOrderLineItemServices
}

// NewUpdatePurchaseOrderLineItemUseCase creates use case with grouped dependencies
func NewUpdatePurchaseOrderLineItemUseCase(
	repositories UpdatePurchaseOrderLineItemRepositories,
	services UpdatePurchaseOrderLineItemServices,
) *UpdatePurchaseOrderLineItemUseCase {
	return &UpdatePurchaseOrderLineItemUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update purchase order line item operation
func (uc *UpdatePurchaseOrderLineItemUseCase) Execute(ctx context.Context, req *purchaseorderlineitempb.UpdatePurchaseOrderLineItemRequest) (*purchaseorderlineitempb.UpdatePurchaseOrderLineItemResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityPurchaseOrderLineItem, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *purchaseorderlineitempb.UpdatePurchaseOrderLineItemResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("purchase order line item update failed: %w", err)
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

func (uc *UpdatePurchaseOrderLineItemUseCase) executeCore(ctx context.Context, req *purchaseorderlineitempb.UpdatePurchaseOrderLineItemRequest) (*purchaseorderlineitempb.UpdatePurchaseOrderLineItemResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order_line_item.validation.id_required", "Purchase order line item ID is required [DEFAULT]"))
	}

	// Set date_modified
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.PurchaseOrderLineItem.UpdatePurchaseOrderLineItem(ctx, req)
}
