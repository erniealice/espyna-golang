package revenue

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
)

// GetRevenueListPageDataRepositories groups all repository dependencies
type GetRevenueListPageDataRepositories struct {
	Revenue revenuepb.RevenueDomainServiceServer
}

// GetRevenueListPageDataServices groups all business service dependencies
type GetRevenueListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetRevenueListPageDataUseCase handles the business logic for fetching the revenue list page data
type GetRevenueListPageDataUseCase struct {
	repositories GetRevenueListPageDataRepositories
	services     GetRevenueListPageDataServices
}

// NewGetRevenueListPageDataUseCase creates a new GetRevenueListPageDataUseCase
func NewGetRevenueListPageDataUseCase(
	repositories GetRevenueListPageDataRepositories,
	services GetRevenueListPageDataServices,
) *GetRevenueListPageDataUseCase {
	return &GetRevenueListPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get revenue list page data operation
func (uc *GetRevenueListPageDataUseCase) Execute(ctx context.Context, req *revenuepb.GetRevenueListPageDataRequest) (*revenuepb.GetRevenueListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenue, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "revenue.validation.request_required", "Request is required [DEFAULT]"))
	}

	return uc.repositories.Revenue.GetRevenueListPageData(ctx, req)
}
