package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	// Protobuf domain services - Entity domain (cross-domain dependency)
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"

	// Protobuf domain services - Revenue domain
	deferredrevenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/deferred_revenue"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenueattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_attribute"
	revenuecategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_category"
	revenuelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
)

// RevenueRepositories contains all revenue domain repositories
type RevenueRepositories struct {
	Revenue          revenuepb.RevenueDomainServiceServer
	RevenueLineItem  revenuelineitempb.RevenueLineItemDomainServiceServer
	RevenueCategory  revenuecategorypb.RevenueCategoryDomainServiceServer
	RevenueAttribute revenueattributepb.RevenueAttributeDomainServiceServer
	DeferredRevenue  deferredrevenuepb.DeferredRevenueDomainServiceServer
	// Cross-domain dependency: payment term lookup for due date computation
	PaymentTerm paymenttermpb.PaymentTermDomainServiceServer
}

// NewRevenueRepositories creates and returns a new set of RevenueRepositories.
// Individual repository failures are logged but do not prevent other repositories
// from being created (graceful degradation per-repository).
func NewRevenueRepositories(dbProvider contracts.Provider, tableConfig *registry.TableConfig) (*RevenueRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()
	repos := &RevenueRepositories{}
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

	if r := tryCreate(entityid.Revenue); r != nil {
		repos.Revenue = r.(revenuepb.RevenueDomainServiceServer)
	}
	if r := tryCreate(entityid.RevenueLineItem); r != nil {
		repos.RevenueLineItem = r.(revenuelineitempb.RevenueLineItemDomainServiceServer)
	}
	if r := tryCreate(entityid.RevenueCategory); r != nil {
		repos.RevenueCategory = r.(revenuecategorypb.RevenueCategoryDomainServiceServer)
	}
	if r := tryCreate(entityid.RevenueAttribute); r != nil {
		repos.RevenueAttribute = r.(revenueattributepb.RevenueAttributeDomainServiceServer)
	}
	if r := tryCreate(entityid.DeferredRevenue); r != nil {
		repos.DeferredRevenue = r.(deferredrevenuepb.DeferredRevenueDomainServiceServer)
	}
	if r := tryCreate(entityid.PaymentTerm); r != nil {
		repos.PaymentTerm = r.(paymenttermpb.PaymentTermDomainServiceServer)
	}

	if len(skipped) > 0 {
		fmt.Printf("⚠️  Revenue repos skipped (no adapter registered): %v\n", skipped)
	}

	return repos, nil
}
