package purchaseorder

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	purchaseorderpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/purchase_order"
)

// GetPurchaseOrderListPageDataRepositories groups all repository dependencies
type GetPurchaseOrderListPageDataRepositories struct {
	PurchaseOrder purchaseorderpb.PurchaseOrderDomainServiceServer
}

// GetPurchaseOrderListPageDataServices groups all business service dependencies
type GetPurchaseOrderListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetPurchaseOrderListPageDataUseCase handles fetching paginated, searchable purchase order list data
type GetPurchaseOrderListPageDataUseCase struct {
	repositories GetPurchaseOrderListPageDataRepositories
	services     GetPurchaseOrderListPageDataServices
}

// NewGetPurchaseOrderListPageDataUseCase creates use case with grouped dependencies
func NewGetPurchaseOrderListPageDataUseCase(
	repositories GetPurchaseOrderListPageDataRepositories,
	services GetPurchaseOrderListPageDataServices,
) *GetPurchaseOrderListPageDataUseCase {
	return &GetPurchaseOrderListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get purchase order list page data operation
func (uc *GetPurchaseOrderListPageDataUseCase) Execute(ctx context.Context, req *purchaseorderpb.GetPurchaseOrderListPageDataRequest) (*purchaseorderpb.GetPurchaseOrderListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityPurchaseOrder, ports.ActionList); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if uc.repositories.PurchaseOrder == nil {
		return nil, errors.New("purchase order repository is not available")
	}
	resp, err := uc.repositories.PurchaseOrder.GetPurchaseOrderListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load purchase order list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *GetPurchaseOrderListPageDataUseCase) validateInput(ctx context.Context, req *purchaseorderpb.GetPurchaseOrderListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Pagination != nil && req.Pagination.Limit > 0 && req.Pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
	}
	if req.Search != nil && len(req.Search.Query) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "purchase_order.validation.search_query_too_long", "[ERR-DEFAULT] Search query is too long"))
	}
	return nil
}
