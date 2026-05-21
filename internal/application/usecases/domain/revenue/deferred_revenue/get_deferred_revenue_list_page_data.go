package deferredrevenue

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	deferredrevenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/deferred_revenue"
)

// GetDeferredRevenueListPageDataRepositories groups all repository dependencies
type GetDeferredRevenueListPageDataRepositories struct {
	DeferredRevenue deferredrevenuepb.DeferredRevenueDomainServiceServer
}

// GetDeferredRevenueListPageDataServices groups all business service dependencies
type GetDeferredRevenueListPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// GetDeferredRevenueListPageDataUseCase handles fetching paginated, searchable deferred revenue list data
type GetDeferredRevenueListPageDataUseCase struct {
	repositories GetDeferredRevenueListPageDataRepositories
	services     GetDeferredRevenueListPageDataServices
}

// NewGetDeferredRevenueListPageDataUseCase creates use case with grouped dependencies
func NewGetDeferredRevenueListPageDataUseCase(
	repositories GetDeferredRevenueListPageDataRepositories,
	services GetDeferredRevenueListPageDataServices,
) *GetDeferredRevenueListPageDataUseCase {
	return &GetDeferredRevenueListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get deferred revenue list page data operation
func (uc *GetDeferredRevenueListPageDataUseCase) Execute(ctx context.Context, req *deferredrevenuepb.GetDeferredRevenueListPageDataRequest) (*deferredrevenuepb.GetDeferredRevenueListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityDeferredRevenue, ports.ActionList); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "deferred_revenue.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	if uc.repositories.DeferredRevenue == nil {
		return nil, errors.New("deferred revenue repository is not available")
	}
	resp, err := uc.repositories.DeferredRevenue.GetDeferredRevenueListPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "deferred_revenue.errors.get_list_page_data_failed", "[ERR-DEFAULT] Failed to load deferred revenue list")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}

func (uc *GetDeferredRevenueListPageDataUseCase) validateInput(ctx context.Context, req *deferredrevenuepb.GetDeferredRevenueListPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "deferred_revenue.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Pagination != nil && req.Pagination.Limit > 0 && req.Pagination.Limit > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "deferred_revenue.validation.invalid_pagination_limit", "[ERR-DEFAULT] Invalid pagination limit"))
	}
	if req.Search != nil && len(req.Search.Query) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "deferred_revenue.validation.search_query_too_long", "[ERR-DEFAULT] Search query is too long"))
	}
	return nil
}
