package purchaseorderlineitem

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	purchaseorderlineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order_line_item"
)

// DeletePurchaseOrderLineItemRepositories groups all repository dependencies
type DeletePurchaseOrderLineItemRepositories struct {
	PurchaseOrderLineItem purchaseorderlineitempb.PurchaseOrderLineItemDomainServiceServer
}

// DeletePurchaseOrderLineItemServices groups all business service dependencies
type DeletePurchaseOrderLineItemServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeletePurchaseOrderLineItemUseCase handles the business logic for deleting purchase order line items
type DeletePurchaseOrderLineItemUseCase struct {
	repositories DeletePurchaseOrderLineItemRepositories
	services     DeletePurchaseOrderLineItemServices
}

// NewDeletePurchaseOrderLineItemUseCase creates a new DeletePurchaseOrderLineItemUseCase
func NewDeletePurchaseOrderLineItemUseCase(
	repositories DeletePurchaseOrderLineItemRepositories,
	services DeletePurchaseOrderLineItemServices,
) *DeletePurchaseOrderLineItemUseCase {
	return &DeletePurchaseOrderLineItemUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete purchase order line item operation
func (uc *DeletePurchaseOrderLineItemUseCase) Execute(ctx context.Context, req *purchaseorderlineitempb.DeletePurchaseOrderLineItemRequest) (*purchaseorderlineitempb.DeletePurchaseOrderLineItemResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityPurchaseOrderLineItem, ports.ActionDelete); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order_line_item.validation.id_required", "Purchase order line item ID is required [DEFAULT]"))
	}

	return uc.repositories.PurchaseOrderLineItem.DeletePurchaseOrderLineItem(ctx, req)
}
