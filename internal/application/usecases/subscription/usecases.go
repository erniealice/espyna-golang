package subscription

import (
	// Subscription use cases
	balanceUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/subscription/balance"
	balanceAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/subscription/balance_attribute"
	invoiceUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/subscription/invoice"
	invoiceAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/subscription/invoice_attribute"
	planUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/subscription/plan"
	planAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/subscription/plan_attribute"
	planSettingsUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/subscription/plan_settings"
	pricePlanUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/subscription/price_plan"
	subscriptionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/subscription/subscription"
	subscriptionAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/subscription/subscription_attribute"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Protobuf domain services for subscription repositories
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

// SubscriptionRepositories contains all subscription domain repositories
type SubscriptionRepositories struct {
	Balance               balancepb.BalanceDomainServiceServer
	BalanceAttribute      balanceattributepb.BalanceAttributeDomainServiceServer
	Client                clientpb.ClientDomainServiceServer
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

// SubscriptionUseCases contains all subscription-related use cases
type SubscriptionUseCases struct {
	Balance               *balanceUseCases.UseCases
	BalanceAttribute      *balanceAttributeUseCases.UseCases
	Invoice               *invoiceUseCases.UseCases
	InvoiceAttribute      *invoiceAttributeUseCases.UseCases
	Plan                  *planUseCases.UseCases
	PlanAttribute         *planAttributeUseCases.UseCases
	PlanSettings          *planSettingsUseCases.UseCases
	PricePlan             *pricePlanUseCases.UseCases
	Subscription          *subscriptionUseCases.UseCases
	SubscriptionAttribute *subscriptionAttributeUseCases.UseCases
}

// NewUseCases creates all subscription use cases with proper constructor injection
func NewUseCases(
	repos SubscriptionRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
) *SubscriptionUseCases {
	// Create use cases for each subscription entity
	balanceUC := balanceUseCases.NewUseCases(
		balanceUseCases.BalanceRepositories{Balance: repos.Balance},
		balanceUseCases.BalanceServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	invoiceUC := invoiceUseCases.NewUseCases(
		invoiceUseCases.InvoiceRepositories{Invoice: repos.Invoice},
		invoiceUseCases.InvoiceServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	planUC := planUseCases.NewUseCases(
		planUseCases.PlanRepositories{Plan: repos.Plan},
		planUseCases.PlanServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	planSettingsUC := planSettingsUseCases.NewUseCases(
		planSettingsUseCases.PlanSettingsRepositories{PlanSettings: repos.PlanSettings},
		planSettingsUseCases.PlanSettingsServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	pricePlanUC := pricePlanUseCases.NewUseCases(
		pricePlanUseCases.PricePlanRepositories{PricePlan: repos.PricePlan, Plan: repos.Plan},
		pricePlanUseCases.PricePlanServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	subscriptionUC := subscriptionUseCases.NewUseCases(
		subscriptionUseCases.SubscriptionRepositories{
			Subscription: repos.Subscription,
			Client:       repos.Client,
			PricePlan:    repos.PricePlan,
		},
		subscriptionUseCases.SubscriptionServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	balanceAttributeUC := balanceAttributeUseCases.NewUseCases(
		balanceAttributeUseCases.BalanceAttributeRepositories{
			BalanceAttribute: repos.BalanceAttribute,
			Balance:          repos.Balance,
			Attribute:        repos.Attribute,
		},
		balanceAttributeUseCases.BalanceAttributeServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	invoiceAttributeUC := invoiceAttributeUseCases.NewUseCases(
		invoiceAttributeUseCases.InvoiceAttributeRepositories{
			InvoiceAttribute: repos.InvoiceAttribute,
			Invoice:          repos.Invoice,
			Attribute:        repos.Attribute,
		},
		invoiceAttributeUseCases.InvoiceAttributeServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	planAttributeUC := planAttributeUseCases.NewUseCases(
		planAttributeUseCases.PlanAttributeRepositories{
			PlanAttribute: repos.PlanAttribute,
			Plan:          repos.Plan,
			Attribute:     repos.Attribute,
		},
		planAttributeUseCases.PlanAttributeServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	subscriptionAttributeUC := subscriptionAttributeUseCases.NewUseCases(
		subscriptionAttributeUseCases.SubscriptionAttributeRepositories{
			SubscriptionAttribute: repos.SubscriptionAttribute,
			Subscription:          repos.Subscription,
			Attribute:             repos.Attribute,
		},
		subscriptionAttributeUseCases.SubscriptionAttributeServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	return &SubscriptionUseCases{
		Balance:               balanceUC,
		BalanceAttribute:      balanceAttributeUC,
		Invoice:               invoiceUC,
		InvoiceAttribute:      invoiceAttributeUC,
		Plan:                  planUC,
		PlanAttribute:         planAttributeUC,
		PlanSettings:          planSettingsUC,
		PricePlan:             pricePlanUC,
		Subscription:          subscriptionUC,
		SubscriptionAttribute: subscriptionAttributeUC,
	}
}
