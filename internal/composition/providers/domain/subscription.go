package domain

import (
	"fmt"

	"leapfor.xyz/espyna/internal/composition/contracts"
	"leapfor.xyz/espyna/internal/infrastructure/registry"

	// Protobuf domain services - Subscription domain
	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"
	clientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/client"
	balancepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/balance"
	balanceattributepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/balance_attribute"
	invoicepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/invoice"
	invoiceattributepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/invoice_attribute"
	planpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan"
	planattributepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan_attribute"
	plansettingspb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan_settings"
	priceplanpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/price_plan"
	subscriptionpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/subscription"
	subscriptionattributepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/subscription_attribute"
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
	balanceRepo, err := repoCreator.CreateRepository("balance", conn, dbTableConfig.Balance)
	if err != nil {
		return nil, fmt.Errorf("failed to create balance repository: %w", err)
	}

	// Create cross-domain client repository (needed by CreateSubscription use case)
	clientRepo, err := repoCreator.CreateRepository("client", conn, dbTableConfig.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to create client repository: %w", err)
	}

	invoiceRepo, err := repoCreator.CreateRepository("invoice", conn, dbTableConfig.Invoice)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice repository: %w", err)
	}

	planRepo, err := repoCreator.CreateRepository("plan", conn, dbTableConfig.Plan)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan repository: %w", err)
	}

	planSettingsRepo, err := repoCreator.CreateRepository("plan_settings", conn, dbTableConfig.PlanSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan_settings repository: %w", err)
	}

	pricePlanRepo, err := repoCreator.CreateRepository("price_plan", conn, dbTableConfig.PricePlan)
	if err != nil {
		return nil, fmt.Errorf("failed to create price_plan repository: %w", err)
	}

	subscriptionRepo, err := repoCreator.CreateRepository("subscription", conn, dbTableConfig.Subscription)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription repository: %w", err)
	}

	balanceAttributeRepo, err := repoCreator.CreateRepository("balance_attribute", conn, dbTableConfig.BalanceAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create balance_attribute repository: %w", err)
	}

	invoiceAttributeRepo, err := repoCreator.CreateRepository("invoice_attribute", conn, dbTableConfig.InvoiceAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice_attribute repository: %w", err)
	}

	planAttributeRepo, err := repoCreator.CreateRepository("plan_attribute", conn, dbTableConfig.PlanAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan_attribute repository: %w", err)
	}

	subscriptionAttributeRepo, err := repoCreator.CreateRepository("subscription_attribute", conn, dbTableConfig.SubscriptionAttribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription_attribute repository: %w", err)
	}

	attributeRepo, err := repoCreator.CreateRepository("attribute", conn, dbTableConfig.Attribute)
	if err != nil {
		return nil, fmt.Errorf("failed to create attribute repository: %w", err)
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
		Attribute:             attributeRepo.(attributepb.AttributeDomainServiceServer),
	}, nil
}
