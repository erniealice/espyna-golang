package purchaseorderlineitem

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	purchaseorderlineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order_line_item"
)

// ListPurchaseOrderLineItemsRepositories groups all repository dependencies
type ListPurchaseOrderLineItemsRepositories struct {
	PurchaseOrderLineItem purchaseorderlineitempb.PurchaseOrderLineItemDomainServiceServer
}

// ListPurchaseOrderLineItemsServices groups all business service dependencies
type ListPurchaseOrderLineItemsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityPurchaseOrderLineItem, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order_line_item.validation.request_required", "Request is required [DEFAULT]"))
	}

	if uc.repositories.PurchaseOrderLineItem == nil {
		return nil, errors.New("purchase order line item repository is not available")
	}
	return uc.repositories.PurchaseOrderLineItem.ListPurchaseOrderLineItems(ctx, req)
}
