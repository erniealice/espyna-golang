package revenue

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
)

// DeleteRevenueRepositories groups all repository dependencies
type DeleteRevenueRepositories struct {
	Revenue revenuepb.RevenueDomainServiceServer
}

// DeleteRevenueServices groups all business service dependencies
type DeleteRevenueServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// DeleteRevenueUseCase handles the business logic for deleting revenues
type DeleteRevenueUseCase struct {
	repositories DeleteRevenueRepositories
	services     DeleteRevenueServices
}

// NewDeleteRevenueUseCase creates a new DeleteRevenueUseCase
func NewDeleteRevenueUseCase(
	repositories DeleteRevenueRepositories,
	services DeleteRevenueServices,
) *DeleteRevenueUseCase {
	return &DeleteRevenueUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete revenue operation
func (uc *DeleteRevenueUseCase) Execute(ctx context.Context, req *revenuepb.DeleteRevenueRequest) (*revenuepb.DeleteRevenueResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityRevenue, entityid.ActionDelete); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue.validation.id_required", "Revenue ID is required [DEFAULT]"))
	}

	return uc.repositories.Revenue.DeleteRevenue(ctx, req)
}
