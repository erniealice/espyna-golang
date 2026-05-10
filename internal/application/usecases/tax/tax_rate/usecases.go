package tax_rate

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	taxratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_rate"
)

const entityTaxRate = "tax_rate"

// TaxRateRepositories groups all repository dependencies for tax_rate use cases.
type TaxRateRepositories struct {
	TaxRate taxratepb.TaxRateDomainServiceServer
}

// TaxRateServices groups all business service dependencies.
type TaxRateServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// UseCases contains all tax_rate use cases.
type UseCases struct {
	ReadTaxRate          *ReadTaxRateUseCase
	ListTaxRates         *ListTaxRatesUseCase
	FindApplicableTaxRate *FindApplicableTaxRateUseCase
}

// NewUseCases creates a new collection of tax_rate use cases.
func NewUseCases(repositories TaxRateRepositories, services TaxRateServices) *UseCases {
	return &UseCases{
		ReadTaxRate: NewReadTaxRateUseCase(
			ReadTaxRateRepositories{TaxRate: repositories.TaxRate},
			ReadTaxRateServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		ListTaxRates: NewListTaxRatesUseCase(
			ListTaxRatesRepositories{TaxRate: repositories.TaxRate},
			ListTaxRatesServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		FindApplicableTaxRate: NewFindApplicableTaxRateUseCase(
			FindApplicableTaxRateRepositories{TaxRate: repositories.TaxRate},
			FindApplicableTaxRateServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
	}
}
