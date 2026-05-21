package purchaseorder

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	purchaseorderpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order"
)

// ReadPurchaseOrderRepositories groups all repository dependencies
type ReadPurchaseOrderRepositories struct {
	PurchaseOrder purchaseorderpb.PurchaseOrderDomainServiceServer
}

// ReadPurchaseOrderServices groups all business service dependencies
type ReadPurchaseOrderServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadPurchaseOrderUseCase handles the business logic for reading a purchase order
type ReadPurchaseOrderUseCase struct {
	repositories ReadPurchaseOrderRepositories
	services     ReadPurchaseOrderServices
}

// NewReadPurchaseOrderUseCase creates use case with grouped dependencies
func NewReadPurchaseOrderUseCase(
	repositories ReadPurchaseOrderRepositories,
	services ReadPurchaseOrderServices,
) *ReadPurchaseOrderUseCase {
	return &ReadPurchaseOrderUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read purchase order operation
func (uc *ReadPurchaseOrderUseCase) Execute(ctx context.Context, req *purchaseorderpb.ReadPurchaseOrderRequest) (*purchaseorderpb.ReadPurchaseOrderResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityPurchaseOrder, ports.ActionRead); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if uc.repositories.PurchaseOrder == nil {
		return nil, errors.New("purchase order repository is not available")
	}
	return uc.repositories.PurchaseOrder.ReadPurchaseOrder(ctx, req)
}

func (uc *ReadPurchaseOrderUseCase) validateInput(ctx context.Context, req *purchaseorderpb.ReadPurchaseOrderRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}
	return nil
}
