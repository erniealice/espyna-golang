package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	// Protobuf domain services - Subscription domain
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
	balanceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance_attribute"
	invoicepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice"
	invoiceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice_attribute"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	planattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_attribute"
	plansettingspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_settings"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	subscriptionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_attribute"
)

// SubscriptionRepositories contains all 11 subscription domain repositories (6 entities + 4 attributes + attribute + cross-domain dependencies)
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
	Subscription          subscriptionpb.SubscriptionDomainServiceServer
	SubscriptionAttribute subscriptionattributepb.SubscriptionAttributeDomainServiceServer
	Attribute             attributepb.AttributeDomainServiceServer
}

// NewSubscriptionRepositories creates and returns a new set of SubscriptionRepositories
func NewSubscriptionRepositories(dbProvider contracts.Provider, dbTableConfig *registry.DatabaseTableConfig) (*SubscriptionRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()

	// Create each repository individually using configured table names directly from dbTableConfig
	balanceRepo, err := repoCreator.CreateRepository(entityid.Balance, conn, dbTableConfig.Balance)
	if err != nil {
		return nil, fmt.Errorf("failed to create balance repository: %w", err)
	}

	// Create cross-domain client repository (needed by CreateSubscription use case)
	clientRepo, err := repoCreator.CreateRepository(entityid.Client, conn, dbTableConfig.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to create client repository: %w", err)
	}

	invoiceRepo, err := repoCreator.CreateRepository(entityid.Invoice, conn, dbTableConfig.Invoice)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice repository: %w", err)
	}

	planRepo, err := repoCreator.CreateRepository(entityid.Plan, conn, dbTableConfig.Plan)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan repository: %w", err)
	}

	planSettingsRepo, err := repoCreator.CreateRepository(entityid.PlanSettings, conn, dbTableConfig.PlanSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan_settings repository: %w", err)
	}

	pricePlanRepo, err := repoCreator.CreateRepository(entityid.PricePlan, conn, dbTableConfig.PricePlan)
	if err != nil {
		return nil, fmt.Errorf("failed to create price_plan repository: %w", err)
	}

	subscriptionRepo, err := repoCreator.CreateRepository(entityid.Subscription, conn, dbTableConfig.Subscription)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription repository: %w", err)
	}

	balanceAttributeRepo, err := repoCreator.CreateRepository(entityid.BalanceAttribute, conn, dbTableConfig.BalanceAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create balance_attribute repository: %w", err)
	}

	invoiceAttributeRepo, err := repoCreator.CreateRepository(entityid.InvoiceAttribute, conn, dbTableConfig.InvoiceAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice_attribute repository: %w", err)
	}

	planAttributeRepo, err := repoCreator.CreateRepository(entityid.PlanAttribute, conn, dbTableConfig.PlanAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan_attribute repository: %w", err)
	}

	subscriptionAttributeRepo, err := repoCreator.CreateRepository(entityid.SubscriptionAttribute, conn, dbTableConfig.SubscriptionAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription_attribute repository: %w", err)
	}

	var attributeServer attributepb.AttributeDomainServiceServer
	attributeRepo, err := repoCreator.CreateRepository(entityid.Attribute, conn, dbTableConfig.Attribute)
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
		Subscription:          subscriptionRepo.(subscriptionpb.SubscriptionDomainServiceServer),
		SubscriptionAttribute: subscriptionAttributeRepo.(subscriptionattributepb.SubscriptionAttributeDomainServiceServer),
		Attribute:             attributeServer,
	}, nil
}
