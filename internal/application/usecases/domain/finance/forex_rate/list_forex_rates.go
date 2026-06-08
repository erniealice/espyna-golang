package forex_rate

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	forexratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/finance/forex_rate"
)

// ListForexRatesRepositories groups repository dependencies.
type ListForexRatesRepositories struct {
	ForexRate forexratepb.ForexRateDomainServiceServer
}

// ListForexRatesServices groups service dependencies.
type ListForexRatesServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// ListForexRatesUseCase handles listing forex rates.
type ListForexRatesUseCase struct {
	repositories ListForexRatesRepositories
	services     ListForexRatesServices
}

// NewListForexRatesUseCase creates a new ListForexRatesUseCase.
func NewListForexRatesUseCase(repositories ListForexRatesRepositories, services ListForexRatesServices) *ListForexRatesUseCase {
	return &ListForexRatesUseCase{repositories: repositories, services: services}
}

// Execute performs the list forex_rates operation.
func (uc *ListForexRatesUseCase) Execute(ctx context.Context, req *forexratepb.ListForexRatesRequest) (*forexratepb.ListForexRatesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityForexRate, entityid.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"forex_rate.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.ForexRate.ListForexRates(ctx, req)
}
