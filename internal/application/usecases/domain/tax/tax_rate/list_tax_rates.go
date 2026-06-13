package tax_rate

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	taxratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_rate"
)

// ListTaxRatesRepositories groups repository dependencies.
type ListTaxRatesRepositories struct {
	TaxRate taxratepb.TaxRateDomainServiceServer
}

// ListTaxRatesServices groups service dependencies.
type ListTaxRatesServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListTaxRatesUseCase handles listing tax rates.
type ListTaxRatesUseCase struct {
	repositories ListTaxRatesRepositories
	services     ListTaxRatesServices
}

// NewListTaxRatesUseCase creates a new ListTaxRatesUseCase.
func NewListTaxRatesUseCase(repositories ListTaxRatesRepositories, services ListTaxRatesServices) *ListTaxRatesUseCase {
	return &ListTaxRatesUseCase{repositories: repositories, services: services}
}

// Execute performs the list tax_rates operation.
func (uc *ListTaxRatesUseCase) Execute(ctx context.Context, req *taxratepb.ListTaxRatesRequest) (*taxratepb.ListTaxRatesResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityTaxRate,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"tax_rate.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.TaxRate.ListTaxRates(ctx, req)
}
