package subscription

import (
	// Subscription use cases
	balanceUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/balance"
	balanceAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/balance_attribute"
	billingEventUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/billing_event"
	invoiceUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/invoice"
	invoiceAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/invoice_attribute"
	planUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/plan"
	planAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/plan_attribute"
	planSettingsUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/plan_settings"
	pricePlanUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/price_plan"
	priceScheduleUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/price_schedule"
	productPricePlanUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/product_price_plan"
	subscriptionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/subscription"
	subscriptionAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/subscription_attribute"
	subscriptionSeatUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/subscription_seat"
	subscriptionWorkspaceUserUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/subscription_workspace_user"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Protobuf domain services for subscription repositories
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	clientworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_workspace_user"
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
	subscriptionseatpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_seat"
	subscriptionworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_workspace_user"
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
	// Outsourcing-vertical seat + servicing membership
	SubscriptionSeat          subscriptionseatpb.SubscriptionSeatDomainServiceServer
	SubscriptionWorkspaceUser subscriptionworkspaceuserpb.SubscriptionWorkspaceUserDomainServiceServer
	ClientWorkspaceUser       clientworkspaceuserpb.ClientWorkspaceUserDomainServiceServer // cross-domain composite-FK pre-check
	Attribute                 attributepb.AttributeDomainServiceServer
}

// SubscriptionUseCases contains all subscription-related use cases.
//
// 20260518-hexagonal-strict-adherence Phase 3 — F6 + F7 closure:
//   - BillingEvent (raw DomainServiceServer leak) wrapped in a Layer-7
//     use case sub-aggregate; nested under .BillingEvent.
//   - MaterializeJobsForSubscription + MaterializeInstanceJobsForSubscription
//     (flat fields) nested under .Subscription.MaterializeJobs and
//     .Subscription.MaterializeInstanceJobs respectively.
type SubscriptionUseCases struct {
	Balance               *balanceUseCases.UseCases
	BalanceAttribute      *balanceAttributeUseCases.UseCases
	BillingEvent          *billingEventUseCases.UseCases
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
	// Outsourcing-vertical seat + servicing membership
	SubscriptionSeat          *subscriptionSeatUseCases.UseCases
	SubscriptionWorkspaceUser *subscriptionWorkspaceUserUseCases.UseCases
}

