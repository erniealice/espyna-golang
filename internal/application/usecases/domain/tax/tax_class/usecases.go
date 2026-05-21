package tax_class

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	taxclasspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_class"
)

const entityTaxClass = "tax_class"

// TaxClassRepositories groups all repository dependencies for tax_class use cases.
type TaxClassRepositories struct {
	TaxClass taxclasspb.TaxClassDomainServiceServer
}

// TaxClassServices groups all business service dependencies.
type TaxClassServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// UseCases contains all tax_class use cases.
type UseCases struct {
	ReadTaxClass       *ReadTaxClassUseCase
	ListTaxClasses     *ListTaxClassesUseCase
	FindByCodeTaxClass *FindByCodeTaxClassUseCase
}

// NewUseCases creates a new collection of tax_class use cases.
func NewUseCases(repositories TaxClassRepositories, services TaxClassServices) *UseCases {
	return &UseCases{
		ReadTaxClass: NewReadTaxClassUseCase(
			ReadTaxClassRepositories{TaxClass: repositories.TaxClass},
			ReadTaxClassServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		ListTaxClasses: NewListTaxClassesUseCase(
			ListTaxClassesRepositories{TaxClass: repositories.TaxClass},
			ListTaxClassesServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		FindByCodeTaxClass: NewFindByCodeTaxClassUseCase(
			FindByCodeTaxClassRepositories{TaxClass: repositories.TaxClass},
			FindByCodeTaxClassServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
	}
}
