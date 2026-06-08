package deferredrevenue

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	deferredrevenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/deferred_revenue"
)

// ListDeferredRevenuesRepositories groups all repository dependencies
type ListDeferredRevenuesRepositories struct {
	DeferredRevenue deferredrevenuepb.DeferredRevenueDomainServiceServer
}

// ListDeferredRevenuesServices groups all business service dependencies
type ListDeferredRevenuesServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListDeferredRevenuesUseCase handles the business logic for listing deferred revenues
type ListDeferredRevenuesUseCase struct {
	repositories ListDeferredRevenuesRepositories
	services     ListDeferredRevenuesServices
}

// NewListDeferredRevenuesUseCase creates a new ListDeferredRevenuesUseCase
func NewListDeferredRevenuesUseCase(
	repositories ListDeferredRevenuesRepositories,
	services ListDeferredRevenuesServices,
) *ListDeferredRevenuesUseCase {
	return &ListDeferredRevenuesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list deferred revenues operation
func (uc *ListDeferredRevenuesUseCase) Execute(ctx context.Context, req *deferredrevenuepb.ListDeferredRevenuesRequest) (*deferredrevenuepb.ListDeferredRevenuesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityDeferredRevenue, entityid.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "deferred_revenue.validation.request_required", "Request is required [DEFAULT]"))
	}

	if uc.repositories.DeferredRevenue == nil {
		return nil, errors.New("deferred revenue repository is not available")
	}
	return uc.repositories.DeferredRevenue.ListDeferredRevenues(ctx, req)
}
