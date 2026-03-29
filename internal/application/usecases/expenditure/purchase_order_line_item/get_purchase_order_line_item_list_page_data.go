package purchaseorderlineitem

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	purchaseorderlineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order_line_item"
)

// GetPurchaseOrderLineItemListPageDataRepositories groups all repository dependencies
type GetPurchaseOrderLineItemListPageDataRepositories struct {
	PurchaseOrderLineItem purchaseorderlineitempb.PurchaseOrderLineItemDomainServiceServer
}

// GetPurchaseOrderLineItemListPageDataServices groups all business service dependencies
type GetPurchaseOrderLineItemListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetPurchaseOrderLineItemListPageDataUseCase handles fetching paginated, searchable purchase order line item list data
type GetPurchaseOrderLineItemListPageDataUseCase struct {
	repositories GetPurchaseOrderLineItemListPageDataRepositories
	services     GetPurchaseOrderLineItemListPageDataServices
}

// NewGetPurchaseOrderLineItemListPageDataUseCase creates use case with grouped dependencies
func NewGetPurchaseOrderLineItemListPageDataUseCase(
	repositories GetPurchaseOrderLineItemListPageDataRepositories,
	services GetPurchaseOrderLineItemListPageDataServices,
) *GetPurchaseOrderLineItemListPageDataUseCase {
	return &GetPurchaseOrderLineItemListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get purchase order line item list page data operation
func (uc *GetPurchaseOrderLineItemListPageDataUseCase) Execute(ctx context.Context, req *purchaseorderlineitempb.GetPurchaseOrderLineItemListPageDataRequest) (*purchaseorderlineitempb.GetPurchaseOrderLineItemListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityPurchaseOrderLineItem, ports.ActionList); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order_line_item.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if uc.repositories.PurchaseOrderLineItem == nil {
		return nil, errors.New("purchase order line item repository is not available")
	}
	resp, err := uc.repositories.PurchaseOrderLineItem.GetPurchaseOrderLineItemListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order_line_item.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load purchase order line item list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *GetPurchaseOrderLineItemListPageDataUseCase) validateInput(ctx context.Context, req *purchaseorderlineitempb.GetPurchaseOrderLineItemListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order_line_item.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Pagination != nil && req.Pagination.Limit > 0 && req.Pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order_line_item.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
	}
	if req.Search != nil && len(req.Search.Query) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order_line_item.validation.search_query_too_long", "[ERR-DEFAULT] Search query is too long"))
	}
	return nil
}
