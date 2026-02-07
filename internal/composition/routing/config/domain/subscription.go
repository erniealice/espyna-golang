package domain

import (
	"fmt"

	subscriptionuc "leapfor.xyz/espyna/internal/application/usecases/subscription"
	"leapfor.xyz/espyna/internal/composition/contracts"
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

// ConfigureSubscriptionDomain configures routes for the Subscription domain with use cases injected directly
func ConfigureSubscriptionDomain(subscriptionUseCases *subscriptionuc.SubscriptionUseCases) contracts.DomainRouteConfiguration {
	// Handle nil use cases gracefully for backward compatibility
	if subscriptionUseCases == nil {
		fmt.Printf("⚠️  Subscription use cases is NIL\n")
		return contracts.DomainRouteConfiguration{
			Domain:  "subscription",
			Prefix:  "/subscription",
			Enabled: false,                            // Disable until use cases are properly initialized
			Routes:  []contracts.RouteConfiguration{}, // No routes without use cases
		}
	}

	fmt.Printf("✅ Subscription use cases are properly initialized!\n")

	// Initialize routes array
	routes := []contracts.RouteConfiguration{}

	// Balance module routes
	if subscriptionUseCases.Balance != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/balance/create",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Balance.CreateBalance, &balancepb.CreateBalanceRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/balance/read",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Balance.ReadBalance, &balancepb.ReadBalanceRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/balance/update",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Balance.UpdateBalance, &balancepb.UpdateBalanceRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/balance/delete",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Balance.DeleteBalance, &balancepb.DeleteBalanceRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/balance/list",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Balance.ListBalances, &balancepb.ListBalancesRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/balance/get-list-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Balance.GetBalanceListPageData, &balancepb.GetBalanceListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/balance/get-item-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Balance.GetBalanceItemPageData, &balancepb.GetBalanceItemPageDataRequest{}),
		})
	}

	// Invoice module routes
	if subscriptionUseCases.Invoice != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/invoice/create",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Invoice.CreateInvoice, &invoicepb.CreateInvoiceRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/invoice/read",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Invoice.ReadInvoice, &invoicepb.ReadInvoiceRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/invoice/update",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Invoice.UpdateInvoice, &invoicepb.UpdateInvoiceRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/invoice/delete",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Invoice.DeleteInvoice, &invoicepb.DeleteInvoiceRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/invoice/list",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Invoice.ListInvoices, &invoicepb.ListInvoicesRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/invoice/get-list-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Invoice.GetInvoiceListPageData, &invoicepb.GetInvoiceListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/invoice/get-item-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Invoice.GetInvoiceItemPageData, &invoicepb.GetInvoiceItemPageDataRequest{}),
		})
	}

	// Plan module routes
	if subscriptionUseCases.Plan != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/plan/create",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Plan.CreatePlan, &planpb.CreatePlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/plan/read",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Plan.ReadPlan, &planpb.ReadPlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/plan/update",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Plan.UpdatePlan, &planpb.UpdatePlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/plan/delete",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Plan.DeletePlan, &planpb.DeletePlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/plan/list",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Plan.ListPlans, &planpb.ListPlansRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/plan/get-list-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Plan.GetPlanListPageData, &planpb.GetPlanListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/plan/get-item-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Plan.GetPlanItemPageData, &planpb.GetPlanItemPageDataRequest{}),
		})
	}

	// Plan Settings module routes
	if subscriptionUseCases.PlanSettings != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/plan_settings/create",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PlanSettings.CreatePlanSettings, &plansettingspb.CreatePlanSettingsRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/plan_settings/read",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PlanSettings.ReadPlanSettings, &plansettingspb.ReadPlanSettingsRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/plan_settings/update",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PlanSettings.UpdatePlanSettings, &plansettingspb.UpdatePlanSettingsRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/plan_settings/delete",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PlanSettings.DeletePlanSettings, &plansettingspb.DeletePlanSettingsRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/plan_settings/list",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PlanSettings.ListPlanSettings, &plansettingspb.ListPlanSettingsRequest{}),
		})

		// routes = append(routes, contracts.RouteConfiguration{
		// 	Method:  "POST",
		// 	Path:    "/api/subscription/plan_settings/get-list-page-data",
		// 	Handler: contracts.NewGenericHandler(subscriptionUseCases.PlanSettings.GetListPageData, &plansettingspb.GetListPageDataRequest{}),
		// })

		// routes = append(routes, contracts.RouteConfiguration{
		// 	Method:  "POST",
		// 	Path:    "/api/subscription/plan_settings/get-item-page-data",
		// 	Handler: contracts.NewGenericHandler(subscriptionUseCases.PlanSettings.GetItemPageData, &plansettingspb.GetItemPageDataRequest{}),
		// })
	}

	// Price Plan module routes
	if subscriptionUseCases.PricePlan != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/price-plan/create",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PricePlan.CreatePricePlan, &priceplanpb.CreatePricePlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/price-plan/read",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PricePlan.ReadPricePlan, &priceplanpb.ReadPricePlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/price-plan/update",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PricePlan.UpdatePricePlan, &priceplanpb.UpdatePricePlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/price-plan/delete",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PricePlan.DeletePricePlan, &priceplanpb.DeletePricePlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/price-plan/list",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PricePlan.ListPricePlans, &priceplanpb.ListPricePlansRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/price-plan/get-list-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PricePlan.GetPricePlanListPageData, &priceplanpb.GetPricePlanListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/price-plan/get-item-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PricePlan.GetPricePlanItemPageData, &priceplanpb.GetPricePlanItemPageDataRequest{}),
		})
	}

	// Subscription module routes
	if subscriptionUseCases.Subscription != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription/create",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Subscription.CreateSubscription, &subscriptionpb.CreateSubscriptionRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription/read",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Subscription.ReadSubscription, &subscriptionpb.ReadSubscriptionRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription/update",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Subscription.UpdateSubscription, &subscriptionpb.UpdateSubscriptionRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription/delete",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Subscription.DeleteSubscription, &subscriptionpb.DeleteSubscriptionRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription/list",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Subscription.ListSubscriptions, &subscriptionpb.ListSubscriptionsRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription/get-list-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Subscription.GetSubscriptionListPageData, &subscriptionpb.GetSubscriptionListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription/get-item-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.Subscription.GetSubscriptionItemPageData, &subscriptionpb.GetSubscriptionItemPageDataRequest{}),
		})
	}

	// Balance Attribute module routes
	if subscriptionUseCases.BalanceAttribute != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/balance-attribute/create",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.BalanceAttribute.CreateBalanceAttribute, &balanceattributepb.CreateBalanceAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/balance-attribute/read",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.BalanceAttribute.ReadBalanceAttribute, &balanceattributepb.ReadBalanceAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/balance-attribute/update",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.BalanceAttribute.UpdateBalanceAttribute, &balanceattributepb.UpdateBalanceAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/balance-attribute/delete",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.BalanceAttribute.DeleteBalanceAttribute, &balanceattributepb.DeleteBalanceAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/balance-attribute/list",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.BalanceAttribute.ListBalanceAttributes, &balanceattributepb.ListBalanceAttributesRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/balance-attribute/get-list-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.BalanceAttribute.GetBalanceAttributeListPageData, &balanceattributepb.GetBalanceAttributeListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/balance-attribute/get-item-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.BalanceAttribute.GetBalanceAttributeItemPageData, &balanceattributepb.GetBalanceAttributeItemPageDataRequest{}),
		})
	}

	// Invoice Attribute module routes
	if subscriptionUseCases.InvoiceAttribute != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/invoice-attribute/create",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.InvoiceAttribute.CreateInvoiceAttribute, &invoiceattributepb.CreateInvoiceAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/invoice-attribute/read",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.InvoiceAttribute.ReadInvoiceAttribute, &invoiceattributepb.ReadInvoiceAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/invoice-attribute/update",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.InvoiceAttribute.UpdateInvoiceAttribute, &invoiceattributepb.UpdateInvoiceAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/invoice-attribute/delete",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.InvoiceAttribute.DeleteInvoiceAttribute, &invoiceattributepb.DeleteInvoiceAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/invoice-attribute/list",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.InvoiceAttribute.ListInvoiceAttributes, &invoiceattributepb.ListInvoiceAttributesRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/invoice-attribute/get-list-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.InvoiceAttribute.GetInvoiceAttributeListPageData, &invoiceattributepb.GetInvoiceAttributeListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/invoice-attribute/get-item-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.InvoiceAttribute.GetInvoiceAttributeItemPageData, &invoiceattributepb.GetInvoiceAttributeItemPageDataRequest{}),
		})
	}

	// Plan Attribute module routes
	if subscriptionUseCases.PlanAttribute != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/plan-attribute/create",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PlanAttribute.CreatePlanAttribute, &planattributepb.CreatePlanAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/plan-attribute/read",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PlanAttribute.ReadPlanAttribute, &planattributepb.ReadPlanAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/plan-attribute/update",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PlanAttribute.UpdatePlanAttribute, &planattributepb.UpdatePlanAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/plan-attribute/delete",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PlanAttribute.DeletePlanAttribute, &planattributepb.DeletePlanAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/plan-attribute/list",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PlanAttribute.ListPlanAttributes, &planattributepb.ListPlanAttributesRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/plan-attribute/get-list-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PlanAttribute.GetPlanAttributeListPageData, &planattributepb.GetPlanAttributeListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/plan-attribute/get-item-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PlanAttribute.GetPlanAttributeItemPageData, &planattributepb.GetPlanAttributeItemPageDataRequest{}),
		})
	}

	// Subscription Attribute module routes
	if subscriptionUseCases.SubscriptionAttribute != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-attribute/create",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionAttribute.CreateSubscriptionAttribute, &subscriptionattributepb.CreateSubscriptionAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-attribute/read",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionAttribute.ReadSubscriptionAttribute, &subscriptionattributepb.ReadSubscriptionAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-attribute/update",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionAttribute.UpdateSubscriptionAttribute, &subscriptionattributepb.UpdateSubscriptionAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-attribute/delete",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionAttribute.DeleteSubscriptionAttribute, &subscriptionattributepb.DeleteSubscriptionAttributeRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-attribute/list",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionAttribute.ListSubscriptionAttributes, &subscriptionattributepb.ListSubscriptionAttributesRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-attribute/get-list-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionAttribute.GetSubscriptionAttributeListPageData, &subscriptionattributepb.GetSubscriptionAttributeListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-attribute/get-item-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionAttribute.GetSubscriptionAttributeItemPageData, &subscriptionattributepb.GetSubscriptionAttributeItemPageDataRequest{}),
		})
	}

	return contracts.DomainRouteConfiguration{
		Domain:  "subscription",
		Prefix:  "/subscription",
		Enabled: true,
		Routes:  routes,
	}
}
