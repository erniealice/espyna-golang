package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	// Protobuf domain services - Tenancy domain
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

// NewTenancyRepositories creates and returns a new set of TenancyRepositories.
// Individual repository failures are logged but do not prevent other repositories
// from being created (graceful degradation per-repository).
func NewTenancyRepositories(dbProvider contracts.Provider, tableConfig *registry.TableConfig) (*TenancyRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()
	repos := &TenancyRepositories{}
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

	if r := tryCreate(entityid.TenantSubscription); r != nil {
		repos.TenantSubscription = r.(tenantsubscriptionpb.TenantSubscriptionDomainServiceServer)
	}
	if r := tryCreate(entityid.TenantPaymentMethod); r != nil {
		repos.TenantPaymentMethod = r.(tenantpaymentmethodpb.TenantPaymentMethodDomainServiceServer)
	}
	if r := tryCreate(entityid.TenantInvoice); r != nil {
		repos.TenantInvoice = r.(tenantinvoicepb.TenantInvoiceDomainServiceServer)
	}

	if len(skipped) > 0 {
		fmt.Printf("Tenancy repos skipped (no adapter registered): %v\n", skipped)
	}

	return repos, nil
}
