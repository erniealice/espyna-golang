package purchaseorder

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
	purchaseorderpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order"
)

const entityPurchaseOrder = "purchase_order"

// CreatePurchaseOrderRepositories groups all repository dependencies
type CreatePurchaseOrderRepositories struct {
	PurchaseOrder purchaseorderpb.PurchaseOrderDomainServiceServer
	PaymentTerm   paymenttermpb.PaymentTermDomainServiceServer
}

// CreatePurchaseOrderServices groups all business service dependencies
type CreatePurchaseOrderServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreatePurchaseOrderUseCase handles the business logic for creating purchase orders
type CreatePurchaseOrderUseCase struct {
	repositories CreatePurchaseOrderRepositories
	services     CreatePurchaseOrderServices
}

// NewCreatePurchaseOrderUseCase creates use case with grouped dependencies
func NewCreatePurchaseOrderUseCase(
	repositories CreatePurchaseOrderRepositories,
	services CreatePurchaseOrderServices,
) *CreatePurchaseOrderUseCase {
	return &CreatePurchaseOrderUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create purchase order operation
func (uc *CreatePurchaseOrderUseCase) Execute(ctx context.Context, req *purchaseorderpb.CreatePurchaseOrderRequest) (*purchaseorderpb.CreatePurchaseOrderResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityPurchaseOrder, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *purchaseorderpb.CreatePurchaseOrderResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "purchase_order.errors.creation_failed", "Purchase order creation failed [DEFAULT]")
				return fmt.Errorf("%s: %w", translatedError, err)
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

func (uc *CreatePurchaseOrderUseCase) executeCore(ctx context.Context, req *purchaseorderpb.CreatePurchaseOrderRequest) (*purchaseorderpb.CreatePurchaseOrderResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if err := uc.enrichPurchaseOrderData(req.Data); err != nil {
		return nil, err
	}

	if uc.repositories.PurchaseOrder == nil {
		return nil, errors.New("purchase order repository is not available")
	}
	return uc.repositories.PurchaseOrder.CreatePurchaseOrder(ctx, req)
}

func (uc *CreatePurchaseOrderUseCase) validateInput(ctx context.Context, req *purchaseorderpb.CreatePurchaseOrderRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order.validation.data_required", "[ERR-DEFAULT] Purchase order data is required"))
	}
	return nil
}

func (uc *CreatePurchaseOrderUseCase) enrichPurchaseOrderData(p *purchaseorderpb.PurchaseOrder) error {
	now := time.Now()
	if p.Id == "" {
		p.Id = uc.services.IDService.GenerateID()
	}
	p.DateCreated = &[]int64{now.UnixMilli()}[0]
	p.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	p.DateModified = &[]int64{now.UnixMilli()}[0]
	p.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	p.Active = true
	return nil
}
