package purchaseorder

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	purchaseorderpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order"
)

// GetPurchaseOrderItemPageDataRepositories groups all repository dependencies
type GetPurchaseOrderItemPageDataRepositories struct {
	PurchaseOrder purchaseorderpb.PurchaseOrderDomainServiceServer
}

// GetPurchaseOrderItemPageDataServices groups all business service dependencies
type GetPurchaseOrderItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetPurchaseOrderItemPageDataUseCase handles fetching full item detail page data for a purchase order
type GetPurchaseOrderItemPageDataUseCase struct {
	repositories GetPurchaseOrderItemPageDataRepositories
	services     GetPurchaseOrderItemPageDataServices
}

// NewGetPurchaseOrderItemPageDataUseCase creates use case with grouped dependencies
func NewGetPurchaseOrderItemPageDataUseCase(
	repositories GetPurchaseOrderItemPageDataRepositories,
	services GetPurchaseOrderItemPageDataServices,
) *GetPurchaseOrderItemPageDataUseCase {
	return &GetPurchaseOrderItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get purchase order item page data operation
func (uc *GetPurchaseOrderItemPageDataUseCase) Execute(ctx context.Context, req *purchaseorderpb.GetPurchaseOrderItemPageDataRequest) (*purchaseorderpb.GetPurchaseOrderItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityPurchaseOrder, entityid.ActionRead); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "purchase_order.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if uc.repositories.PurchaseOrder == nil {
		return nil, errors.New("purchase order repository is not available")
	}
	resp, err := uc.repositories.PurchaseOrder.GetPurchaseOrderItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "purchase_order.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load purchase order item")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *GetPurchaseOrderItemPageDataUseCase) validateInput(ctx context.Context, req *purchaseorderpb.GetPurchaseOrderItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "purchase_order.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.PurchaseOrderId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "purchase_order.validation.id_required", "[ERR-DEFAULT] Purchase order ID is required"))
	}
	return nil
}
