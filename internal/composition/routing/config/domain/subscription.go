package domain

import (
	"fmt"

	subscriptionuc "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
	balanceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance_attribute"
	invoicepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice"
	invoiceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice_attribute"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	planattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_attribute"
	plansettingspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_settings"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	pricescheduleworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule_workspace_user"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	subscriptionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_attribute"
	subscriptiongrouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
	subscriptiongroupmemberpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_member"
	subscriptiongroupproductplanstaffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
	subscriptiongroupworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_workspace_user"
	subscriptionseatpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_seat"
	subscriptionworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_workspace_user"
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

	// Product Price Plan module routes
	if subscriptionUseCases.ProductPricePlan != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/product-price-plan/create",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.ProductPricePlan.CreateProductPricePlan, &productpriceplanpb.CreateProductPricePlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/product-price-plan/read",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.ProductPricePlan.ReadProductPricePlan, &productpriceplanpb.ReadProductPricePlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/product-price-plan/update",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.ProductPricePlan.UpdateProductPricePlan, &productpriceplanpb.UpdateProductPricePlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/product-price-plan/delete",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.ProductPricePlan.DeleteProductPricePlan, &productpriceplanpb.DeleteProductPricePlanRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/product-price-plan/list",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.ProductPricePlan.ListProductPricePlans, &productpriceplanpb.ListProductPricePlansRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/product-price-plan/get-list-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.ProductPricePlan.GetProductPricePlanListPageData, &productpriceplanpb.GetProductPricePlanListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/product-price-plan/get-item-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.ProductPricePlan.GetProductPricePlanItemPageData, &productpriceplanpb.GetProductPricePlanItemPageDataRequest{}),
		})
	}

	// Price Schedule module routes
	if subscriptionUseCases.PriceSchedule != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/price-schedule/create",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PriceSchedule.CreatePriceSchedule, &priceschedulepb.CreatePriceScheduleRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/price-schedule/read",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PriceSchedule.ReadPriceSchedule, &priceschedulepb.ReadPriceScheduleRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/price-schedule/update",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PriceSchedule.UpdatePriceSchedule, &priceschedulepb.UpdatePriceScheduleRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/price-schedule/delete",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PriceSchedule.DeletePriceSchedule, &priceschedulepb.DeletePriceScheduleRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/price-schedule/list",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PriceSchedule.ListPriceSchedules, &priceschedulepb.ListPriceSchedulesRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/price-schedule/get-list-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PriceSchedule.GetPriceScheduleListPageData, &priceschedulepb.GetPriceScheduleListPageDataRequest{}),
		})

		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/price-schedule/get-item-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PriceSchedule.GetPriceScheduleItemPageData, &priceschedulepb.GetPriceScheduleItemPageDataRequest{}),
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

	// Subscription Seat module routes (outsourcing vertical).
	// CRUD + page-data go through generic proto handlers. The SR-2 lifecycle ops
	// (replace, set-status) take Go-shaped (non-proto) requests and are invoked
	// directly from the view/action layer via the use-case aggregate, so they are
	// intentionally NOT exposed as generic JSON routes here.
	if subscriptionUseCases.SubscriptionSeat != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-seat/create",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionSeat.CreateSubscriptionSeat, &subscriptionseatpb.CreateSubscriptionSeatRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-seat/read",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionSeat.ReadSubscriptionSeat, &subscriptionseatpb.ReadSubscriptionSeatRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-seat/update",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionSeat.UpdateSubscriptionSeat, &subscriptionseatpb.UpdateSubscriptionSeatRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-seat/delete",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionSeat.DeleteSubscriptionSeat, &subscriptionseatpb.DeleteSubscriptionSeatRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-seat/list",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionSeat.ListSubscriptionSeats, &subscriptionseatpb.ListSubscriptionSeatsRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-seat/get-list-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionSeat.GetSubscriptionSeatListPageData, &subscriptionseatpb.GetSubscriptionSeatListPageDataRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-seat/get-item-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionSeat.GetSubscriptionSeatItemPageData, &subscriptionseatpb.GetSubscriptionSeatItemPageDataRequest{}),
		})
	}

	// Subscription Workspace User module routes (servicing membership).
	if subscriptionUseCases.SubscriptionWorkspaceUser != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-workspace-user/create",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionWorkspaceUser.CreateSubscriptionWorkspaceUser, &subscriptionworkspaceuserpb.CreateSubscriptionWorkspaceUserRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-workspace-user/read",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionWorkspaceUser.ReadSubscriptionWorkspaceUser, &subscriptionworkspaceuserpb.ReadSubscriptionWorkspaceUserRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-workspace-user/update",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionWorkspaceUser.UpdateSubscriptionWorkspaceUser, &subscriptionworkspaceuserpb.UpdateSubscriptionWorkspaceUserRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-workspace-user/delete",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionWorkspaceUser.DeleteSubscriptionWorkspaceUser, &subscriptionworkspaceuserpb.DeleteSubscriptionWorkspaceUserRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-workspace-user/list",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionWorkspaceUser.ListSubscriptionWorkspaceUsers, &subscriptionworkspaceuserpb.ListSubscriptionWorkspaceUsersRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-workspace-user/get-list-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionWorkspaceUser.GetSubscriptionWorkspaceUserListPageData, &subscriptionworkspaceuserpb.GetSubscriptionWorkspaceUserListPageDataRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-workspace-user/get-item-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionWorkspaceUser.GetSubscriptionWorkspaceUserItemPageData, &subscriptionworkspaceuserpb.GetSubscriptionWorkspaceUserItemPageDataRequest{}),
		})
	}

	// Subscription Group module routes.
	if subscriptionUseCases.SubscriptionGroup != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group/create",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroup.CreateSubscriptionGroup, &subscriptiongrouppb.CreateSubscriptionGroupRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group/read",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroup.ReadSubscriptionGroup, &subscriptiongrouppb.ReadSubscriptionGroupRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group/update",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroup.UpdateSubscriptionGroup, &subscriptiongrouppb.UpdateSubscriptionGroupRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group/delete",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroup.DeleteSubscriptionGroup, &subscriptiongrouppb.DeleteSubscriptionGroupRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group/list",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroup.ListSubscriptionGroups, &subscriptiongrouppb.ListSubscriptionGroupsRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group/get-list-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroup.GetSubscriptionGroupListPageData, &subscriptiongrouppb.GetSubscriptionGroupListPageDataRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group/get-item-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroup.GetSubscriptionGroupItemPageData, &subscriptiongrouppb.GetSubscriptionGroupItemPageDataRequest{}),
		})
	}

	// Subscription Group Member module routes.
	if subscriptionUseCases.SubscriptionGroupMember != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group-member/create",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroupMember.CreateSubscriptionGroupMember, &subscriptiongroupmemberpb.CreateSubscriptionGroupMemberRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group-member/read",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroupMember.ReadSubscriptionGroupMember, &subscriptiongroupmemberpb.ReadSubscriptionGroupMemberRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group-member/update",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroupMember.UpdateSubscriptionGroupMember, &subscriptiongroupmemberpb.UpdateSubscriptionGroupMemberRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group-member/delete",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroupMember.DeleteSubscriptionGroupMember, &subscriptiongroupmemberpb.DeleteSubscriptionGroupMemberRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group-member/list",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroupMember.ListSubscriptionGroupMembers, &subscriptiongroupmemberpb.ListSubscriptionGroupMembersRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group-member/get-list-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroupMember.GetSubscriptionGroupMemberListPageData, &subscriptiongroupmemberpb.GetSubscriptionGroupMemberListPageDataRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group-member/get-item-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroupMember.GetSubscriptionGroupMemberItemPageData, &subscriptiongroupmemberpb.GetSubscriptionGroupMemberItemPageDataRequest{}),
		})
	}

	// Subscription Group Workspace User module routes.
	if subscriptionUseCases.SubscriptionGroupWorkspaceUser != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group-workspace-user/create",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroupWorkspaceUser.CreateSubscriptionGroupWorkspaceUser, &subscriptiongroupworkspaceuserpb.CreateSubscriptionGroupWorkspaceUserRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group-workspace-user/read",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroupWorkspaceUser.ReadSubscriptionGroupWorkspaceUser, &subscriptiongroupworkspaceuserpb.ReadSubscriptionGroupWorkspaceUserRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group-workspace-user/update",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroupWorkspaceUser.UpdateSubscriptionGroupWorkspaceUser, &subscriptiongroupworkspaceuserpb.UpdateSubscriptionGroupWorkspaceUserRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group-workspace-user/delete",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroupWorkspaceUser.DeleteSubscriptionGroupWorkspaceUser, &subscriptiongroupworkspaceuserpb.DeleteSubscriptionGroupWorkspaceUserRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group-workspace-user/list",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroupWorkspaceUser.ListSubscriptionGroupWorkspaceUsers, &subscriptiongroupworkspaceuserpb.ListSubscriptionGroupWorkspaceUsersRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group-workspace-user/get-list-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroupWorkspaceUser.GetSubscriptionGroupWorkspaceUserListPageData, &subscriptiongroupworkspaceuserpb.GetSubscriptionGroupWorkspaceUserListPageDataRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group-workspace-user/get-item-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroupWorkspaceUser.GetSubscriptionGroupWorkspaceUserItemPageData, &subscriptiongroupworkspaceuserpb.GetSubscriptionGroupWorkspaceUserItemPageDataRequest{}),
		})
	}

	// Subscription Group Product Plan Staff module routes.
	if subscriptionUseCases.SubscriptionGroupProductPlanStaff != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group-product-plan-staff/create",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroupProductPlanStaff.CreateSubscriptionGroupProductPlanStaff, &subscriptiongroupproductplanstaffpb.CreateSubscriptionGroupProductPlanStaffRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group-product-plan-staff/read",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroupProductPlanStaff.ReadSubscriptionGroupProductPlanStaff, &subscriptiongroupproductplanstaffpb.ReadSubscriptionGroupProductPlanStaffRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group-product-plan-staff/update",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroupProductPlanStaff.UpdateSubscriptionGroupProductPlanStaff, &subscriptiongroupproductplanstaffpb.UpdateSubscriptionGroupProductPlanStaffRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group-product-plan-staff/delete",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroupProductPlanStaff.DeleteSubscriptionGroupProductPlanStaff, &subscriptiongroupproductplanstaffpb.DeleteSubscriptionGroupProductPlanStaffRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group-product-plan-staff/list",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroupProductPlanStaff.ListSubscriptionGroupProductPlanStaffs, &subscriptiongroupproductplanstaffpb.ListSubscriptionGroupProductPlanStaffsRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group-product-plan-staff/get-list-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroupProductPlanStaff.GetSubscriptionGroupProductPlanStaffListPageData, &subscriptiongroupproductplanstaffpb.GetSubscriptionGroupProductPlanStaffListPageDataRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/subscription-group-product-plan-staff/get-item-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.SubscriptionGroupProductPlanStaff.GetSubscriptionGroupProductPlanStaffItemPageData, &subscriptiongroupproductplanstaffpb.GetSubscriptionGroupProductPlanStaffItemPageDataRequest{}),
		})
	}

	// Price Schedule Workspace User module routes.
	if subscriptionUseCases.PriceScheduleWorkspaceUser != nil {
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/price-schedule-workspace-user/create",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PriceScheduleWorkspaceUser.CreatePriceScheduleWorkspaceUser, &pricescheduleworkspaceuserpb.CreatePriceScheduleWorkspaceUserRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/price-schedule-workspace-user/read",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PriceScheduleWorkspaceUser.ReadPriceScheduleWorkspaceUser, &pricescheduleworkspaceuserpb.ReadPriceScheduleWorkspaceUserRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/price-schedule-workspace-user/update",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PriceScheduleWorkspaceUser.UpdatePriceScheduleWorkspaceUser, &pricescheduleworkspaceuserpb.UpdatePriceScheduleWorkspaceUserRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/price-schedule-workspace-user/delete",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PriceScheduleWorkspaceUser.DeletePriceScheduleWorkspaceUser, &pricescheduleworkspaceuserpb.DeletePriceScheduleWorkspaceUserRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/price-schedule-workspace-user/list",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PriceScheduleWorkspaceUser.ListPriceScheduleWorkspaceUsers, &pricescheduleworkspaceuserpb.ListPriceScheduleWorkspaceUsersRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/price-schedule-workspace-user/get-list-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PriceScheduleWorkspaceUser.GetPriceScheduleWorkspaceUserListPageData, &pricescheduleworkspaceuserpb.GetPriceScheduleWorkspaceUserListPageDataRequest{}),
		})
		routes = append(routes, contracts.RouteConfiguration{
			Method:  "POST",
			Path:    "/api/subscription/price-schedule-workspace-user/get-item-page-data",
			Handler: contracts.NewGenericHandler(subscriptionUseCases.PriceScheduleWorkspaceUser.GetPriceScheduleWorkspaceUserItemPageData, &pricescheduleworkspaceuserpb.GetPriceScheduleWorkspaceUserItemPageDataRequest{}),
		})
	}

	return contracts.DomainRouteConfiguration{
		Domain:  "subscription",
		Prefix:  "/subscription",
		Enabled: true,
		Routes:  routes,
	}
}
