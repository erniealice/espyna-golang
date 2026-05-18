package purchaseorderlineitem

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	purchaseorderlineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order_line_item"
)

// GetPurchaseOrderLineItemItemPageDataRepositories groups all repository dependencies
type GetPurchaseOrderLineItemItemPageDataRepositories struct {
	PurchaseOrderLineItem purchaseorderlineitempb.PurchaseOrderLineItemDomainServiceServer
}

// GetPurchaseOrderLineItemItemPageDataServices groups all business service dependencies
type GetPurchaseOrderLineItemItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetPurchaseOrderLineItemItemPageDataUseCase handles the business logic for getting a single purchase order line item page data
type GetPurchaseOrderLineItemItemPageDataUseCase struct {
	repositories GetPurchaseOrderLineItemItemPageDataRepositories
	services     GetPurchaseOrderLineItemItemPageDataServices
}

// NewGetPurchaseOrderLineItemItemPageDataUseCase creates a new use case
func NewGetPurchaseOrderLineItemItemPageDataUseCase(
	repositories GetPurchaseOrderLineItemItemPageDataRepositories,
	services GetPurchaseOrderLineItemItemPageDataServices,
) *GetPurchaseOrderLineItemItemPageDataUseCase {
	return &GetPurchaseOrderLineItemItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get purchase order line item item page data operation
func (uc *GetPurchaseOrderLineItemItemPageDataUseCase) Execute(ctx context.Context, req *purchaseorderlineitempb.GetPurchaseOrderLineItemItemPageDataRequest) (*purchaseorderlineitempb.GetPurchaseOrderLineItemItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityPurchaseOrderLineItem, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.PurchaseOrderLineItemId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order_line_item.validation.id_required", "Purchase order line item ID is required [DEFAULT]"))
	}

	if uc.repositories.PurchaseOrderLineItem == nil {
		return nil, errors.New("purchase order line item repository is not available")
	}
	return uc.repositories.PurchaseOrderLineItem.GetPurchaseOrderLineItemItemPageData(ctx, req)
}
