package purchaseorder

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	purchaseorderpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order"
)

// ListPurchaseOrdersRepositories groups all repository dependencies
type ListPurchaseOrdersRepositories struct {
	PurchaseOrder purchaseorderpb.PurchaseOrderDomainServiceServer
}

// ListPurchaseOrdersServices groups all business service dependencies
type ListPurchaseOrdersServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityPurchaseOrder,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "purchase_order.validation.request_required", "Request is required [DEFAULT]"))
	}

	if uc.repositories.PurchaseOrder == nil {
		return nil, errors.New("purchase order repository is not available")
	}
	return uc.repositories.PurchaseOrder.ListPurchaseOrders(ctx, req)
}
