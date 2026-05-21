package purchaseorder

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	purchaseorderpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order"
)

// DeletePurchaseOrderRepositories groups all repository dependencies
type DeletePurchaseOrderRepositories struct {
	PurchaseOrder purchaseorderpb.PurchaseOrderDomainServiceServer
}

// DeletePurchaseOrderServices groups all business service dependencies
type DeletePurchaseOrderServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeletePurchaseOrderUseCase handles the business logic for deleting purchase orders
type DeletePurchaseOrderUseCase struct {
	repositories DeletePurchaseOrderRepositories
	services     DeletePurchaseOrderServices
}

// NewDeletePurchaseOrderUseCase creates a new DeletePurchaseOrderUseCase
func NewDeletePurchaseOrderUseCase(
	repositories DeletePurchaseOrderRepositories,
	services DeletePurchaseOrderServices,
) *DeletePurchaseOrderUseCase {
	return &DeletePurchaseOrderUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete purchase order operation
func (uc *DeletePurchaseOrderUseCase) Execute(ctx context.Context, req *purchaseorderpb.DeletePurchaseOrderRequest) (*purchaseorderpb.DeletePurchaseOrderResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityPurchaseOrder, ports.ActionDelete); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order.validation.id_required", "Purchase order ID is required [DEFAULT]"))
	}

	return uc.repositories.PurchaseOrder.DeletePurchaseOrder(ctx, req)
}
