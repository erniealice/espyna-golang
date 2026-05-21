package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/tax"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeTax creates all tax use cases from provider repositories.
func InitializeTax(
	repos *domain.TaxRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*tax.TaxUseCases, error) {
	return tax.NewUseCases(
		tax.TaxRepositories{
			TaxAuthority:        repos.TaxAuthority,
			TaxRegistrationKind: repos.TaxRegistrationKind,
			TaxTreatment:        repos.TaxTreatment,
			TaxClass:            repos.TaxClass,
			TaxRate:             repos.TaxRate,
			TaxRegistration:     repos.TaxRegistration,
			// Cross-domain repos for ComputeTaxesForRevenue.
			Revenue:                repos.Revenue,
			RevenueLineItem:        repos.RevenueLineItem,
			RevenueTaxLine:         repos.RevenueTaxLine,
			Workspace:              repos.Workspace,
			Product:                repos.Product,
			WithholdingCertificate: repos.WithholdingCertificate,
		},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
	), nil
}
