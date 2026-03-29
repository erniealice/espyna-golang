package purchaseorder

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	purchaseorderpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order"
)

// ListPurchaseOrdersRepositories groups all repository dependencies
type ListPurchaseOrdersRepositories struct {
	PurchaseOrder purchaseorderpb.PurchaseOrderDomainServiceServer
}

// ListPurchaseOrdersServices groups all business service dependencies
type ListPurchaseOrdersServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListPurchaseOrdersUseCase handles the business logic for listing purchase orders
type ListPurchaseOrdersUseCase struct {
	repositories ListPurchaseOrdersRepositories
	services     ListPurchaseOrdersServices
}

// NewListPurchaseOrdersUseCase creates a new ListPurchaseOrdersUseCase
func NewListPurchaseOrdersUseCase(
	repositories ListPurchaseOrdersRepositories,
	services ListPurchaseOrdersServices,
) *ListPurchaseOrdersUseCase {
	return &ListPurchaseOrdersUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list purchase orders operation
func (uc *ListPurchaseOrdersUseCase) Execute(ctx context.Context, req *purchaseorderpb.ListPurchaseOrdersRequest) (*purchaseorderpb.ListPurchaseOrdersResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityPurchaseOrder, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order.validation.request_required", "Request is required [DEFAULT]"))
	}

	if uc.repositories.PurchaseOrder == nil {
		return nil, errors.New("purchase order repository is not available")
	}
	return uc.repositories.PurchaseOrder.ListPurchaseOrders(ctx, req)
}
