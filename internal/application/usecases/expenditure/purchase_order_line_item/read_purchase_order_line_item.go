package purchaseorderlineitem

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	purchaseorderlineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order_line_item"
)

// ReadPurchaseOrderLineItemRepositories groups all repository dependencies
type ReadPurchaseOrderLineItemRepositories struct {
	PurchaseOrderLineItem purchaseorderlineitempb.PurchaseOrderLineItemDomainServiceServer
}

// ReadPurchaseOrderLineItemServices groups all business service dependencies
type ReadPurchaseOrderLineItemServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadPurchaseOrderLineItemUseCase handles the business logic for reading a purchase order line item
type ReadPurchaseOrderLineItemUseCase struct {
	repositories ReadPurchaseOrderLineItemRepositories
	services     ReadPurchaseOrderLineItemServices
}

// NewReadPurchaseOrderLineItemUseCase creates use case with grouped dependencies
func NewReadPurchaseOrderLineItemUseCase(
	repositories ReadPurchaseOrderLineItemRepositories,
	services ReadPurchaseOrderLineItemServices,
) *ReadPurchaseOrderLineItemUseCase {
	return &ReadPurchaseOrderLineItemUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read purchase order line item operation
func (uc *ReadPurchaseOrderLineItemUseCase) Execute(ctx context.Context, req *purchaseorderlineitempb.ReadPurchaseOrderLineItemRequest) (*purchaseorderlineitempb.ReadPurchaseOrderLineItemResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityPurchaseOrderLineItem, ports.ActionRead); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if uc.repositories.PurchaseOrderLineItem == nil {
		return nil, errors.New("purchase order line item repository is not available")
	}
	return uc.repositories.PurchaseOrderLineItem.ReadPurchaseOrderLineItem(ctx, req)
}

func (uc *ReadPurchaseOrderLineItemUseCase) validateInput(ctx context.Context, req *purchaseorderlineitempb.ReadPurchaseOrderLineItemRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order_line_item.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order_line_item.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order_line_item.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}
	return nil
}
