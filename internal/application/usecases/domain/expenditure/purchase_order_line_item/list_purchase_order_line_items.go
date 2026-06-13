package purchaseorderlineitem

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	purchaseorderlineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order_line_item"
)

// ListPurchaseOrderLineItemsRepositories groups all repository dependencies
type ListPurchaseOrderLineItemsRepositories struct {
	PurchaseOrderLineItem purchaseorderlineitempb.PurchaseOrderLineItemDomainServiceServer
}

// ListPurchaseOrderLineItemsServices groups all business service dependencies
type ListPurchaseOrderLineItemsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListPurchaseOrderLineItemsUseCase handles the business logic for listing purchase order line items
type ListPurchaseOrderLineItemsUseCase struct {
	repositories ListPurchaseOrderLineItemsRepositories
	services     ListPurchaseOrderLineItemsServices
}

// NewListPurchaseOrderLineItemsUseCase creates a new ListPurchaseOrderLineItemsUseCase
func NewListPurchaseOrderLineItemsUseCase(
	repositories ListPurchaseOrderLineItemsRepositories,
	services ListPurchaseOrderLineItemsServices,
) *ListPurchaseOrderLineItemsUseCase {
	return &ListPurchaseOrderLineItemsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list purchase order line items operation
func (uc *ListPurchaseOrderLineItemsUseCase) Execute(ctx context.Context, req *purchaseorderlineitempb.ListPurchaseOrderLineItemsRequest) (*purchaseorderlineitempb.ListPurchaseOrderLineItemsResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityPurchaseOrderLineItem,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "purchase_order_line_item.validation.request_required", "Request is required [DEFAULT]"))
	}

	if uc.repositories.PurchaseOrderLineItem == nil {
		return nil, errors.New("purchase order line item repository is not available")
	}
	return uc.repositories.PurchaseOrderLineItem.ListPurchaseOrderLineItems(ctx, req)
}
