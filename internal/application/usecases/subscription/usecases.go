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
	priceScheduleUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/subscription/price_schedule"
	productPricePlanUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/subscription/product_price_plan"
	subscriptionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/subscription/subscription"
	subscriptionAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/subscription/subscription_attribute"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Protobuf domain services for subscription repositories
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
	balanceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance_attribute"
	billingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billing_event"
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
	BillingEvent          billingeventpb.BillingEventDomainServiceServer
	Client                clientpb.ClientDomainServiceServer
	Invoice               invoicepb.InvoiceDomainServiceServer
	InvoiceAttribute      invoiceattributepb.InvoiceAttributeDomainServiceServer
	Plan                  planpb.PlanDomainServiceServer
	PlanAttribute         planattributepb.PlanAttributeDomainServiceServer
	PlanSettings          plansettingspb.PlanSettingsDomainServiceServer
	PricePlan             priceplanpb.PricePlanDomainServiceServer
	PriceSchedule         priceschedulepb.PriceScheduleDomainServiceServer
	ProductPlan           productplanpb.ProductPlanDomainServiceServer // Cross-domain (Model D: product_price_plan.product_plan_id FK validation)
	ProductPricePlan      productpriceplanpb.ProductPricePlanDomainServiceServer
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
	PriceSchedule         *priceScheduleUseCases.UseCases
	ProductPricePlan      *productPricePlanUseCases.UseCases
	Subscription          *subscriptionUseCases.UseCases
	SubscriptionAttribute *subscriptionAttributeUseCases.UseCases

	// BillingEvent exposes the BillingEvent domain server directly (no use-case
	// wrapper yet). centymo views invoke ListBySubscription / SetStatus through
	// this for the milestone-billing Package tab + mark-ready/waive handlers.
	// nil-safe: when the adapter isn't registered, callers degrade to empty
	// milestone lists and disable the mark-ready button.
	BillingEvent billingeventpb.BillingEventDomainServiceServer

	// MaterializeJobsForSubscription exposes the auto-spawn-jobs-from-
	// subscription use case (plan §3) directly so centymo's retroactive
	// spawn handler + create-form opt-out can invoke it. nil-safe.
	// 2026-04-29 auto-spawn-jobs-from-subscription Phase D.
	MaterializeJobsForSubscription *subscriptionUseCases.MaterializeJobsForSubscriptionUseCase
}

// NewUseCases creates all subscription use cases with proper constructor injection.
//
// `refChecker` is the application-port ReferenceChecker used by the
// client-scope guards on UpdatePlan (§3.1) and UpdatePricePlan (§3.5). Pass
// `ports.NewNoOpReferenceChecker()` from non-postgres providers and tests
// that don't need to gate on cross-row state.
func NewUseCases(
	repos SubscriptionRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
	jobTemplateInstantiator subscriptionUseCases.JobTemplateInstantiator,
	refChecker ports.ReferenceChecker,
) *SubscriptionUseCases {
	if refChecker == nil {
		refChecker = ports.NewNoOpReferenceChecker()
	}
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
		planUseCases.PlanRepositories{
			Plan:             repos.Plan,
			PricePlan:        repos.PricePlan,
			ProductPlan:      repos.ProductPlan,
			ProductPricePlan: repos.ProductPricePlan,
			PriceSchedule:    repos.PriceSchedule,
			Subscription:     repos.Subscription,
			Client:           repos.Client,
		},
		planUseCases.PlanServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
			ReferenceChecker:     refChecker,
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
		pricePlanUseCases.PricePlanRepositories{
			PricePlan:     repos.PricePlan,
			Plan:          repos.Plan,
			PriceSchedule: repos.PriceSchedule,
			Client:        repos.Client,
		},
		pricePlanUseCases.PricePlanServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
			ReferenceChecker:     refChecker,
		},
	)

	priceScheduleUC := priceScheduleUseCases.NewUseCases(
		priceScheduleUseCases.PriceScheduleRepositories{PriceSchedule: repos.PriceSchedule},
		priceScheduleUseCases.PriceScheduleServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	productPricePlanUC := productPricePlanUseCases.NewUseCases(
		productPricePlanUseCases.ProductPricePlanRepositories{
			ProductPricePlan: repos.ProductPricePlan,
			PricePlan:        repos.PricePlan,
			ProductPlan:      repos.ProductPlan,
		},
		productPricePlanUseCases.ProductPricePlanServices{
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
			AuthorizationService:    authSvc,
			TransactionService:      txSvc,
			TranslationService:      i18nSvc,
			IDService:               idService,
			JobTemplateInstantiator: jobTemplateInstantiator,
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
		PriceSchedule:         priceScheduleUC,
		ProductPricePlan:      productPricePlanUC,
		Subscription:          subscriptionUC,
		SubscriptionAttribute: subscriptionAttributeUC,
		BillingEvent:          repos.BillingEvent,
	}
}
