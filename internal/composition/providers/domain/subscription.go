package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	// Protobuf domain services - Subscription domain
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
	balanceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance_attribute"
	invoicepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice"
	invoiceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice_attribute"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	planattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_attribute"
	plansettingspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_settings"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	subscriptionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_attribute"
)

// SubscriptionRepositories contains all subscription domain repositories
type SubscriptionRepositories struct {
	Balance               balancepb.BalanceDomainServiceServer
	BalanceAttribute      balanceattributepb.BalanceAttributeDomainServiceServer
	Client                clientpb.ClientDomainServiceServer // Cross-domain dependency
	Invoice               invoicepb.InvoiceDomainServiceServer
	InvoiceAttribute      invoiceattributepb.InvoiceAttributeDomainServiceServer
	Plan                  planpb.PlanDomainServiceServer
	PlanAttribute         planattributepb.PlanAttributeDomainServiceServer
	PlanSettings          plansettingspb.PlanSettingsDomainServiceServer
	PricePlan             priceplanpb.PricePlanDomainServiceServer
	PriceSchedule         priceschedulepb.PriceScheduleDomainServiceServer
	ProductPlan           productplanpb.ProductPlanDomainServiceServer // Cross-domain dependency (Model D)
	ProductPricePlan      productpriceplanpb.ProductPricePlanDomainServiceServer
	Subscription          subscriptionpb.SubscriptionDomainServiceServer
	SubscriptionAttribute subscriptionattributepb.SubscriptionAttributeDomainServiceServer
	Attribute             attributepb.AttributeDomainServiceServer
}

// NewSubscriptionRepositories creates and returns a new set of SubscriptionRepositories
func NewSubscriptionRepositories(dbProvider contracts.Provider, tableConfig *registry.TableConfig) (*SubscriptionRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()

	// Create each repository individually using configured table names from tableConfig
	balanceRepo, err := repoCreator.CreateRepository(entityid.Balance, conn, tableConfig.TableName(entityid.Balance))
	if err != nil {
		return nil, fmt.Errorf("failed to create balance repository: %w", err)
	}

	// Create cross-domain client repository (needed by CreateSubscription use case)
	clientRepo, err := repoCreator.CreateRepository(entityid.Client, conn, tableConfig.TableName(entityid.Client))
	if err != nil {
		return nil, fmt.Errorf("failed to create client repository: %w", err)
	}

	invoiceRepo, err := repoCreator.CreateRepository(entityid.Invoice, conn, tableConfig.TableName(entityid.Invoice))
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice repository: %w", err)
	}

	planRepo, err := repoCreator.CreateRepository(entityid.Plan, conn, tableConfig.TableName(entityid.Plan))
	if err != nil {
		return nil, fmt.Errorf("failed to create plan repository: %w", err)
	}

	planSettingsRepo, err := repoCreator.CreateRepository(entityid.PlanSettings, conn, tableConfig.TableName(entityid.PlanSettings))
	if err != nil {
		return nil, fmt.Errorf("failed to create plan_settings repository: %w", err)
	}

	pricePlanRepo, err := repoCreator.CreateRepository(entityid.PricePlan, conn, tableConfig.TableName(entityid.PricePlan))
	if err != nil {
		return nil, fmt.Errorf("failed to create price_plan repository: %w", err)
	}

	priceScheduleRepo, err := repoCreator.CreateRepository(entityid.PriceSchedule, conn, tableConfig.TableName(entityid.PriceSchedule))
	if err != nil {
		return nil, fmt.Errorf("failed to create price_schedule repository: %w", err)
	}

	productPricePlanRepo, err := repoCreator.CreateRepository(entityid.ProductPricePlan, conn, tableConfig.TableName(entityid.ProductPricePlan))
	if err != nil {
		return nil, fmt.Errorf("failed to create product_price_plan repository: %w", err)
	}

	// Cross-domain product_plan repository (Model D: used to validate
	// ProductPricePlan.product_plan_id FK and plan_id match in use cases)
	productPlanRepo, err := repoCreator.CreateRepository(entityid.ProductPlan, conn, tableConfig.TableName(entityid.ProductPlan))
	if err != nil {
		return nil, fmt.Errorf("failed to create product_plan repository: %w", err)
	}

	subscriptionRepo, err := repoCreator.CreateRepository(entityid.Subscription, conn, tableConfig.TableName(entityid.Subscription))
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription repository: %w", err)
	}

	balanceAttributeRepo, err := repoCreator.CreateRepository(entityid.BalanceAttribute, conn, tableConfig.TableName(entityid.BalanceAttribute))
	if err != nil {
		return nil, fmt.Errorf("failed to create balance_attribute repository: %w", err)
	}

	invoiceAttributeRepo, err := repoCreator.CreateRepository(entityid.InvoiceAttribute, conn, tableConfig.TableName(entityid.InvoiceAttribute))
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice_attribute repository: %w", err)
	}

	planAttributeRepo, err := repoCreator.CreateRepository(entityid.PlanAttribute, conn, tableConfig.TableName(entityid.PlanAttribute))
	if err != nil {
		return nil, fmt.Errorf("failed to create plan_attribute repository: %w", err)
	}

	subscriptionAttributeRepo, err := repoCreator.CreateRepository(entityid.SubscriptionAttribute, conn, tableConfig.TableName(entityid.SubscriptionAttribute))
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription_attribute repository: %w", err)
	}

	var attributeServer attributepb.AttributeDomainServiceServer
	attributeRepo, err := repoCreator.CreateRepository(entityid.Attribute, conn, tableConfig.TableName(entityid.Attribute))
	if err == nil {
		attributeServer = attributeRepo.(attributepb.AttributeDomainServiceServer)
	}

	// Type assert each repository to its interface
	return &SubscriptionRepositories{
		Balance:               balanceRepo.(balancepb.BalanceDomainServiceServer),
		BalanceAttribute:      balanceAttributeRepo.(balanceattributepb.BalanceAttributeDomainServiceServer),
		Client:                clientRepo.(clientpb.ClientDomainServiceServer),
		Invoice:               invoiceRepo.(invoicepb.InvoiceDomainServiceServer),
		InvoiceAttribute:      invoiceAttributeRepo.(invoiceattributepb.InvoiceAttributeDomainServiceServer),
		Plan:                  planRepo.(planpb.PlanDomainServiceServer),
		PlanAttribute:         planAttributeRepo.(planattributepb.PlanAttributeDomainServiceServer),
		PlanSettings:          planSettingsRepo.(plansettingspb.PlanSettingsDomainServiceServer),
		PricePlan:             pricePlanRepo.(priceplanpb.PricePlanDomainServiceServer),
		PriceSchedule:         priceScheduleRepo.(priceschedulepb.PriceScheduleDomainServiceServer),
		ProductPlan:           productPlanRepo.(productplanpb.ProductPlanDomainServiceServer),
		ProductPricePlan:      productPricePlanRepo.(productpriceplanpb.ProductPricePlanDomainServiceServer),
		Subscription:          subscriptionRepo.(subscriptionpb.SubscriptionDomainServiceServer),
		SubscriptionAttribute: subscriptionAttributeRepo.(subscriptionattributepb.SubscriptionAttributeDomainServiceServer),
		Attribute:             attributeServer,
	}, nil
}
