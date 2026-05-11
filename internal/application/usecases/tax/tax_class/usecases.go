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
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
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
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		ListTaxClasses: NewListTaxClassesUseCase(
			ListTaxClassesRepositories{TaxClass: repositories.TaxClass},
			ListTaxClassesServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		FindByCodeTaxClass: NewFindByCodeTaxClassUseCase(
			FindByCodeTaxClassRepositories{TaxClass: repositories.TaxClass},
			FindByCodeTaxClassServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
	}
}
