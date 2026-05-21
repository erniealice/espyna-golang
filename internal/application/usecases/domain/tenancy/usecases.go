// Package tenancy contains use cases for the tenancy domain.
// These entities model the Ichizen platform subscription, payment methods,
// and invoices for a workspace tenant (billing-side records owned by the
// Ichizen platform itself, not the workspace's customers/suppliers).
package tenancy

import (
	// Tenancy use cases
	tenantInvoiceUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/tenancy/tenant_invoice"
	tenantPaymentMethodUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/tenancy/tenant_payment_method"
	tenantSubscriptionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/tenancy/tenant_subscription"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Protobuf domain services
	tenantinvoicepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tenancy/tenant_invoice"
	tenantpaymentmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tenancy/tenant_payment_method"
	tenantsubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tenancy/tenant_subscription"
)

// TenancyRepositories contains all tenancy domain repositories.
type TenancyRepositories struct {
	TenantSubscription  tenantsubscriptionpb.TenantSubscriptionDomainServiceServer
	TenantPaymentMethod tenantpaymentmethodpb.TenantPaymentMethodDomainServiceServer
	TenantInvoice       tenantinvoicepb.TenantInvoiceDomainServiceServer
}

// TenancyUseCases contains all tenancy-related use cases.
type TenancyUseCases struct {
	TenantSubscription  *tenantSubscriptionUseCases.UseCases
	TenantPaymentMethod *tenantPaymentMethodUseCases.UseCases
	TenantInvoice       *tenantInvoiceUseCases.UseCases
}

// NewUseCases creates all tenancy use cases with proper constructor injection.
func NewUseCases(
	repos TenancyRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idService ports.IDGenerator,
) *TenancyUseCases {
	svcSub := tenantSubscriptionUseCases.TenantSubscriptionServices{
		Authorizer:  authSvc,
		Transactor:  txSvc,
		Translator:  i18nSvc,
		IDGenerator: idService,
	}
	svcPM := tenantPaymentMethodUseCases.TenantPaymentMethodServices{
		Authorizer:  authSvc,
		Transactor:  txSvc,
		Translator:  i18nSvc,
		IDGenerator: idService,
	}
	svcInv := tenantInvoiceUseCases.TenantInvoiceServices{
		Authorizer:  authSvc,
		Transactor:  txSvc,
		Translator:  i18nSvc,
		IDGenerator: idService,
	}

	return &TenancyUseCases{
		TenantSubscription: tenantSubscriptionUseCases.NewUseCases(
			tenantSubscriptionUseCases.TenantSubscriptionRepositories{TenantSubscription: repos.TenantSubscription},
			svcSub,
		),
		TenantPaymentMethod: tenantPaymentMethodUseCases.NewUseCases(
			tenantPaymentMethodUseCases.TenantPaymentMethodRepositories{TenantPaymentMethod: repos.TenantPaymentMethod},
			svcPM,
		),
		TenantInvoice: tenantInvoiceUseCases.NewUseCases(
			tenantInvoiceUseCases.TenantInvoiceRepositories{TenantInvoice: repos.TenantInvoice},
			svcInv,
		),
	}
}
