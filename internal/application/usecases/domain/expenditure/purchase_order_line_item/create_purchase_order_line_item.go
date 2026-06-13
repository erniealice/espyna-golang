package purchaseorderlineitem

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	purchaseorderlineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order_line_item"
)

const entityPurchaseOrderLineItem = "purchase_order_line_item"

// CreatePurchaseOrderLineItemRepositories groups all repository dependencies
type CreatePurchaseOrderLineItemRepositories struct {
	PurchaseOrderLineItem purchaseorderlineitempb.PurchaseOrderLineItemDomainServiceServer
}

// CreatePurchaseOrderLineItemServices groups all business service dependencies
type CreatePurchaseOrderLineItemServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreatePurchaseOrderLineItemUseCase handles the business logic for creating purchase order line items
type CreatePurchaseOrderLineItemUseCase struct {
	repositories CreatePurchaseOrderLineItemRepositories
	services     CreatePurchaseOrderLineItemServices
}

// NewCreatePurchaseOrderLineItemUseCase creates use case with grouped dependencies
func NewCreatePurchaseOrderLineItemUseCase(
	repositories CreatePurchaseOrderLineItemRepositories,
	services CreatePurchaseOrderLineItemServices,
) *CreatePurchaseOrderLineItemUseCase {
	return &CreatePurchaseOrderLineItemUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create purchase order line item operation
func (uc *CreatePurchaseOrderLineItemUseCase) Execute(ctx context.Context, req *purchaseorderlineitempb.CreatePurchaseOrderLineItemRequest) (*purchaseorderlineitempb.CreatePurchaseOrderLineItemResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityPurchaseOrderLineItem,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *purchaseorderlineitempb.CreatePurchaseOrderLineItemResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("purchase order line item creation failed: %w", err)
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

func (uc *CreatePurchaseOrderLineItemUseCase) executeCore(ctx context.Context, req *purchaseorderlineitempb.CreatePurchaseOrderLineItemRequest) (*purchaseorderlineitempb.CreatePurchaseOrderLineItemResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "purchase_order_line_item.validation.data_required", "Purchase order line item data is required [DEFAULT]"))
	}

	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.Active = true

	return uc.repositories.PurchaseOrderLineItem.CreatePurchaseOrderLineItem(ctx, req)
}
