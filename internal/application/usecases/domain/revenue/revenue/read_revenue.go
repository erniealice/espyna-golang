package revenue

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
)

// ReadRevenueRepositories groups all repository dependencies
type ReadRevenueRepositories struct {
	Revenue revenuepb.RevenueDomainServiceServer
}

// ReadRevenueServices groups all business service dependencies
type ReadRevenueServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadRevenueUseCase handles the business logic for reading a revenue
type ReadRevenueUseCase struct {
	repositories ReadRevenueRepositories
	services     ReadRevenueServices
}

// NewReadRevenueUseCase creates use case with grouped dependencies
func NewReadRevenueUseCase(
	repositories ReadRevenueRepositories,
	services ReadRevenueServices,
) *ReadRevenueUseCase {
	return &ReadRevenueUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read revenue operation
func (uc *ReadRevenueUseCase) Execute(ctx context.Context, req *revenuepb.ReadRevenueRequest) (*revenuepb.ReadRevenueResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityRevenue,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue.validation.id_required", "Revenue ID is required [DEFAULT]"))
	}

	return uc.repositories.Revenue.ReadRevenue(ctx, req)
}
