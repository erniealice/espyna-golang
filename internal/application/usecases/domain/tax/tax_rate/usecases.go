package tax_rate

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	taxratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_rate"
)

const entityTaxRate = "tax_rate"

// TaxRateRepositories groups all repository dependencies for tax_rate use cases.
type TaxRateRepositories struct {
	TaxRate taxratepb.TaxRateDomainServiceServer
}

// TaxRateServices groups all business service dependencies.
type TaxRateServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UseCases contains all tax_rate use cases.
type UseCases struct {
	ReadTaxRate           *ReadTaxRateUseCase
	ListTaxRates          *ListTaxRatesUseCase
	FindApplicableTaxRate *FindApplicableTaxRateUseCase
}

// NewUseCases creates a new collection of tax_rate use cases.
func NewUseCases(repositories TaxRateRepositories, services TaxRateServices) *UseCases {
	return &UseCases{
		ReadTaxRate: NewReadTaxRateUseCase(
			ReadTaxRateRepositories{TaxRate: repositories.TaxRate},
			ReadTaxRateServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		ListTaxRates: NewListTaxRatesUseCase(
			ListTaxRatesRepositories{TaxRate: repositories.TaxRate},
			ListTaxRatesServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		FindApplicableTaxRate: NewFindApplicableTaxRateUseCase(
			FindApplicableTaxRateRepositories{TaxRate: repositories.TaxRate},
			FindApplicableTaxRateServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
	}
}
