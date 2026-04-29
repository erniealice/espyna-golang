package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	// Protobuf domain services - Entity domain (cross-domain dependency)
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"

	// Protobuf domain services - Revenue domain
	deferredrevenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/deferred_revenue"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenueattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_attribute"
	revenuecategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_category"
	revenuelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"

	// Protobuf domain services - Subscription domain (cross-domain dependency
	// for the recognize-revenue use case)
	billingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billing_event"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"

	// Protobuf domain services - Operation domain (cross-domain dependency for
	// the milestone-billing branch of recognize-revenue and for
	// MaterializeBillingEventsForJob).
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
)

// RevenueRepositories contains all revenue domain repositories.
//
// In addition to revenue-domain repos, this struct carries cross-domain
// dependencies needed by the RecognizeRevenueFromSubscription use case:
// Subscription, PricePlan, ProductPricePlan, PriceSchedule, and Client.
type RevenueRepositories struct {
	Revenue          revenuepb.RevenueDomainServiceServer
	RevenueLineItem  revenuelineitempb.RevenueLineItemDomainServiceServer
	RevenueCategory  revenuecategorypb.RevenueCategoryDomainServiceServer
	RevenueAttribute revenueattributepb.RevenueAttributeDomainServiceServer
	DeferredRevenue  deferredrevenuepb.DeferredRevenueDomainServiceServer
	// Cross-domain dependency: payment term lookup for due date computation
	PaymentTerm paymenttermpb.PaymentTermDomainServiceServer

	// Cross-domain dependencies for the RecognizeRevenueFromSubscription use
	// case (plan §5 Phase B). All optional — the use case returns an
	// appropriate validation error when called with a nil repo.
	Subscription     subscriptionpb.SubscriptionDomainServiceServer
	PricePlan        priceplanpb.PricePlanDomainServiceServer
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer
	PriceSchedule    priceschedulepb.PriceScheduleDomainServiceServer
	Client           clientpb.ClientDomainServiceServer

	// Cross-domain dependencies for the milestone-billing engine branch
	// (Phase C — milestone-billing plan §3) and the
	// MaterializeBillingEventsForJob use case. All optional — the use case
	// rejects MILESTONE branch with `billing_event_repository_unavailable`
	// when nil.
	BillingEvent     billingeventpb.BillingEventDomainServiceServer
	JobTemplatePhase jobtemplatephasepb.JobTemplatePhaseDomainServiceServer
	Job              jobpb.JobDomainServiceServer
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

	// Cross-domain reads required by RecognizeRevenueFromSubscription.
	if r := tryCreate(entityid.Subscription); r != nil {
		repos.Subscription = r.(subscriptionpb.SubscriptionDomainServiceServer)
	}
	if r := tryCreate(entityid.PricePlan); r != nil {
		repos.PricePlan = r.(priceplanpb.PricePlanDomainServiceServer)
	}
	if r := tryCreate(entityid.ProductPricePlan); r != nil {
		repos.ProductPricePlan = r.(productpriceplanpb.ProductPricePlanDomainServiceServer)
	}
	if r := tryCreate(entityid.PriceSchedule); r != nil {
		repos.PriceSchedule = r.(priceschedulepb.PriceScheduleDomainServiceServer)
	}
	if r := tryCreate(entityid.Client); r != nil {
		repos.Client = r.(clientpb.ClientDomainServiceServer)
	}

	// Milestone-billing cross-domain reads (Phase C).
	if r := tryCreate(entityid.BillingEvent); r != nil {
		repos.BillingEvent = r.(billingeventpb.BillingEventDomainServiceServer)
	}
	if r := tryCreate(entityid.JobTemplatePhase); r != nil {
		repos.JobTemplatePhase = r.(jobtemplatephasepb.JobTemplatePhaseDomainServiceServer)
	}
	if r := tryCreate(entityid.Job); r != nil {
		repos.Job = r.(jobpb.JobDomainServiceServer)
	}

	if len(skipped) > 0 {
		fmt.Printf("⚠️  Revenue repos skipped (no adapter registered): %v\n", skipped)
	}

	return repos, nil
}
