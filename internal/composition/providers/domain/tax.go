package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	// Protobuf domain services - entity (cross-domain for ComputeTaxesForRevenue)
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"

	// Protobuf domain services - product (cross-domain for ComputeTaxesForRevenue)
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"

	// Protobuf domain services - revenue (cross-domain for ComputeTaxesForRevenue)
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenuelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
	revenuetaxlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_tax_line"

	// Protobuf domain services - Tax domain
	taxauthoritypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_authority"
	taxclasspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_class"
	taxratepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_rate"
	taxregistrationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration"
	taxregistrationkindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration_kind"
	taxtreatmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_treatment"

	// Protobuf domain services - treasury (cross-domain for ComputeTaxesForRevenue)
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

	// Cross-domain repos for ComputeTaxesForRevenue.
	Revenue                revenuepb.RevenueDomainServiceServer
	RevenueLineItem        revenuelineitempb.RevenueLineItemDomainServiceServer
	RevenueTaxLine         revenuetaxlinepb.RevenueTaxLineDomainServiceServer
	Workspace              workspacepb.WorkspaceDomainServiceServer
	Product                productpb.ProductDomainServiceServer
	WithholdingCertificate withholdingcertificatepb.WithholdingCertificateDomainServiceServer
}

// NewTaxRepositories creates and returns a new set of TaxRepositories.
// Individual repository failures are logged but do not prevent other repositories
// from being created (graceful degradation per-repository).
func NewTaxRepositories(dbProvider contracts.Provider, tableConfig *registry.TableConfig) (*TaxRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()
	repos := &TaxRepositories{}
	var skipped []string

	// Helper: try to create a repository, log and skip on failure
	tryCreate := func(entity string) interface{} {
		repo, err := repoCreator.CreateRepository(entity, conn, tableConfig.TableName(entity))
		if err != nil {
			skipped = append(skipped, entity)
			return nil
		}
		return repo
	}

	if r := tryCreate(entityid.TaxAuthority); r != nil {
		repos.TaxAuthority = r.(taxauthoritypb.TaxAuthorityDomainServiceServer)
	}
	if r := tryCreate(entityid.TaxRegistrationKind); r != nil {
		repos.TaxRegistrationKind = r.(taxregistrationkindpb.TaxRegistrationKindDomainServiceServer)
	}
	if r := tryCreate(entityid.TaxTreatment); r != nil {
		repos.TaxTreatment = r.(taxtreatmentpb.TaxTreatmentDomainServiceServer)
	}
	if r := tryCreate(entityid.TaxClass); r != nil {
		repos.TaxClass = r.(taxclasspb.TaxClassDomainServiceServer)
	}
	if r := tryCreate(entityid.TaxRate); r != nil {
		repos.TaxRate = r.(taxratepb.TaxRateDomainServiceServer)
	}
	if r := tryCreate(entityid.TaxRegistration); r != nil {
		repos.TaxRegistration = r.(taxregistrationpb.TaxRegistrationDomainServiceServer)
	}

	// Cross-domain repos for ComputeTaxesForRevenue.
	if r := tryCreate(entityid.Revenue); r != nil {
		repos.Revenue = r.(revenuepb.RevenueDomainServiceServer)
	}
	if r := tryCreate(entityid.RevenueLineItem); r != nil {
		repos.RevenueLineItem = r.(revenuelineitempb.RevenueLineItemDomainServiceServer)
	}
	if r := tryCreate(entityid.RevenueTaxLine); r != nil {
		repos.RevenueTaxLine = r.(revenuetaxlinepb.RevenueTaxLineDomainServiceServer)
	}
	if r := tryCreate(entityid.Workspace); r != nil {
		repos.Workspace = r.(workspacepb.WorkspaceDomainServiceServer)
	}
	if r := tryCreate(entityid.Product); r != nil {
		repos.Product = r.(productpb.ProductDomainServiceServer)
	}
	if r := tryCreate(entityid.WithholdingCertificate); r != nil {
		repos.WithholdingCertificate = r.(withholdingcertificatepb.WithholdingCertificateDomainServiceServer)
	}

	if len(skipped) > 0 {
		fmt.Printf("Tax repos skipped (no adapter registered): %v\n", skipped)
	}

	return repos, nil
}