// NewUseCases creates all subscription use cases with proper constructor injection.
//
// `refChecker` is the application-port ReferenceChecker used by the
// client-scope guards on UpdatePlan (§3.1) and UpdatePricePlan (§3.5). Pass
// `ports.NewNoOpReferenceChecker()` from non-postgres providers and tests
// that don't need to gate on cross-row state.
func NewUseCases(
	repos SubscriptionRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idService ports.IDGenerator,
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
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	invoiceUC := invoiceUseCases.NewUseCases(
		invoiceUseCases.InvoiceRepositories{Invoice: repos.Invoice},
		invoiceUseCases.InvoiceServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
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
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idService,
			ReferenceChecker: refChecker,
		},
	)

	planSettingsUC := planSettingsUseCases.NewUseCases(
		planSettingsUseCases.PlanSettingsRepositories{PlanSettings: repos.PlanSettings},
		planSettingsUseCases.PlanSettingsServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
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
			Authorizer:       authSvc,
			Transactor:       txSvc,
			Translator:       i18nSvc,
			IDGenerator:      idService,
			ReferenceChecker: refChecker,
		},
	)

	priceScheduleUC := priceScheduleUseCases.NewUseCases(
		priceScheduleUseCases.PriceScheduleRepositories{PriceSchedule: repos.PriceSchedule},
		priceScheduleUseCases.PriceScheduleServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	productPricePlanUC := productPricePlanUseCases.NewUseCases(
		productPricePlanUseCases.ProductPricePlanRepositories{
			ProductPricePlan: repos.ProductPricePlan,
			PricePlan:        repos.PricePlan,
			ProductPlan:      repos.ProductPlan,
		},
		productPricePlanUseCases.ProductPricePlanServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	subscriptionUC := subscriptionUseCases.NewUseCases(
		subscriptionUseCases.SubscriptionRepositories{
			Subscription: repos.Subscription,
			Client:       repos.Client,
			PricePlan:    repos.PricePlan,
		},
		subscriptionUseCases.SubscriptionServices{
			Authorizer:              authSvc,
			Transactor:              txSvc,
			Translator:              i18nSvc,
			IDGenerator:             idService,
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
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	invoiceAttributeUC := invoiceAttributeUseCases.NewUseCases(
		invoiceAttributeUseCases.InvoiceAttributeRepositories{
			InvoiceAttribute: repos.InvoiceAttribute,
			Invoice:          repos.Invoice,
			Attribute:        repos.Attribute,
		},
		invoiceAttributeUseCases.InvoiceAttributeServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	planAttributeUC := planAttributeUseCases.NewUseCases(
		planAttributeUseCases.PlanAttributeRepositories{
			PlanAttribute: repos.PlanAttribute,
			Plan:          repos.Plan,
			Attribute:     repos.Attribute,
		},
		planAttributeUseCases.PlanAttributeServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	subscriptionAttributeUC := subscriptionAttributeUseCases.NewUseCases(
		subscriptionAttributeUseCases.SubscriptionAttributeRepositories{
			SubscriptionAttribute: repos.SubscriptionAttribute,
			Subscription:          repos.Subscription,
			Attribute:             repos.Attribute,
		},
		subscriptionAttributeUseCases.SubscriptionAttributeServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	// Outsourcing-vertical seat use cases (CRUD + SR-2 replace + set-status).
	subscriptionSeatUC := subscriptionSeatUseCases.NewUseCases(
		subscriptionSeatUseCases.SubscriptionSeatRepositories{
			SubscriptionSeat: repos.SubscriptionSeat,
			Subscription:     repos.Subscription,
		},
		subscriptionSeatUseCases.SubscriptionSeatServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	// Outsourcing-vertical servicing-membership use cases (composite-FK pre-check
	// against client_workspace_user; client_id stamped from the subscription).
	subscriptionWorkspaceUserUC := subscriptionWorkspaceUserUseCases.NewUseCases(
		subscriptionWorkspaceUserUseCases.SubscriptionWorkspaceUserRepositories{
			SubscriptionWorkspaceUser: repos.SubscriptionWorkspaceUser,
			Subscription:              repos.Subscription,
			ClientWorkspaceUser:       repos.ClientWorkspaceUser,
		},
		subscriptionWorkspaceUserUseCases.SubscriptionWorkspaceUserServices{
			Authorizer:  authSvc,
			Transactor:  txSvc,
			Translator:  i18nSvc,
			IDGenerator: idService,
		},
	)

	// Phase 3 F7 closure — wrap the raw BillingEvent DomainServiceServer in
	// a Layer-7 use case sub-aggregate. Constructor is nil-safe when the
	// adapter isn't registered.
	billingEventUC := billingEventUseCases.NewUseCases(
		billingEventUseCases.BillingEventRepositories{BillingEvent: repos.BillingEvent},
		billingEventUseCases.BillingEventServices{
			Authorizer: authSvc,
			Transactor: txSvc,
			Translator: i18nSvc,
		},
	)

	return &SubscriptionUseCases{
		Balance:                   balanceUC,
		BalanceAttribute:          balanceAttributeUC,
		BillingEvent:              billingEventUC,
		Invoice:                   invoiceUC,
		InvoiceAttribute:          invoiceAttributeUC,
		Plan:                      planUC,
		PlanAttribute:             planAttributeUC,
		PlanSettings:              planSettingsUC,
		PricePlan:                 pricePlanUC,
		PriceSchedule:             priceScheduleUC,
		ProductPricePlan:          productPricePlanUC,
		Subscription:              subscriptionUC,
		SubscriptionAttribute:     subscriptionAttributeUC,
		SubscriptionSeat:          subscriptionSeatUC,
		SubscriptionWorkspaceUser: subscriptionWorkspaceUserUC,
	}
}
