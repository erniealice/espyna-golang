package tax

import (
	// Tax use cases
	computeTaxesUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/tax/compute_taxes_for_revenue"
	taxAuthorityUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/tax/tax_authority"
	taxClassUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/tax/tax_class"
	taxRateUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/tax/tax_rate"
	taxRegistrationUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/tax/tax_registration"
	taxRegistrationKindUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/tax/tax_registration_kind"
	taxTreatmentUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/tax/tax_treatment"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"

	// Protobuf domain services — entity
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"

	// Protobuf domain services — product
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"

	// Protobuf domain services — revenue
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenuelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
	revenuetaxlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_tax_line"

	// Protobuf domain services — tax
	taxauthoritypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_authority"
	taxclasspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_class"
	taxratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_rate"
	taxregistrationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration"
	taxregistrationkindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration_kind"
	taxtreatmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_treatment"

	// Protobuf domain services — treasury
	withholdingcertificatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/withholding_certificate"
)

// TaxRepositories contains all tax domain repositories.
// Also carries cross-domain repos needed by ComputeTaxesForRevenue.
type TaxRepositories struct {
	TaxAuthority        taxauthoritypb.TaxAuthorityDomainServiceServer
	TaxRegistrationKind taxregistrationkindpb.TaxRegistrationKindDomainServiceServer
	TaxTreatment        taxtreatmentpb.TaxTreatmentDomainServiceServer
	TaxClass            taxclasspb.TaxClassDomainServiceServer
	TaxRate             taxratepb.TaxRateDomainServiceServer
	TaxRegistration     taxregistrationpb.TaxRegistrationDomainServiceServer

	// Cross-domain repos for ComputeTaxesForRevenue (all optional — nil = feature disabled).
	Revenue                revenuepb.RevenueDomainServiceServer
	RevenueLineItem        revenuelineitempb.RevenueLineItemDomainServiceServer
	RevenueTaxLine         revenuetaxlinepb.RevenueTaxLineDomainServiceServer
	Workspace              workspacepb.WorkspaceDomainServiceServer
	Product                productpb.ProductDomainServiceServer
	WithholdingCertificate withholdingcertificatepb.WithholdingCertificateDomainServiceServer
}

// TaxUseCases contains all tax-related use cases.
type TaxUseCases struct {
	TaxAuthority        *taxAuthorityUseCases.UseCases
	TaxRegistrationKind *taxRegistrationKindUseCases.UseCases
	TaxTreatment        *taxTreatmentUseCases.UseCases
	TaxClass            *taxClassUseCases.UseCases
	TaxRate             *taxRateUseCases.UseCases
	TaxRegistration     *taxRegistrationUseCases.UseCases
	ComputeTaxes        *computeTaxesUseCases.UseCases
}

// NewUseCases creates all tax use cases with proper constructor injection.
func NewUseCases(
	repos TaxRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idService ports.IDGenerator,
	actionGate *actiongate.ActionGatekeeper,
) *TaxUseCases {
	taxAuthorityUC := taxAuthorityUseCases.NewUseCases(
		taxAuthorityUseCases.TaxAuthorityRepositories{
			TaxAuthority: repos.TaxAuthority,
		},
		taxAuthorityUseCases.TaxAuthorityServices{
			Authorizer:       authSvc,
			Translator:       i18nSvc,
			ActionGatekeeper: actionGate,
		},
	)

	taxRegistrationKindUC := taxRegistrationKindUseCases.NewUseCases(
		taxRegistrationKindUseCases.TaxRegistrationKindRepositories{
			TaxRegistrationKind: repos.TaxRegistrationKind,
		},
		taxRegistrationKindUseCases.TaxRegistrationKindServices{
			Authorizer:       authSvc,
			Translator:       i18nSvc,
			ActionGatekeeper: actionGate,
		},
	)

	taxTreatmentUC := taxTreatmentUseCases.NewUseCases(
		taxTreatmentUseCases.TaxTreatmentRepositories{
			TaxTreatment: repos.TaxTreatment,
		},
		taxTreatmentUseCases.TaxTreatmentServices{
			Authorizer:       authSvc,
			Translator:       i18nSvc,
			ActionGatekeeper: actionGate,
		},
	)

	taxClassUC := taxClassUseCases.NewUseCases(
		taxClassUseCases.TaxClassRepositories{
			TaxClass: repos.TaxClass,
		},
		taxClassUseCases.TaxClassServices{
			Authorizer:       authSvc,
			Translator:       i18nSvc,
			ActionGatekeeper: actionGate,
		},
	)

	taxRateUC := taxRateUseCases.NewUseCases(
		taxRateUseCases.TaxRateRepositories{
			TaxRate: repos.TaxRate,
		},
		taxRateUseCases.TaxRateServices{
			Authorizer:       authSvc,
			Translator:       i18nSvc,
			ActionGatekeeper: actionGate,
		},
	)

	taxRegistrationUC := taxRegistrationUseCases.NewUseCases(
		taxRegistrationUseCases.TaxRegistrationRepositories{
			TaxRegistration:     repos.TaxRegistration,
			TaxRegistrationKind: repos.TaxRegistrationKind,
		},
		taxRegistrationUseCases.TaxRegistrationServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idService,
			ActionGatekeeper: actionGate,
		},
	)

	computeTaxesUC := computeTaxesUseCases.NewUseCases(
		computeTaxesUseCases.ComputeTaxesRepositories{
			Revenue:                repos.Revenue,
			RevenueLineItem:        repos.RevenueLineItem,
			RevenueTaxLine:         repos.RevenueTaxLine,
			Workspace:              repos.Workspace,
			Product:                repos.Product,
			TaxTreatment:           repos.TaxTreatment,
			TaxClass:               repos.TaxClass,
			TaxRate:                repos.TaxRate,
			TaxRegistration:        repos.TaxRegistration,
			TaxRegistrationKind:    repos.TaxRegistrationKind,
			TaxAuthority:           repos.TaxAuthority,
			WithholdingCertificate: repos.WithholdingCertificate,
		},
		computeTaxesUseCases.ComputeTaxesServices{
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idService,
			ActionGatekeeper: actionGate,
		},
	)

	return &TaxUseCases{
		TaxAuthority:        taxAuthorityUC,
		TaxRegistrationKind: taxRegistrationKindUC,
		TaxTreatment:        taxTreatmentUC,
		TaxClass:            taxClassUC,
		TaxRate:             taxRateUC,
		TaxRegistration:     taxRegistrationUC,
		ComputeTaxes:        computeTaxesUC,
	}
}
