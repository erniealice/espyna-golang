package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/tenancy"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeTenancy creates all tenancy use cases from provider repositories.
func InitializeTenancy(
	repos *domain.TenancyRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idSvc ports.IDGenerator,
	actionGate *actiongate.ActionGatekeeper,
) (*tenancy.TenancyUseCases, error) {
	return tenancy.NewUseCases(
		tenancy.TenancyRepositories{
			TenantSubscription:  repos.TenantSubscription,
			TenantPaymentMethod: repos.TenantPaymentMethod,
			TenantInvoice:       repos.TenantInvoice,
		},
		authSvc,
		txSvc,
		i18nSvc,
		idSvc,
		actionGate,
	), nil
}
