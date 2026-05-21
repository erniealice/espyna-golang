package compute_taxes_for_revenue

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"

	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenuelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
	revenuetaxlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_tax_line"
	taxauthoritypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_authority"
	taxclasspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_class"
	taxratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_rate"
	taxregistrationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration"
	taxregistrationkindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration_kind"
	taxtreatmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_treatment"
	withholdingcertificatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/withholding_certificate"
)

// ComputeTaxesRepositories groups all repository dependencies for compute.
type ComputeTaxesRepositories struct {
	Revenue                revenuepb.RevenueDomainServiceServer
	RevenueLineItem        revenuelineitempb.RevenueLineItemDomainServiceServer
	RevenueTaxLine         revenuetaxlinepb.RevenueTaxLineDomainServiceServer
	Workspace              workspacepb.WorkspaceDomainServiceServer
	Product                productpb.ProductDomainServiceServer
	TaxTreatment           taxtreatmentpb.TaxTreatmentDomainServiceServer
	TaxClass               taxclasspb.TaxClassDomainServiceServer
	TaxRate                taxratepb.TaxRateDomainServiceServer
	TaxRegistration        taxregistrationpb.TaxRegistrationDomainServiceServer
	TaxRegistrationKind    taxregistrationkindpb.TaxRegistrationKindDomainServiceServer
	TaxAuthority           taxauthoritypb.TaxAuthorityDomainServiceServer
	WithholdingCertificate withholdingcertificatepb.WithholdingCertificateDomainServiceServer
}

// ComputeTaxesServices groups all service dependencies.
type ComputeTaxesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains the compute taxes use case.
type UseCases struct {
	ComputeTaxesForRevenue *ComputeTaxesForRevenueUseCase
}

// NewUseCases creates the compute taxes use cases.
func NewUseCases(
	repos ComputeTaxesRepositories,
	services ComputeTaxesServices,
) *UseCases {
	return &UseCases{
		ComputeTaxesForRevenue: NewComputeTaxesForRevenueUseCase(repos, services),
	}
}
