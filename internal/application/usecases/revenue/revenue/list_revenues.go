package revenue

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
)

// ListRevenuesRepositories groups all repository dependencies
type ListRevenuesRepositories struct {
	Revenue revenuepb.RevenueDomainServiceServer
}

// ListRevenuesServices groups all business service dependencies
type ListRevenuesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListRevenuesUseCase handles the business logic for listing revenues
type ListRevenuesUseCase struct {
	repositories ListRevenuesRepositories
	services     ListRevenuesServices
}

// NewListRevenuesUseCase creates a new ListRevenuesUseCase
func NewListRevenuesUseCase(
	repositories ListRevenuesRepositories,
	services ListRevenuesServices,
) *ListRevenuesUseCase {
	return &ListRevenuesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list revenues operation
func (uc *ListRevenuesUseCase) Execute(ctx context.Context, req *revenuepb.ListRevenuesRequest) (*revenuepb.ListRevenuesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenue, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "revenue.validation.request_required", "Request is required [DEFAULT]"))
	}

	return uc.repositories.Revenue.ListRevenues(ctx, req)
}
