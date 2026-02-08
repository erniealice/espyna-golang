package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases"
	"github.com/erniealice/espyna-golang/internal/orchestration/workflow/executor"
)

// RegisterSubscriptionUseCases registers all subscription domain use cases with the registry.
// Subscription domain includes: Balance, Invoice, Plan, PlanSettings, PricePlan, Subscription
// and their associated attribute entities.
func RegisterSubscriptionUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Subscription == nil {
		return
	}

	registerBalanceUseCases(useCases, register)
	registerInvoiceUseCases(useCases, register)
	registerPlanUseCases(useCases, register)
	registerPlanSettingsUseCases(useCases, register)
	registerPricePlanUseCases(useCases, register)
	registerSubscriptionCoreUseCases(useCases, register)
	registerSubscriptionAttributeUseCases(useCases, register)
}

func registerBalanceUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Subscription.Balance == nil {
		return
	}
	if useCases.Subscription.Balance.CreateBalance != nil {
		register("subscription.balance.create", executor.New(useCases.Subscription.Balance.CreateBalance.Execute))
	}
	if useCases.Subscription.Balance.ReadBalance != nil {
		register("subscription.balance.read", executor.New(useCases.Subscription.Balance.ReadBalance.Execute))
	}
	if useCases.Subscription.Balance.UpdateBalance != nil {
		register("subscription.balance.update", executor.New(useCases.Subscription.Balance.UpdateBalance.Execute))
	}
	if useCases.Subscription.Balance.DeleteBalance != nil {
		register("subscription.balance.delete", executor.New(useCases.Subscription.Balance.DeleteBalance.Execute))
	}
	if useCases.Subscription.Balance.ListBalances != nil {
		register("subscription.balance.list", executor.New(useCases.Subscription.Balance.ListBalances.Execute))
	}
}

func registerInvoiceUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Subscription.Invoice == nil {
		return
	}
	if useCases.Subscription.Invoice.CreateInvoice != nil {
		register("subscription.invoice.create", executor.New(useCases.Subscription.Invoice.CreateInvoice.Execute))
	}
	if useCases.Subscription.Invoice.ReadInvoice != nil {
		register("subscription.invoice.read", executor.New(useCases.Subscription.Invoice.ReadInvoice.Execute))
	}
	if useCases.Subscription.Invoice.UpdateInvoice != nil {
		register("subscription.invoice.update", executor.New(useCases.Subscription.Invoice.UpdateInvoice.Execute))
	}
	if useCases.Subscription.Invoice.DeleteInvoice != nil {
		register("subscription.invoice.delete", executor.New(useCases.Subscription.Invoice.DeleteInvoice.Execute))
	}
	if useCases.Subscription.Invoice.ListInvoices != nil {
		register("subscription.invoice.list", executor.New(useCases.Subscription.Invoice.ListInvoices.Execute))
	}
}

func registerPlanUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Subscription.Plan == nil {
		return
	}
	if useCases.Subscription.Plan.CreatePlan != nil {
		register("subscription.plan.create", executor.New(useCases.Subscription.Plan.CreatePlan.Execute))
	}
	if useCases.Subscription.Plan.ReadPlan != nil {
		register("subscription.plan.read", executor.New(useCases.Subscription.Plan.ReadPlan.Execute))
	}
	if useCases.Subscription.Plan.UpdatePlan != nil {
		register("subscription.plan.update", executor.New(useCases.Subscription.Plan.UpdatePlan.Execute))
	}
	if useCases.Subscription.Plan.DeletePlan != nil {
		register("subscription.plan.delete", executor.New(useCases.Subscription.Plan.DeletePlan.Execute))
	}
	if useCases.Subscription.Plan.ListPlans != nil {
		register("subscription.plan.list", executor.New(useCases.Subscription.Plan.ListPlans.Execute))
	}
}

func registerPlanSettingsUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Subscription.PlanSettings == nil {
		return
	}
	if useCases.Subscription.PlanSettings.CreatePlanSettings != nil {
		register("subscription.plan_settings.create", executor.New(useCases.Subscription.PlanSettings.CreatePlanSettings.Execute))
	}
	if useCases.Subscription.PlanSettings.ReadPlanSettings != nil {
		register("subscription.plan_settings.read", executor.New(useCases.Subscription.PlanSettings.ReadPlanSettings.Execute))
	}
	if useCases.Subscription.PlanSettings.UpdatePlanSettings != nil {
		register("subscription.plan_settings.update", executor.New(useCases.Subscription.PlanSettings.UpdatePlanSettings.Execute))
	}
	if useCases.Subscription.PlanSettings.DeletePlanSettings != nil {
		register("subscription.plan_settings.delete", executor.New(useCases.Subscription.PlanSettings.DeletePlanSettings.Execute))
	}
	if useCases.Subscription.PlanSettings.ListPlanSettings != nil {
		register("subscription.plan_settings.list", executor.New(useCases.Subscription.PlanSettings.ListPlanSettings.Execute))
	}
}

func registerPricePlanUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Subscription.PricePlan == nil {
		return
	}
	if useCases.Subscription.PricePlan.CreatePricePlan != nil {
		register("subscription.price_plan.create", executor.New(useCases.Subscription.PricePlan.CreatePricePlan.Execute))
	}
	if useCases.Subscription.PricePlan.ReadPricePlan != nil {
		register("subscription.price_plan.read", executor.New(useCases.Subscription.PricePlan.ReadPricePlan.Execute))
	}
	if useCases.Subscription.PricePlan.UpdatePricePlan != nil {
		register("subscription.price_plan.update", executor.New(useCases.Subscription.PricePlan.UpdatePricePlan.Execute))
	}
	if useCases.Subscription.PricePlan.DeletePricePlan != nil {
		register("subscription.price_plan.delete", executor.New(useCases.Subscription.PricePlan.DeletePricePlan.Execute))
	}
	if useCases.Subscription.PricePlan.ListPricePlans != nil {
		register("subscription.price_plan.list", executor.New(useCases.Subscription.PricePlan.ListPricePlans.Execute))
	}
}

func registerSubscriptionCoreUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Subscription.Subscription == nil {
		return
	}
	if useCases.Subscription.Subscription.CreateSubscription != nil {
		register("subscription.subscription.create", executor.New(useCases.Subscription.Subscription.CreateSubscription.Execute))
	}
	if useCases.Subscription.Subscription.ReadSubscription != nil {
		register("subscription.subscription.read", executor.New(useCases.Subscription.Subscription.ReadSubscription.Execute))
	}
	if useCases.Subscription.Subscription.UpdateSubscription != nil {
		register("subscription.subscription.update", executor.New(useCases.Subscription.Subscription.UpdateSubscription.Execute))
	}
	if useCases.Subscription.Subscription.DeleteSubscription != nil {
		register("subscription.subscription.delete", executor.New(useCases.Subscription.Subscription.DeleteSubscription.Execute))
	}
	if useCases.Subscription.Subscription.ListSubscriptions != nil {
		register("subscription.subscription.list", executor.New(useCases.Subscription.Subscription.ListSubscriptions.Execute))
	}
}

func registerSubscriptionAttributeUseCases(useCases *usecases.Aggregate, register func(string, ports.ActivityExecutor)) {
	if useCases.Subscription.SubscriptionAttribute == nil {
		return
	}
	if useCases.Subscription.SubscriptionAttribute.CreateSubscriptionAttribute != nil {
		register("subscription.subscription_attribute.create", executor.New(useCases.Subscription.SubscriptionAttribute.CreateSubscriptionAttribute.Execute))
	}
	if useCases.Subscription.SubscriptionAttribute.CreateSubscriptionAttributesByCode != nil {
		register("subscription.subscription_attribute.create_by_code", executor.New(useCases.Subscription.SubscriptionAttribute.CreateSubscriptionAttributesByCode.Execute))
	}
	if useCases.Subscription.SubscriptionAttribute.ReadSubscriptionAttribute != nil {
		register("subscription.subscription_attribute.read", executor.New(useCases.Subscription.SubscriptionAttribute.ReadSubscriptionAttribute.Execute))
	}
	if useCases.Subscription.SubscriptionAttribute.UpdateSubscriptionAttribute != nil {
		register("subscription.subscription_attribute.update", executor.New(useCases.Subscription.SubscriptionAttribute.UpdateSubscriptionAttribute.Execute))
	}
	if useCases.Subscription.SubscriptionAttribute.DeleteSubscriptionAttribute != nil {
		register("subscription.subscription_attribute.delete", executor.New(useCases.Subscription.SubscriptionAttribute.DeleteSubscriptionAttribute.Execute))
	}
	if useCases.Subscription.SubscriptionAttribute.ListSubscriptionAttributes != nil {
		register("subscription.subscription_attribute.list", executor.New(useCases.Subscription.SubscriptionAttribute.ListSubscriptionAttributes.Execute))
	}
}
