package purchaseorderlineitem

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	purchaseorderlineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order_line_item"
)

// DeletePurchaseOrderLineItemRepositories groups all repository dependencies
type DeletePurchaseOrderLineItemRepositories struct {
	PurchaseOrderLineItem purchaseorderlineitempb.PurchaseOrderLineItemDomainServiceServer
}

// DeletePurchaseOrderLineItemServices groups all business service dependencies
type DeletePurchaseOrderLineItemServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
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
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityPurchaseOrderLineItem, ports.ActionDelete); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "purchase_order_line_item.validation.id_required", "Purchase order line item ID is required [DEFAULT]"))
	}

	return uc.repositories.PurchaseOrderLineItem.DeletePurchaseOrderLineItem(ctx, req)
}
